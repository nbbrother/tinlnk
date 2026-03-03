package circuitbreaker

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Config struct {
	MaxRequests  uint32        // HalfOpen状态最大请求数
	Interval     time.Duration // 统计窗口
	Timeout      time.Duration // Open状态超时时间
	FailureRatio float64       // 失败率阈值
	MinRequests  uint32        // 最小请求数（用于计算失败率）
}

type CircuitBreaker struct {
	mu          sync.Mutex
	state       State
	config      Config
	failures    uint32
	successes   uint32
	requests    uint32
	lastFailure time.Time
	openTime    time.Time
}

func New(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// Allow 检查是否允许请求通过
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// 检查是否超时，可以转为HalfOpen
		if time.Since(cb.openTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.requests = 0
			cb.failures = 0
			cb.successes = 0
			return true
		}
		return false

	case StateHalfOpen:
		// HalfOpen状态限制请求数
		if cb.requests < cb.config.MaxRequests {
			cb.requests++
			return true
		}
		return false
	}

	return false
}

// Success 记录成功
func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++

	if cb.state == StateHalfOpen {
		// HalfOpen状态下，达到最大请求数且全部成功则恢复
		if cb.requests >= cb.config.MaxRequests {
			cb.state = StateClosed
			cb.reset()
		}
	}
}

// Failure 记录失败
func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		total := cb.failures + cb.successes
		if total >= cb.config.MinRequests {
			failureRatio := float64(cb.failures) / float64(total)
			if failureRatio >= cb.config.FailureRatio {
				cb.state = StateOpen
				cb.openTime = time.Now()
			}
		}

	case StateHalfOpen:
		// HalfOpen状态下任何失败都立即打开熔断
		cb.state = StateOpen
		cb.openTime = time.Now()
	}
}

func (cb *CircuitBreaker) reset() {
	cb.failures = 0
	cb.successes = 0
	cb.requests = 0
}

// State 获取当前状态
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Stats 获取统计信息
func (cb *CircuitBreaker) Stats() (state State, failures, successes uint32) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state, cb.failures, cb.successes
}
