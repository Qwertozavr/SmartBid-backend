package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

func adTimeValue(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
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
			"published_at",
			"expires_at",
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
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &pretendentID, &publishedAt, &expiresAt, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		return domain.Ad{}, err
	}

	ad.Description = &description
	ad.PretendentID = adPretendentIDValue(pretendentID)
	ad.PublishedAt = adTimeValue(publishedAt)
	ad.ExpiresAt = adTimeValue(expiresAt)
	ad.Status = domain.AdStatusCreated

	return ad, nil
}

func (r *AdRepository) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	return r.findByID(ctx, id, false)
}

func (r *AdRepository) FindByIDForUpdate(ctx context.Context, id string) (domain.Ad, error) {
	return r.findByID(ctx, id, true)
}

func findAdByIDQuery(id string, forUpdate bool) (string, []any, error) {
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
			goqu.I("ads.published_at"),
			goqu.I("ads.expires_at"),
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
		return "", nil, err
	}
	if forUpdate {
		query += " FOR UPDATE"
	}
	return query, args, nil
}

func (r *AdRepository) findByID(ctx context.Context, id string, forUpdate bool) (domain.Ad, error) {
	query, args, err := findAdByIDQuery(id, forUpdate)
	if err != nil {
		return domain.Ad{}, err
	}

	var ad domain.Ad
	var description string
	var pretendentID sql.NullInt64
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).
		Scan(&ad.ID, &ad.Title, &ad.ChatId, &ad.MessageId, &description, &ad.Photo, &ad.Price, &pretendentID, &ad.Status, &publishedAt, &expiresAt, &ad.CreatedAt, &ad.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ad{}, domain.ErrAdNotFound
		}
		return domain.Ad{}, err
	}

	ad.Description = &description
	ad.PretendentID = adPretendentIDValue(pretendentID)
	ad.PublishedAt = adTimeValue(publishedAt)
	ad.ExpiresAt = adTimeValue(expiresAt)

	return ad, nil
}

func publishAdQuery(input domain.PublishAdUpdate) (string, []any, error) {
	return goquPostgresDialect.
		Update("ads").
		Set(goqu.Record{
			"status_id":    goqu.L("(select id from ad_status where slug = ?)", domain.AdStatusPublished),
			"published_at": input.PublishedAt,
			"expires_at":   input.ExpiresAt,
			"updated_at":   goqu.L("now()"),
		}).
		Where(
			goqu.I("id").Eq(input.AdID),
			goqu.I("status_id").Eq(goqu.L("(select id from ad_status where slug = ?)", domain.AdStatusCreated)),
		).
		Prepared(true).
		ToSQL()
}

func (r *AdRepository) Publish(ctx context.Context, input domain.PublishAdUpdate) error {
	query, args, err := publishAdQuery(input)
	if err != nil {
		return err
	}

	commandTag, err := executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrAdNotActive
	}

	return nil
}

func updateAdPriceQuery(input domain.UpdateAdPriceInput) (string, []any, error) {
	return goquPostgresDialect.
		Update("ads").
		Set(goqu.Record{
			"final_price":   input.Price,
			"pretendent_id": input.PretendentID,
			"updated_at":    goqu.L("now()"),
		}).
		Where(
			goqu.I("id").Eq(input.AdID),
			goqu.I("expires_at").Gt(input.Now),
			goqu.I("status_id").Eq(goqu.L("(select id from ad_status where slug = ?)", domain.AdStatusPublished)),
		).
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
			return domain.AdPriceUpdate{}, domain.ErrAdNotActive
		}
		return domain.AdPriceUpdate{}, err
	}

	return update, nil
}

func transitionAdStatusQuery(id string, from, to domain.AdStatus) (string, []any, error) {
	return goquPostgresDialect.
		Update("ads").
		Set(goqu.Record{
			"status_id":  goqu.L("(select id from ad_status where slug = ?)", to),
			"updated_at": goqu.L("now()"),
		}).
		Where(
			goqu.I("id").Eq(id),
			goqu.I("status_id").Eq(goqu.L("(select id from ad_status where slug = ?)", from)),
		).
		Prepared(true).
		ToSQL()
}

func (r *AdRepository) TransitionStatus(ctx context.Context, id string, from, to domain.AdStatus) error {
	query, args, err := transitionAdStatusQuery(id, from, to)
	if err != nil {
		return err
	}

	commandTag, err := executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrAdNotActive
	}

	return nil
}

func claimExpiredAdsQuery() string {
	return `
		select
			ads.id::text,
			ads.title,
			ads.chat_id,
			ads.message_id,
			ads.description,
			coalesce(ads.photo, ''::bytea),
			ads.final_price,
			ads.pretendent_id,
			ad_status.slug,
			ads.published_at,
			ads.expires_at,
			ads.created_at,
			ads.updated_at
		from ads
		join ad_status on ad_status.id = ads.status_id
		where ad_status.slug = $1
			and ads.expires_at <= $2
		order by ads.expires_at
		limit $3
		for update skip locked
	`
}

func (r *AdRepository) ClaimExpired(ctx context.Context, now time.Time, limit int) ([]domain.Ad, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, claimExpiredAdsQuery(), domain.AdStatusPublished, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ads := make([]domain.Ad, 0, limit)
	for rows.Next() {
		var ad domain.Ad
		var description string
		var pretendentID sql.NullInt64
		var publishedAt sql.NullTime
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&ad.ID,
			&ad.Title,
			&ad.ChatId,
			&ad.MessageId,
			&description,
			&ad.Photo,
			&ad.Price,
			&pretendentID,
			&ad.Status,
			&publishedAt,
			&expiresAt,
			&ad.CreatedAt,
			&ad.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ad.Description = &description
		ad.PretendentID = adPretendentIDValue(pretendentID)
		ad.PublishedAt = adTimeValue(publishedAt)
		ad.ExpiresAt = adTimeValue(expiresAt)
		ads = append(ads, ad)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ads, nil
}
