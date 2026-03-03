package lock

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var (
	ErrLockNotHeld = errors.New("lock not held")
	ErrLockFailed  = errors.New("failed to acquire lock")
)

type DistributedLock struct {
	rdb   *redis.Client
	key   string
	value string
	ttl   time.Duration
}

func NewDistributedLock(rdb *redis.Client, key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		rdb:   rdb,
		key:   "lock:" + key,
		value: uuid.New().String(),
		ttl:   ttl,
	}
}

func (l *DistributedLock) Lock(ctx context.Context) error {
	ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockFailed
	}
	return nil
}

func (l *DistributedLock) LockWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		if err := l.Lock(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			continue
		}
	}
	return ErrLockFailed
}

func (l *DistributedLock) Unlock(ctx context.Context) error {
	script := redis.NewScript(`
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end
    `)

	result, err := script.Run(ctx, l.rdb, []string{l.key}, l.value).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}

func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	script := redis.NewScript(`
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("PEXPIRE", KEYS[1], ARGV[2])
        else
            return 0
        end
    `)

	result, err := script.Run(ctx, l.rdb, []string{l.key}, l.value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}
