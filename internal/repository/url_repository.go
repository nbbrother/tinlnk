package repository

import (
	"context"
	"hash/fnv"
	"tinLink/internal/model"

	"gorm.io/gorm"
)

const TableCount = 64

type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) getTableName(shortCode string) string {
	h := fnv.New32a()
	h.Write([]byte(shortCode))
	tableIndex := h.Sum32() % TableCount
	return model.URL{}.TableName(int(tableIndex))
}

func (r *URLRepository) Create(ctx context.Context, url *model.URL) error {
	tableName := r.getTableName(url.ShortCode)
	return r.db.WithContext(ctx).Table(tableName).Create(url).Error
}

func (r *URLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	tableName := r.getTableName(shortCode)
	var url model.URL
	err := r.db.WithContext(ctx).Table(tableName).
		Where("short_code = ?", shortCode).First(&url).Error
	if err != nil {
		return nil, err
	}
	return &url, nil
}

// GetByLongURL 扫描所有表查找longURL（仅用于去重，可接受性能损失）
func (r *URLRepository) GetByLongURL(ctx context.Context, longURL string) (*model.URL, error) {
	for i := 0; i < TableCount; i++ {
		tableName := model.URL{}.TableName(i)
		var url model.URL
		err := r.db.WithContext(ctx).Table(tableName).
			Where("long_url = ?", longURL).First(&url).Error
		if err == nil {
			return &url, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *URLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	tableName := r.getTableName(shortCode)
	var count int64
	err := r.db.WithContext(ctx).Table(tableName).
		Where("short_code = ?", shortCode).Count(&count).Error
	return count > 0, err
}

func (r *URLRepository) DeleteByShortCode(ctx context.Context, shortCode string) error {
	tableName := r.getTableName(shortCode)
	return r.db.WithContext(ctx).Table(tableName).
		Where("short_code = ?", shortCode).Delete(&model.URL{}).Error
}

func (r *URLRepository) UpdateAccessCount(ctx context.Context, shortCode string) error {
	tableName := r.getTableName(shortCode)
	return r.db.WithContext(ctx).Table(tableName).
		Where("short_code = ?", shortCode).
		UpdateColumn("access_count", gorm.Expr("access_count + 1")).Error
}

// DB 返回底层的 *gorm.DB 实例，用于特殊查询（如预热布隆过滤器）
func (r *URLRepository) DB() *gorm.DB {
	return r.db
}
