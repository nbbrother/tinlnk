
```markdown
# TinyLink - URL Shortener Service

## Introduction
TinyLink is a high-performance, scalable URL shortening service. It converts long URLs into concise short links and provides access statistics, monitoring, and alerting capabilities, suitable for marketing campaigns, link management, and data analytics.

## Features
- **Short Link Generation**: Supports auto-generated and custom short codes, using Snowflake algorithm for global uniqueness and Base62 encoding for URL-friendly strings.
- **Fast Redirection**: Multi-level caching (local memory + Redis) accelerates redirection, handling high concurrency; Bloom filter blocks non-existent short codes to prevent cache penetration.
- **Access Statistics**: Records PV (page views) and UV (unique visitors) for each short link, with daily aggregation.
- **Hotspot Detection**: Automatically identifies popular short links and pre-warms them into local cache to handle traffic spikes.
- **Observability**: Integrated with Prometheus metrics, Jaeger distributed tracing, and Zap structured logging for easy operations and troubleshooting.
- **Circuit Breaker & Rate Limiting**: Circuit breaker protects downstream services; IP-level token bucket rate limiting prevents malicious attacks.
- **Distributed Lock**: Redis-based distributed lock ensures concurrency safety for critical operations (e.g., short code generation).
- **Graceful Shutdown**: Supports smooth termination, ensuring ongoing requests complete before releasing resources.

## Quick Start

### Prerequisites
- Docker and Docker Compose (recommended)
- Go 1.25+

### Start with Docker Compose
1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/tinylink.git
   cd tinylink



# TinyLink - 短链接服务

## 项目简介
TinyLink 是一个高性能、可扩展的短链接生成与跳转服务。它可以将长 URL 转换为简洁的短链接，并提供访问统计、监控告警等功能，适用于营销推广、链接管理、数据分析等场景。

## 功能特性
- **短链接生成**：支持自动生成短码和自定义短码，基于 Snowflake 算法保证全局唯一性，Base62 编码生成 URL 友好的短字符串。
- **高效跳转**：多级缓存（本地内存 + Redis）加速跳转，支持高并发访问；布隆过滤器拦截不存在短码，防止缓存穿透。
- **访问统计**：记录每个短链接的 PV（访问次数）和 UV（独立访客），支持按日统计。
- **热点探测**：自动识别热点短链接并预热到本地缓存，应对突发流量。
- **可观测性**：集成 Prometheus 指标监控、Jaeger 分布式追踪、Zap 结构化日志，便于运维和排查问题。
- **熔断与限流**：熔断器保护下游服务，IP 级令牌桶限流，防止恶意攻击。
- **分布式锁**：基于 Redis 实现分布式锁，保障关键操作（如短码生成）的并发安全。
- **平滑关闭**：支持优雅停机，确保请求处理完成后再释放资源。

## 快速开始

### 环境要求
- Docker 及 Docker Compose (推荐)
- Go 1.25+

### 使用 Docker Compose 一键启动
1. 克隆项目并进入目录：
   ```bash
   git clone https://github.com/your-repo/tinylink.git
   cd tinylink