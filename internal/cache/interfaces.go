package cache

import (
	"context"
	"time"
)

// Cache - основной интерфейс для работы с кэшем
type Cache interface {
	// Базовые операции
	Set(ctx context.Context, key string, value interface{}) error
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Строковые операции
	SetString(ctx context.Context, key string, value string) error
	GetString(ctx context.Context, key string) (string, error)

	// Управление соединением
	HealthCheck(ctx context.Context) error
	Close() error
}

// CounterCache - интерфейс для работы со счетчиками
type CounterCache interface {
	IncrementClickCount(ctx context.Context, shortCode string) error
	GetClickCount(ctx context.Context, shortCode string) (int64, error)
	SetClickCount(ctx context.Context, shortCode string, count int64) error
}

// RateLimiter - интерфейс для rate limiting
type RateLimiter interface {
	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)
}

// CacheManager - полный интерфейс кэша (композиция интерфейсов)
type CacheManager interface {
	Cache
	CounterCache
	RateLimiter

	// Дополнительные операции
	FlushCache(ctx context.Context) error
}

// NullCache - заглушка для работы без кэша (Null Object Pattern)
type NullCache struct{}

func NewNullCache() *NullCache {
	return &NullCache{}
}

func (n *NullCache) Set(ctx context.Context, key string, value interface{}) error {
	return nil // Ничего не делаем
}

func (n *NullCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (n *NullCache) Get(ctx context.Context, key string, dest interface{}) error {
	return ErrCacheMiss // Всегда miss
}

func (n *NullCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *NullCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (n *NullCache) SetString(ctx context.Context, key string, value string) error {
	return nil
}

func (n *NullCache) GetString(ctx context.Context, key string) (string, error) {
	return "", ErrCacheMiss
}

func (n *NullCache) HealthCheck(ctx context.Context) error {
	return nil // Всегда "здоров"
}

func (n *NullCache) Close() error {
	return nil
}
