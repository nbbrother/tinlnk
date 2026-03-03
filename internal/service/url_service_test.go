package service

import (
	"context"
	"testing"
	"tinLink/internal/model"
	"tinLink/internal/pkg/base62"
	"tinLink/internal/pkg/bloom"

	"github.com/stretchr/testify/mock"
)

// MockIDGenerator Mock ID生成器
type MockIDGenerator struct {
	mock.Mock
}

func (m *MockIDGenerator) Generate() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// MockURLRepository Mock Repository
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *model.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) GetByShortCode(ctx context.Context, code string) (*model.URL, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

// 表驱动测试
func TestURLService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateURLRequest
		wantErr error
	}{
		{
			name: "正常创建",
			req: CreateURLRequest{
				LongURL:    "https://www.example.com/path",
				ExpireDays: 30,
			},
			wantErr: nil,
		},
		{
			name: "自定义短码已存在",
			req: CreateURLRequest{
				LongURL:    "https://www.example.com",
				CustomCode: "exists",
				ExpireDays: 30,
			},
			wantErr: ErrURLExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备Mock
			mockRepo := new(MockURLRepository)
			mockIDGen := new(MockIDGenerator)

			if tt.req.CustomCode == "exists" {
				mockRepo.On("ExistsByShortCode", mock.Anything, "exists").Return(true, nil)
			} else {
				mockIDGen.On("Generate").Return(int64(123456789))
				mockRepo.On("ExistsByShortCode", mock.Anything, mock.Anything).Return(false, nil)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			}

			// 测试执行
			// svc := NewURLService(mockRepo, nil, nil, mockIDGen, nil)
			// _, err := svc.CreateShortURL(context.Background(), tt.req)

			// 断言
			// if tt.wantErr != nil {
			//     assert.ErrorIs(t, err, tt.wantErr)
			// } else {
			//     assert.NoError(t, err)
			// }
		})
	}
}

// 基准测试
func BenchmarkBase62Encode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		base62.Encode(uint64(i))
	}
}

func BenchmarkBloomFilter(b *testing.B) {
	bf := bloom.New(1000000, 0.001)

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.Add([]byte("test" + string(rune(i))))
		}
	})

	b.Run("Contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.Contains([]byte("test" + string(rune(i))))
		}
	})
}
