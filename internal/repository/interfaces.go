package repository

import (
	"context"
	"github.com/Kosench/go-url-shortener/internal/model"
)

type URLRepository interface {
	Create(ctx context.Context, url *model.URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error)
	ExistsByShortCode(ctx context.Context, shortCode string) (bool, error)
	IncrementClickCount(ctx context.Context, id int64) error
}
