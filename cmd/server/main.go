package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"tinLink/internal/config"
	"tinLink/internal/handler"
	"tinLink/internal/middleware"
	"tinLink/internal/pkg/circuitbreaker"
	"tinLink/internal/pkg/snowflake"
	"tinLink/internal/pkg/tracer"
	"tinLink/internal/repository"
	"tinLink/internal/service"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化链路追踪（若启用）
	if cfg.Tracing.Enabled {
		tp, err := tracer.InitTracer(cfg.Tracing.Endpoint, cfg.Tracing.Service)
		if err != nil {
			log.Printf("Warning: Failed to init tracer: %v", err)
		} else {
			defer func() {
				if tp != nil {
					tp.Shutdown(context.Background())
				}
			}()
		}
	}

	// 3. 初始化MySQL连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect MySQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 4. 初始化Redis连接
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect Redis: %v", err)
	}

	// 5. 初始化Snowflake ID生成器
	idGenerator, err := snowflake.New(cfg.Server.MachineID)
	if err != nil {
		log.Fatalf("Failed to init snowflake: %v", err)
	}

	// 6. 初始化熔断器
	cb := circuitbreaker.New(circuitbreaker.Config{
		MaxRequests:  100,
		Interval:     10 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  10,
	})

	// 7. 初始化各层组件
	urlRepo := repository.NewURLRepository(db)
	cacheRepo := repository.NewCacheRepository(rdb)
	localCache := repository.NewLocalCache(10000, 5*time.Minute)
	urlService := service.NewURLService(urlRepo, cacheRepo, localCache, idGenerator, cb)
	statsService := service.NewStatsService(urlRepo, cacheRepo)
	urlHandler := handler.NewURLHandler(urlService)
	statsHandler := handler.NewStatsHandler(statsService)

	// 8. 配置路由
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// 中间件
	r.Use(middleware.Tracing())
	r.Use(middleware.Metrics())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.Cors())
	r.Use(middleware.RateLimit(cfg.RateLimit.QPS, cfg.RateLimit.Burst))

	// 健康检查 & 监控
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API路由
	v1 := r.Group("/api/v1")
	{
		v1.POST("/shorten", urlHandler.Shorten)
		v1.GET("/urls/:code", urlHandler.GetURL)
		v1.DELETE("/urls/:code", urlHandler.DeleteURL)
		v1.GET("/stats/:code", statsHandler.GetStats)
		v1.GET("/stats/:code/daily", statsHandler.GetDailyStats)
	}

	r.GET("/:code", urlHandler.Redirect)

	// 9. 启动服务器
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 10. 优雅退出
	gracefulShutdown(srv, sqlDB, rdb)
}

func gracefulShutdown(srv *http.Server, sqlDB *sql.DB, rdb *redis.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing MySQL: %v", err)
	}

	if err := rdb.Close(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	log.Println("Server exited gracefully")
}
