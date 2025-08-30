package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Kosench/go-url-shortener/internal/model"
	"time"

	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
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
		return apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to create URL",
			err,
		)
	}

	return nil
}

func (r *PostgresURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	query := `
	SELECT id, original_url, short_code, click_count, created_at
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
		return nil, fmt.Errorf("URL with short code '%s': %w", shortCode, apperrors.ErrURLNotFound)
	}

	if err != nil {
		return nil, apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to get URL",
			err,
		)
	}

	return url, nil
}

func (r PostgresURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to check short code existence",
			err,
		)
	}

	return exists, nil
}
