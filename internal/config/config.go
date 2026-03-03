package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Tracing   TracingConfig   `mapstructure:"tracing"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
}

type ServerConfig struct {
	Addr      string `mapstructure:"addr"`
	Mode      string `mapstructure:"mode"`
	MachineID int64  `mapstructure:"machine_id"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

type RateLimitConfig struct {
	QPS   int `mapstructure:"qps"`
	Burst int `mapstructure:"burst"`
}

type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
	Service  string `mapstructure:"service"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetEnvPrefix("TINLINK")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定所有环境变量
	bindings := []string{
		"server.addr", "server.mode", "server.machine_id",
		"mysql.host", "mysql.port", "mysql.user", "mysql.password", "mysql.database",
		"redis.host", "redis.port", "redis.password",
		"rate_limit.qps", "rate_limit.burst",
		"tracing.enabled", "tracing.endpoint", "tracing.service",
		"metrics.enabled", "metrics.path",
	}
	for _, key := range bindings {
		_ = viper.BindEnv(key)
	}

	// 设置默认值
	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.machine_id", 1)
	viper.SetDefault("mysql.max_open_conns", 100)
	viper.SetDefault("mysql.max_idle_conns", 10)
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.pool_size", 100)
	viper.SetDefault("redis.min_idle_conns", 10)
	viper.SetDefault("rate_limit.qps", 1000)
	viper.SetDefault("rate_limit.burst", 2000)
	viper.SetDefault("tracing.enabled", false)
	viper.SetDefault("tracing.service", "tinlink")
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.path", "/metrics")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using environment variables and defaults")
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 手动环境变量覆盖
	overrideFromEnv(&cfg)

	// 调试日志
	log.Printf("Config loaded - Server: %s (mode: %s)", cfg.Server.Addr, cfg.Server.Mode)
	log.Printf("Config loaded - MySQL: %s:%d/%s", cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	log.Printf("Config loaded - Redis: %s:%d", cfg.Redis.Host, cfg.Redis.Port)
	log.Printf("Config loaded - RateLimit: QPS=%d, Burst=%d", cfg.RateLimit.QPS, cfg.RateLimit.Burst)
	log.Printf("Config loaded - MySQL Pool: MaxOpen=%d, MaxIdle=%d", cfg.MySQL.MaxOpenConns, cfg.MySQL.MaxIdleConns)

	return &cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("TINLINK_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("TINLINK_SERVER_MODE"); v != "" {
		cfg.Server.Mode = v
	}
	if v := os.Getenv("TINLINK_MYSQL_HOST"); v != "" {
		cfg.MySQL.Host = v
	}
	if v := os.Getenv("TINLINK_MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.MySQL.Port = port
		}
	}
	if v := os.Getenv("TINLINK_MYSQL_USER"); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("TINLINK_MYSQL_PASSWORD"); v != "" {
		cfg.MySQL.Password = v
	}
	if v := os.Getenv("TINLINK_MYSQL_DATABASE"); v != "" {
		cfg.MySQL.Database = v
	}
	if v := os.Getenv("TINLINK_REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("TINLINK_REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("TINLINK_TRACING_ENDPOINT"); v != "" {
		cfg.Tracing.Endpoint = v
	}
	if v := os.Getenv("TINLINK_TRACING_ENABLED"); v != "" {
		cfg.Tracing.Enabled = (v == "true" || v == "1")
	}
}
