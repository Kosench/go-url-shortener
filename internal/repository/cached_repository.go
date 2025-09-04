package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Kosench/go-url-shortener/internal/cache"
	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
	"github.com/Kosench/go-url-shortener/internal/model"
)

// CachedURLRepository - репозиторий с кэшированием
type CachedURLRepository struct {
	db    *sql.DB
	cache *cache.RedisClient
}

// NewCachedURLRepository создает новый репозиторий с кэшем
func NewCachedURLRepository(db *sql.DB, cache *cache.RedisClient) URLRepository {
	return &CachedURLRepository{
		db:    db,
		cache: cache,
	}
}

// Create создает новую запись URL
func (r *CachedURLRepository) Create(ctx context.Context, url *model.URL) error {
	// Атомарная вставка
	query := `
	INSERT INTO urls (original_url, short_code, created_at)
	VALUES ($1, $2, $3)
	ON CONFLICT (short_code) DO NOTHING
	RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		url.OriginalURL,
		url.ShortCode,
		url.CreatedAt,
	).Scan(&url.ID)

	if err == sql.ErrNoRows {
		return apperrors.ErrShortCodeExists
	}

	if err != nil {
		return apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to create URL",
			err,
		)
	}

	// Кэшируем созданный URL
	cacheKey := cache.CacheKeys.URL(url.ShortCode)
	if err := r.cache.Set(ctx, cacheKey, url); err != nil {
		// Логируем ошибку кэша, но не прерываем операцию
		log.Printf("Failed to cache URL: %v", err)
	}

	// Также кэшируем маппинг original URL -> short code для быстрого поиска дубликатов
	reverseCacheKey := cache.CacheKeys.ShortCode(url.OriginalURL)
	if err := r.cache.SetString(ctx, reverseCacheKey, url.ShortCode); err != nil {
		log.Printf("Failed to cache reverse mapping: %v", err)
	}

	return nil
}

// GetByShortCode получает URL по короткому коду
func (r *CachedURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	// Сначала проверяем кэш
	cacheKey := cache.CacheKeys.URL(shortCode)
	var cachedURL model.URL

	err := r.cache.Get(ctx, cacheKey, &cachedURL)
	if err == nil {
		// Cache hit - возвращаем из кэша
		return &cachedURL, nil
	}

	if err != cache.ErrCacheMiss {
		// Логируем ошибку кэша, но продолжаем с БД
		log.Printf("Cache error: %v", err)
	}

	// Cache miss - идем в БД
	query := `
	SELECT id, original_url, short_code, click_count, created_at
	FROM urls
	WHERE short_code = $1
	`

	url := &model.URL{}
	err = r.db.QueryRowContext(ctx, query, shortCode).Scan(
		&url.ID,
		&url.OriginalURL,
		&url.ShortCode,
		&url.ClickCount,
		&url.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL with short code '%s': %w", shortCode, apperrors.ErrURLNotFound)
	}

	if err != nil {
		return nil, apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to get URL",
			err,
		)
	}

	// Кэшируем результат
	if err := r.cache.Set(ctx, cacheKey, url); err != nil {
		log.Printf("Failed to cache URL: %v", err)
	}

	// Синхронизируем счетчик кликов с кэшем
	if err := r.cache.SetClickCount(ctx, shortCode, url.ClickCount); err != nil {
		log.Printf("Failed to cache click count: %v", err)
	}

	return url, nil
}

// ExistsByShortCode проверяет существование короткого кода
func (r *CachedURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	// Сначала проверяем кэш
	cacheKey := cache.CacheKeys.URL(shortCode)
	exists, err := r.cache.Exists(ctx, cacheKey)
	if err == nil && exists {
		return true, nil
	}

	// Проверяем в БД
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	var dbExists bool
	err = r.db.QueryRowContext(ctx, query, shortCode).Scan(&dbExists)
	if err != nil {
		return false, apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to check short code existence",
			err,
		)
	}

	return dbExists, nil
}

// IncrementClickCount увеличивает счетчик кликов
func (r *CachedURLRepository) IncrementClickCount(ctx context.Context, id int64) error {
	// Обновляем в БД
	query := `
	UPDATE urls
	SET click_count = click_count + 1
	WHERE id = $1
	RETURNING short_code, click_count
	`

	var shortCode string
	var newCount int64

	err := r.db.QueryRowContext(ctx, query, id).Scan(&shortCode, &newCount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("URL with ID %d: %w", id, apperrors.ErrURLNotFound)
	}

	if err != nil {
		return apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to increment click count",
			err,
		)
	}

	// Обновляем счетчик в кэше
	if err := r.cache.SetClickCount(ctx, shortCode, newCount); err != nil {
		log.Printf("Failed to update click count in cache: %v", err)
	}

	// Инвалидируем кэш URL чтобы при следующем запросе обновился click_count
	cacheKey := cache.CacheKeys.URL(shortCode)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("Failed to invalidate URL cache: %v", err)
	}

	return nil
}

// GetByOriginalURL ищет существующий короткий код для URL (для предотвращения дубликатов)
func (r *CachedURLRepository) GetByOriginalURL(ctx context.Context, originalURL string) (*model.URL, error) {
	// Проверяем кэш обратного маппинга
	reverseCacheKey := cache.CacheKeys.ShortCode(originalURL)
	shortCode, err := r.cache.GetString(ctx, reverseCacheKey)
	if err == nil && shortCode != "" {
		// Нашли в кэше, получаем полный URL
		return r.GetByShortCode(ctx, shortCode)
	}

	// Ищем в БД
	query := `
	SELECT id, original_url, short_code, click_count, created_at
	FROM urls
	WHERE original_url = $1
	ORDER BY created_at DESC
	LIMIT 1
	`

	url := &model.URL{}
	err = r.db.QueryRowContext(ctx, query, originalURL).Scan(
		&url.ID,
		&url.OriginalURL,
		&url.ShortCode,
		&url.ClickCount,
		&url.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.ErrURLNotFound
	}

	if err != nil {
		return nil, apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to get URL by original",
			err,
		)
	}

	// Кэшируем результаты
	if err := r.cache.SetString(ctx, reverseCacheKey, url.ShortCode); err != nil {
		log.Printf("Failed to cache reverse mapping: %v", err)
	}

	cacheKey := cache.CacheKeys.URL(url.ShortCode)
	if err := r.cache.Set(ctx, cacheKey, url); err != nil {
		log.Printf("Failed to cache URL: %v", err)
	}

	return url, nil
}

// BatchIncrementClickCount для массового обновления счетчиков (из Redis в БД)
func (r *CachedURLRepository) BatchIncrementClickCount(ctx context.Context, updates map[string]int64) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE urls 
		SET click_count = click_count + $1 
		WHERE short_code = $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for shortCode, increment := range updates {
		if _, err := stmt.ExecContext(ctx, increment, shortCode); err != nil {
			return fmt.Errorf("failed to update clicks for %s: %w", shortCode, err)
		}
	}

	return tx.Commit()
}

// WarmupCache предзагружает популярные URL в кэш
func (r *CachedURLRepository) WarmupCache(ctx context.Context, limit int) error {
	query := `
	SELECT id, original_url, short_code, click_count, created_at
	FROM urls
	ORDER BY click_count DESC, created_at DESC
	LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("failed to query popular URLs: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.ClickCount, &url.CreatedAt); err != nil {
			log.Printf("Failed to scan URL: %v", err)
			continue
		}

		cacheKey := cache.CacheKeys.URL(url.ShortCode)
		if err := r.cache.SetWithTTL(ctx, cacheKey, &url, 24*time.Hour); err != nil {
			log.Printf("Failed to cache URL %s: %v", url.ShortCode, err)
		} else {
			count++
		}
	}

	log.Printf("Warmed up cache with %d URLs", count)
	return nil
}
