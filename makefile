.PHONY: all build run test clean docker lint

APP_NAME := tinLink
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S)
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 默认目标
all: lint test build

# 编译
build:
	@echo "Building $(APP_NAME)..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/server

# 运行
run:
	@go run ./cmd/server

# 测试
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# 覆盖率报告
coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 基准测试
bench:
	@go test -bench=. -benchmem ./...

# 代码检查
lint:
	@golangci-lint run ./...

# 清理
clean:
	@rm -rf bin/ coverage.out coverage.html

# Docker构建
docker-build:
	@docker build -t $(APP_NAME):$(VERSION) .
	@docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Docker运行
docker-up:
	@docker-compose up -d

docker-down:
	@docker-compose down

docker-logs:
	@docker-compose logs -f tinlink

# 数据库迁移
migrate-up:
	@migrate -path migrations -database "mysql://root:root123@tcp(localhost:3306)/tinlink" up

migrate-down:
	@migrate -path migrations -database "mysql://root:root123@tcp(localhost:3306)/tinlink" down

# 生成API文档
swagger:
	@swag init -g cmd/server/main.go -o docs

# 帮助
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  test        - Run tests"
	@echo "  coverage    - Generate coverage report"
	@echo "  bench       - Run benchmarks"
	@echo "  lint        - Run linter"
	@echo "  docker-up   - Start with docker-compose"
	@echo "  docker-down - Stop docker-compose"