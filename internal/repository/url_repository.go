package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Kosench/go-url-shortener/internal/model"

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

func (r *PostgresURLRepository) Create(ctx context.Context, url *model.URL) error {
	// Атомарная вставка: если short_code уже существует, RETURNING не вернёт строк -> sql.ErrNoRows
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
		// конфликт уникальности short_code
		return apperrors.ErrShortCodeExists
	}

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

func (r *PostgresURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
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

func (r *PostgresURLRepository) IncrementClickCount(ctx context.Context, id int64) error {
	query := `
	UPDATE urls
	SET click_count = click_count + 1
	WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to increment click count",
			err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewBusinessError(
			"DATABASE_ERROR",
			"failed to get rows affected",
			err,
		)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("URL with ID %d: %w", id, apperrors.ErrURLNotFound)
	}

	return nil
}