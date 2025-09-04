package cache

import "errors"

// Ошибки кэша
var (
	// ErrCacheMiss возникает когда ключ не найден в кэше
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheConnectionFailed возникает при проблемах с подключением к кэшу
	ErrCacheConnectionFailed = errors.New("cache connection failed")

	// ErrCacheTimeout возникает при таймауте операции
	ErrCacheTimeout = errors.New("cache operation timeout")

	// ErrInvalidCacheKey возникает при невалидном ключе
	ErrInvalidCacheKey = errors.New("invalid cache key")

	// ErrCacheFull возникает когда кэш переполнен
	ErrCacheFull = errors.New("cache is full")
)

// CacheError - структурированная ошибка кэша
type CacheError struct {
	Op  string // Операция: "get", "set", "delete"
	Key string // Ключ кэша
	Err error  // Оригинальная ошибка
}

func (e *CacheError) Error() string {
	if e.Key != "" {
		return "cache " + e.Op + " '" + e.Key + "': " + e.Err.Error()
	}
	return "cache " + e.Op + ": " + e.Err.Error()
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

// NewCacheError создает новую структурированную ошибку
func NewCacheError(op, key string, err error) error {
	return &CacheError{
		Op:  op,
		Key: key,
		Err: err,
	}
}
