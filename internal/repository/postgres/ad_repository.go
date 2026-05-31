package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"smartbid-backend/internal/domain"
)

type AdRepository struct {
	pool *pgxpool.Pool
}

func NewAdRepository(pool *pgxpool.Pool) *AdRepository {
	return &AdRepository{pool: pool}
}

func (r *AdRepository) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	const query = `
		with status as (
			select id
			from ad_status
			where slug = $4
		), inserted as (
			insert into ads (title, description, start_price, final_price, status_id)
			select $1, $2, $3, $3, status.id
			from status
			returning id, title, description, start_price, status_id, created_at, updated_at
		)
		select
			inserted.id::text,
			inserted.title,
			inserted.description,
			inserted.start_price,
			ad_status.slug,
			inserted.created_at,
			inserted.updated_at
		from inserted
		join ad_status on ad_status.id = inserted.status_id
	`

	var ad domain.Ad
	err := executor(ctx, r.pool).QueryRow(ctx, query, input.Title, input.Description, input.Price, domain.AdStatusCreated).
		Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &ad.Status, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		return domain.Ad{}, err
	}

	return ad, nil
}

func (r *AdRepository) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	const query = `
		select
			ads.id::text,
			ads.title,
			ads.description,
			ads.start_price,
			ad_status.slug,
			ads.created_at,
			ads.updated_at
		from ads
		join ad_status on ad_status.id = ads.status_id
		where ads.id = $1
	`

	var ad domain.Ad
	err := executor(ctx, r.pool).QueryRow(ctx, query, id).
		Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &ad.Status, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ad{}, domain.ErrAdNotFound
		}
		return domain.Ad{}, err
	}

	return ad, nil
}
