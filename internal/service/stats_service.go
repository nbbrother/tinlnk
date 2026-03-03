package service

import (
	"context"
	"time"
	"tinLink/internal/repository"
)

type Stats struct {
	TotalPV    int64
	TotalUV    int64
	TodayPV    int64
	TodayUV    int64
	CreatedAt  time.Time
	LastAccess time.Time
}

type DailyStats struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

type StatsService struct {
	urlRepo   *repository.URLRepository
	cacheRepo *repository.CacheRepository
}

func NewStatsService(
	urlRepo *repository.URLRepository,
	cacheRepo *repository.CacheRepository,
) *StatsService {
	return &StatsService{urlRepo: urlRepo, cacheRepo: cacheRepo}
}

func (s *StatsService) GetStats(ctx context.Context, shortCode string) (*Stats, error) {
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, ErrURLNotFound
	}

	totalPV, _ := s.cacheRepo.GetPV(ctx, shortCode)
	totalUV, _ := s.cacheRepo.GetUV(ctx, shortCode)
	todayPV, _ := s.cacheRepo.GetTodayPV(ctx, shortCode)
	todayUV, _ := s.cacheRepo.GetTodayUV(ctx, shortCode)

	return &Stats{
		TotalPV:    totalPV,
		TotalUV:    totalUV,
		TodayPV:    todayPV,
		TodayUV:    todayUV,
		CreatedAt:  url.CreatedAt,
		LastAccess: time.Now(),
	}, nil
}

func (s *StatsService) GetDailyStats(ctx context.Context, shortCode string, days int) ([]DailyStats, error) {
	result := make([]DailyStats, 0, days)

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("20060102")
		pv, _ := s.cacheRepo.GetDayPV(ctx, shortCode, date)
		uv, _ := s.cacheRepo.GetDayUV(ctx, shortCode, date)
		result = append(result, DailyStats{
			Date: date,
			PV:   pv,
			UV:   uv,
		})
	}

	return result, nil
}
