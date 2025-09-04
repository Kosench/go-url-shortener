package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Проверяем, что RedisClient реализует все интерфейсы
var (
	_ Cache        = (*RedisClient)(nil)
	_ CounterCache = (*RedisClient)(nil)
	_ RateLimiter  = (*RedisClient)(nil)
	_ CacheManager = (*RedisClient)(nil)
)

// RedisClient - реализация кэша на основе Redis
type RedisClient struct {
	client     *redis.Client
	ttl        time.Duration
	keyBuilder *KeyBuilder
}

// RedisConfig - конфигурация для Redis
type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	CacheTTL     int    // в секундах
	Namespace    string // опциональный namespace для ключей
}

// NewRedisClient создает новый Redis клиент
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, NewCacheError("connect", "", fmt.Errorf("failed to connect to Redis: %w", err))
	}

	return &RedisClient{
		client:     client,
		ttl:        time.Duration(cfg.CacheTTL) * time.Second,
		keyBuilder: NewKeyBuilder(cfg.Namespace),
	}, nil
}

// === Реализация интерфейса Cache ===

// Set сохраняет значение в кэш с дефолтным TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}) error {
	return r.SetWithTTL(ctx, key, value, r.ttl)
}

// SetWithTTL сохраняет значение с кастомным TTL
func (r *RedisClient) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if key == "" {
		return NewCacheError("set", key, ErrInvalidCacheKey)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return NewCacheError("set", key, fmt.Errorf("failed to marshal value: %w", err))
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return NewCacheError("set", key, err)
	}

	return nil
}

// Get получает значение из кэша
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	if key == "" {
		return NewCacheError("get", key, ErrInvalidCacheKey)
	}

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return NewCacheError("get", key, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return NewCacheError("get", key, fmt.Errorf("failed to unmarshal value: %w", err))
	}

	return nil
}

// Delete удаляет значения из кэша
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// Фильтруем пустые ключи
	validKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			validKeys = append(validKeys, key)
		}
	}

	if len(validKeys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, validKeys...).Err(); err != nil {
		return NewCacheError("delete", "", err)
	}

	return nil
}

// Exists проверяет существование ключа
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, NewCacheError("exists", key, ErrInvalidCacheKey)
	}

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, NewCacheError("exists", key, err)
	}

	return result > 0, nil
}

// GetString получает строковое значение
func (r *RedisClient) GetString(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", NewCacheError("get", key, ErrInvalidCacheKey)
	}

	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrCacheMiss
		}
		return "", NewCacheError("get", key, err)
	}

	return value, nil
}

// SetString сохраняет строковое значение
func (r *RedisClient) SetString(ctx context.Context, key string, value string) error {
	if key == "" {
		return NewCacheError("set", key, ErrInvalidCacheKey)
	}

	if err := r.client.Set(ctx, key, value, r.ttl).Err(); err != nil {
		return NewCacheError("set", key, err)
	}

	return nil
}

// HealthCheck проверяет соединение с Redis
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return NewCacheError("ping", "", err)
	}
	return nil
}

// Close закрывает соединение с Redis
func (r *RedisClient) Close() error {
	if err := r.client.Close(); err != nil {
		return NewCacheError("close", "", err)
	}
	return nil
}

// === Реализация интерфейса CounterCache ===

// IncrementClickCount атомарно увеличивает счетчик кликов
func (r *RedisClient) IncrementClickCount(ctx context.Context, shortCode string) error {
	key := r.keyBuilder.Clicks(shortCode)

	if err := r.client.Incr(ctx, key).Err(); err != nil {
		return NewCacheError("increment", key, err)
	}

	// Устанавливаем TTL для счетчика (дольше обычного кэша)
	r.client.Expire(ctx, key, r.ttl*24)

	return nil
}

// GetClickCount получает количество кликов из кэша
func (r *RedisClient) GetClickCount(ctx context.Context, shortCode string) (int64, error) {
	key := r.keyBuilder.Clicks(shortCode)

	result, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrCacheMiss
		}
		return 0, NewCacheError("get", key, err)
	}

	return result, nil
}

// SetClickCount устанавливает счетчик кликов
func (r *RedisClient) SetClickCount(ctx context.Context, shortCode string, count int64) error {
	key := r.keyBuilder.Clicks(shortCode)

	// Храним счетчики дольше обычного кэша
	if err := r.client.Set(ctx, key, count, r.ttl*24).Err(); err != nil {
		return NewCacheError("set", key, err)
	}

	return nil
}

// === Реализация интерфейса RateLimiter ===

// IncrementRateLimit увеличивает счетчик для rate limiting
func (r *RedisClient) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	// Используем pipeline для атомарности
	pipe := r.client.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, NewCacheError("increment", key, err)
	}

	return incr.Val(), nil
}

// === Реализация интерфейса CacheManager ===

// FlushCache очищает весь кэш (использовать осторожно!)
func (r *RedisClient) FlushCache(ctx context.Context) error {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return NewCacheError("flush", "", err)
	}
	return nil
}

// === Дополнительные методы ===

// GetMultiple получает несколько значений за один запрос
func (r *RedisClient) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, NewCacheError("mget", "", err)
	}

	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if values[i] != nil {
			result[key] = values[i].(string)
		}
	}

	return result, nil
}

// SetMultiple устанавливает несколько значений за один запрос
func (r *RedisClient) SetMultiple(ctx context.Context, items map[string]interface{}) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return NewCacheError("set", key, err)
		}
		pipe.Set(ctx, key, data, r.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return NewCacheError("mset", "", err)
	}

	return nil
}

// GetTTL возвращает оставшееся время жизни ключа
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, NewCacheError("ttl", key, err)
	}

	if ttl == -2 {
		return 0, ErrCacheMiss
	}

	return ttl, nil
}

// ScanKeys возвращает ключи по паттерну (использовать осторожно в production!)
func (r *RedisClient) ScanKeys(ctx context.Context, pattern string, limit int64) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		var batch []string
		var err error

		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return nil, NewCacheError("scan", pattern, err)
		}

		keys = append(keys, batch...)

		if cursor == 0 || int64(len(keys)) >= limit {
			break
		}
	}

	if int64(len(keys)) > limit {
		keys = keys[:limit]
	}

	return keys, nil
}

// Info возвращает информацию о Redis сервере
func (r *RedisClient) Info(ctx context.Context) (map[string]string, error) {
	info, err := r.client.Info(ctx).Result()
	if err != nil {
		return nil, NewCacheError("info", "", err)
	}

	// Парсим info в map (упрощенно)
	result := make(map[string]string)
	// ... парсинг info строки
	result["raw"] = info

	return result, nil
}

// GetKeyBuilder возвращает построитель ключей
func (r *RedisClient) GetKeyBuilder() *KeyBuilder {
	return r.keyBuilder
}
