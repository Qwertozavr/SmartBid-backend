package postgres

import (
	"context"
	"database/sql"
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

func adPretendentIDValue(pretendentID sql.NullInt64) *int {
	if !pretendentID.Valid {
		return nil
	}
	value := int(pretendentID.Int64)
	return &value
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
			"status_id":   goqu.L("(select id from ad_status where slug = ?)", domain.AdStatusCreated),
		}).
		Returning(
			goqu.L("id::text"),
			"title",
			"chat_id",
			"message_id",
			"description",
			goqu.L("coalesce(photo, ''::bytea)"),
			"final_price",
			"pretendent_id",
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
	var pretendentID sql.NullInt64
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &pretendentID, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		return domain.Ad{}, err
	}

	ad.Description = &description
	ad.PretendentID = adPretendentIDValue(pretendentID)
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
			goqu.I("ads.final_price"),
			goqu.I("ads.pretendent_id"),
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
	var pretendentID sql.NullInt64
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &pretendentID, &ad.Status, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ad{}, domain.ErrAdNotFound
		}
		return domain.Ad{}, err
	}

	ad.Description = &description
	ad.PretendentID = adPretendentIDValue(pretendentID)

	return ad, nil
}

func updateAdPriceQuery(input domain.UpdateAdPriceInput) (string, []any, error) {
	return goquPostgresDialect.
		Update("ads").
		Set(goqu.Record{
			"final_price":   input.Price,
			"pretendent_id": input.PretendentID,
			"updated_at":    goqu.L("now()"),
		}).
		Where(goqu.I("id").Eq(input.AdID)).
		Returning(
			goqu.L("id::text"),
			"final_price",
		).
		Prepared(true).
		ToSQL()
}

func (r *AdRepository) UpdatePrice(ctx context.Context, input domain.UpdateAdPriceInput) (domain.AdPriceUpdate, error) {
	query, args, err := updateAdPriceQuery(input)
	if err != nil {
		return domain.AdPriceUpdate{}, err
	}

	var update domain.AdPriceUpdate
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&update.AdID, &update.Price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AdPriceUpdate{}, domain.ErrAdNotFound
		}
		return domain.AdPriceUpdate{}, err
	}

	return update, nil
}

func updateAdStatusQuery(id string, status domain.AdStatus) (string, []any, error) {
	return goquPostgresDialect.
		Update("ads").
		Set(goqu.Record{
			"status_id":  goqu.L("(select id from ad_status where slug = ?)", status),
			"updated_at": goqu.L("now()"),
		}).
		Where(goqu.I("id").Eq(id)).
		Prepared(true).
		ToSQL()
}

func (r *AdRepository) UpdateStatus(ctx context.Context, id string, status domain.AdStatus) error {
	query, args, err := updateAdStatusQuery(id, status)
	if err != nil {
		return err
	}

	commandTag, err := executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrAdNotFound
	}

	return nil
}
