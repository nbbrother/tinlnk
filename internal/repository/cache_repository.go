package repository

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	KeyPrefixURL   = "url:"
	KeyPrefixPV    = "pv:"
	KeyPrefixUV    = "uv:"
	KeyPrefixDayPV = "day:pv:"
	KeyPrefixDayUV = "day:uv:"
	KeyPrefixLock  = "lock:"
)

type CacheRepository struct {
	rdb *redis.Client
}

func NewCacheRepository(rdb *redis.Client) *CacheRepository {
	return &CacheRepository{rdb: rdb}
}

// ========== URL缓存 ==========

func (r *CacheRepository) SetURL(ctx context.Context, shortCode, longURL string, ttl time.Duration) error {
	return r.rdb.Set(ctx, KeyPrefixURL+shortCode, longURL, ttl).Err()
}

func (r *CacheRepository) GetURL(ctx context.Context, shortCode string) (string, error) {
	return r.rdb.Get(ctx, KeyPrefixURL+shortCode).Result()
}

func (r *CacheRepository) DeleteURL(ctx context.Context, shortCode string) error {
	return r.rdb.Del(ctx, KeyPrefixURL+shortCode).Err()
}

// ========== 统计相关 ==========

func (r *CacheRepository) IncrPV(ctx context.Context, shortCode string) error {
	pipe := r.rdb.Pipeline()

	// 总PV
	pipe.Incr(ctx, KeyPrefixPV+shortCode)

	// 当日PV
	today := time.Now().Format("20060102")
	dayKey := KeyPrefixDayPV + shortCode + ":" + today
	pipe.Incr(ctx, dayKey)
	pipe.Expire(ctx, dayKey, 48*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *CacheRepository) AddUV(ctx context.Context, shortCode, ip string) error {
	pipe := r.rdb.Pipeline()

	// 总UV (HyperLogLog)
	pipe.PFAdd(ctx, KeyPrefixUV+shortCode, ip)

	// 当日UV
	today := time.Now().Format("20060102")
	dayKey := KeyPrefixDayUV + shortCode + ":" + today
	pipe.PFAdd(ctx, dayKey, ip)
	pipe.Expire(ctx, dayKey, 48*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *CacheRepository) GetPV(ctx context.Context, shortCode string) (int64, error) {
	return r.rdb.Get(ctx, KeyPrefixPV+shortCode).Int64()
}

func (r *CacheRepository) GetUV(ctx context.Context, shortCode string) (int64, error) {
	return r.rdb.PFCount(ctx, KeyPrefixUV+shortCode).Result()
}

func (r *CacheRepository) GetTodayPV(ctx context.Context, shortCode string) (int64, error) {
	today := time.Now().Format("20060102")
	return r.rdb.Get(ctx, KeyPrefixDayPV+shortCode+":"+today).Int64()
}

func (r *CacheRepository) GetTodayUV(ctx context.Context, shortCode string) (int64, error) {
	today := time.Now().Format("20060102")
	return r.rdb.PFCount(ctx, KeyPrefixDayUV+shortCode+":"+today).Result()
}

func (r *CacheRepository) GetDayPV(ctx context.Context, shortCode, date string) (int64, error) {
	return r.rdb.Get(ctx, KeyPrefixDayPV+shortCode+":"+date).Int64()
}

func (r *CacheRepository) GetDayUV(ctx context.Context, shortCode, date string) (int64, error) {
	return r.rdb.PFCount(ctx, KeyPrefixDayUV+shortCode+":"+date).Result()
}

// ========== 分布式锁 ==========

func (r *CacheRepository) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, KeyPrefixLock+key, "1", ttl).Result()
}

func (r *CacheRepository) ReleaseLock(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, KeyPrefixLock+key).Err()
}
