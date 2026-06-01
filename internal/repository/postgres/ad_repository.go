package postgres

import (
	"context"
	"errors"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"smartbid-backend/internal/domain"
)

var goquPostgresDialect = goqu.Dialect("postgres")

type AdRepository struct {
	pool *pgxpool.Pool
}

func NewAdRepository(pool *pgxpool.Pool) *AdRepository {
	return &AdRepository{pool: pool}
}

func adDescriptionValue(description *string) string {
	if description == nil {
		return ""
	}
	return *description
}

func createAdQuery(input domain.CreateAdInput) (string, []any, error) {
	return goquPostgresDialect.
		Insert("ads").
		Rows(goqu.Record{
			"title":       input.Title,
			"chat_id":     input.ChatId,
			"message_id":  input.MessageId,
			"description": adDescriptionValue(input.Description),
			"photo":       input.Photo,
			"start_price": input.Price,
			"final_price": input.Price,
			"status_id": goqu.L("(select id from ad_status where slug = ?)", domain.AdStatusCreated),
		}).
		Returning(
			goqu.L("id::text"),
			"title",
			"chat_id",
			"message_id",
			"description",
			goqu.L("coalesce(photo, ''::bytea)"),
			"start_price",
			"created_at",
			"updated_at",
		).
		Prepared(true).
		ToSQL()
}

func (r *AdRepository) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	query, args, err := createAdQuery(input)
	if err != nil {
		return domain.Ad{}, err
	}

	var ad domain.Ad
	var description string
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		return domain.Ad{}, err
	}

	ad.Description = &description
	ad.Status = domain.AdStatusCreated

	return ad, nil
}

func (r *AdRepository) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	query, args, err := goquPostgresDialect.
		From("ads").
		Select(
			goqu.L("ads.id::text"),
			goqu.I("ads.title"),
			goqu.I("ads.chat_id"),
			goqu.I("ads.message_id"),
			goqu.I("ads.description"),
			goqu.L("coalesce(ads.photo, ''::bytea)"),
			goqu.I("ads.start_price"),
			goqu.I("ad_status.slug"),
			goqu.I("ads.created_at"),
			goqu.I("ads.updated_at"),
		).
		Join(
			goqu.T("ad_status"),
			goqu.On(goqu.I("ad_status.id").Eq(goqu.I("ads.status_id"))),
		).
		Where(goqu.I("ads.id").Eq(id)).
		Prepared(true).
		ToSQL()
	if err != nil {
		return domain.Ad{}, err
	}

	var ad domain.Ad
	var description string
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &ad.Status, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ad{}, domain.ErrAdNotFound
		}
		return domain.Ad{}, err
	}

	ad.Description = &description

	return ad, nil
}
