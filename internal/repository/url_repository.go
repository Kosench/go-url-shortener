package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Kosench/go-url-shortener/internal/model"
	"time"
)

type PostgresURLRepository struct {
	db *sql.DB
}

func NewPostgresURLRepository(db *sql.DB) URLRepository {
	return &PostgresURLRepository{
		db: db,
	}
}

func (r PostgresURLRepository) Create(ctx context.Context, url *model.URL) error {
	query := `
	INSERT INTO urls (original_url, short_code, created_at)
	VALUES ($1, $2, $3)
	RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		url.OriginalURL,
		url.ShortCode,
		time.Now(),
	).Scan(&url.ID)

	if err != nil {
		return fmt.Errorf("failed to create URL: %w", err)
	}

	return nil
}

func (r *PostgresURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	query := `
	SELECT original_url
	FROM urls
	WHERE short_code = $1
	`

	url := &model.URL{}
	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(
		&url.ID,
		&url.OriginalURL,
		&url.ShortCode,
		&url.ClickCount,
		&url.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

func (r PostgresURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check short code existence: %w", err)
	}

	return exists, nil
}
