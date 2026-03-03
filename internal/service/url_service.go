package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/singleflight"

	"tinLink/internal/model"
	"tinLink/internal/pkg/base62"
	"tinLink/internal/pkg/bloom"
	"tinLink/internal/pkg/circuitbreaker"
	"tinLink/internal/repository"
)

var tracer = otel.Tracer("service")

var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExpired  = errors.New("url expired")
	ErrURLExists   = errors.New("url already exists")
	ErrInvalidURL  = errors.New("invalid url")
	ErrInvalidCode = errors.New("invalid custom code")
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type IDGenerator interface {
	Generate() int64
}

type URLService struct {
	urlRepo     *repository.URLRepository
	cacheRepo   *repository.CacheRepository
	localCache  *repository.LocalCache
	idGenerator IDGenerator
	bloomFilter *bloom.Filter
	cb          *circuitbreaker.CircuitBreaker
	sfGroup     singleflight.Group
	hotSpot     *HotSpotDetector
	mu          sync.RWMutex
}

func NewURLService(
	urlRepo *repository.URLRepository,
	cacheRepo *repository.CacheRepository,
	localCache *repository.LocalCache,
	idGenerator IDGenerator,
	cb *circuitbreaker.CircuitBreaker,
) *URLService {
	// 布隆过滤器：1亿容量，0.01%误判率（约230MB内存，可根据实际调整）
	bf := bloom.New(10000000, 0.001) // 减小为1000万容量，内存约23MB

	s := &URLService{
		urlRepo:     urlRepo,
		cacheRepo:   cacheRepo,
		localCache:  localCache,
		idGenerator: idGenerator,
		bloomFilter: bf,
		cb:          cb,
		hotSpot:     NewHotSpotDetector(1000, time.Minute),
	}

	// 异步预热布隆过滤器（加载最近活跃的短码）
	go s.warmUpBloomFilter()

	// 启动热点数据预热
	go s.warmUpHotSpot()

	return s
}

// warmUpBloomFilter 预热布隆过滤器，加载所有未过期的短码
func (s *URLService) warmUpBloomFilter() {
	ctx := context.Background()
	batchSize := 10000 // 每批加载数量，可根据内存和性能调整
	totalLoaded := 0
	start := time.Now()

	for i := 0; i < repository.TableCount; i++ {
		tableName := model.URL{}.TableName(i)
		offset := 0
		for {
			var codes []string
			err := s.urlRepo.DB().WithContext(ctx).Table(tableName).
				Where("expire_at > ?", time.Now()).
				Select("short_code").
				Limit(batchSize).Offset(offset).
				Find(&codes).Error
			if err != nil {
				log.Printf("Failed to load bloom filter from table %s: %v", tableName, err)
				break
			}
			if len(codes) == 0 {
				break
			}
			for _, code := range codes {
				s.bloomFilter.Add([]byte(code))
			}
			totalLoaded += len(codes)
			offset += len(codes)
			if len(codes) < batchSize {
				break // 已加载完该表所有数据
			}
		}
	}

	log.Printf("Bloom filter warm-up completed. Loaded %d short codes in %v", totalLoaded, time.Since(start))
}

type CreateURLRequest struct {
	LongURL    string
	CustomCode string
	ExpireDays int
}

func (s *URLService) CreateShortURL(ctx context.Context, req CreateURLRequest) (*model.URL, error) {
	ctx, span := tracer.Start(ctx, "URLService.CreateShortURL")
	defer span.End()

	var shortCode string

	if req.CustomCode != "" {
		shortCode = req.CustomCode
		exists, err := s.urlRepo.ExistsByShortCode(ctx, shortCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrURLExists
		}
	} else {
		// 检查URL是否已存在（布隆过滤器快速判断）
		if s.bloomFilter.Contains([]byte(req.LongURL)) {
			if existing, err := s.urlRepo.GetByLongURL(ctx, req.LongURL); err == nil && existing != nil {
				span.SetAttributes(attribute.Bool("cache_hit", true))
				return existing, nil
			}
		}
		id := s.idGenerator.Generate()
		shortCode = base62.Encode(uint64(id))
	}

	expireAt := time.Now().AddDate(0, 0, req.ExpireDays)
	url := &model.URL{
		ID:        s.idGenerator.Generate(),
		ShortCode: shortCode,
		LongURL:   req.LongURL,
		ExpireAt:  expireAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.urlRepo.Create(ctx, url); err != nil {
		span.RecordError(err)
		return nil, err
	}

	ttl := time.Until(expireAt)
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	s.cacheRepo.SetURL(ctx, shortCode, req.LongURL, ttl)

	s.bloomFilter.Add([]byte(req.LongURL))

	span.SetAttributes(attribute.String("short_code", shortCode))
	return url, nil
}

func (s *URLService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	ctx, span := tracer.Start(ctx, "URLService.GetLongURL")
	defer span.End()

	if !s.cb.Allow() {
		return "", ErrCircuitOpen
	}

	if longURL, ok := s.localCache.Get(shortCode); ok {
		span.SetAttributes(attribute.String("cache_level", "local"))
		s.hotSpot.Record(shortCode)
		s.cb.Success()
		return longURL, nil
	}

	result, err, _ := s.sfGroup.Do(shortCode, func() (interface{}, error) {
		if longURL, err := s.cacheRepo.GetURL(ctx, shortCode); err == nil && longURL != "" {
			span.SetAttributes(attribute.String("cache_level", "redis"))
			s.localCache.Set(shortCode, longURL)
			return longURL, nil
		}

		url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
		if err != nil {
			return "", ErrURLNotFound
		}

		if url.ExpireAt.Before(time.Now()) {
			go s.DeleteURL(context.Background(), shortCode) // 使用新context
			return "", ErrURLExpired
		}

		ttl := time.Until(url.ExpireAt)
		if ttl > 24*time.Hour {
			ttl = 24 * time.Hour
		}
		s.cacheRepo.SetURL(ctx, shortCode, url.LongURL, ttl)
		s.localCache.Set(shortCode, url.LongURL)

		span.SetAttributes(attribute.String("cache_level", "db"))
		return url.LongURL, nil
	})

	if err != nil {
		s.cb.Failure()
		return "", err
	}

	longURL, ok := result.(string)
	if !ok {
		return "", errors.New("unexpected type from singleflight")
	}

	s.cb.Success()
	s.hotSpot.Record(shortCode)
	return longURL, nil
}

func (s *URLService) GetURLDetail(ctx context.Context, shortCode string) (*model.URL, error) {
	return s.urlRepo.GetByShortCode(ctx, shortCode)
}

func (s *URLService) DeleteURL(ctx context.Context, shortCode string) error {
	if err := s.urlRepo.DeleteByShortCode(ctx, shortCode); err != nil {
		return err
	}
	s.cacheRepo.DeleteURL(ctx, shortCode)
	s.localCache.Delete(shortCode)
	return nil
}

func (s *URLService) RecordAccess(ctx context.Context, shortCode, ip, ua, referer string) {
	// 使用独立的后台context，避免原请求取消
	bgCtx := context.Background()
	s.cacheRepo.IncrPV(bgCtx, shortCode)
	s.cacheRepo.AddUV(bgCtx, shortCode, ip)
	// 可扩展写入消息队列等
}

func (s *URLService) warmUpHotSpot() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		hotKeys := s.hotSpot.GetTopK(100)
		for _, key := range hotKeys {
			if _, ok := s.localCache.Get(key); !ok {
				if longURL, err := s.cacheRepo.GetURL(context.Background(), key); err == nil {
					s.localCache.Set(key, longURL)
				}
			}
		}
	}
}
