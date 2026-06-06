package postgres

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"smartbid-backend/internal/domain"
)

func TestPublishAdQueryStoresTimerAndRequiresCreatedStatus(t *testing.T) {
	publishedAt := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	query, args, err := publishAdQuery(domain.PublishAdUpdate{
		AdID:        "ad-id",
		PublishedAt: publishedAt,
		ExpiresAt:   publishedAt.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(query, `"published_at"=$`) || !strings.Contains(query, `"expires_at"=$`) {
		t.Fatalf("expected timer fields, got query: %s", query)
	}
	if !strings.Contains(query, `(select id from ad_status where slug = $`) {
		t.Fatalf("expected created-state guard, got query: %s", query)
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %#v", len(args), args)
	}
}

func TestCreateAdQueryBuildsScalarStatusSubquery(t *testing.T) {
	query, args, err := createAdQuery(domain.CreateAdInput{
		Title:     "title",
		ChatId:    1,
		MessageId: 2,
		Price:     100,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(query, `(select id from ad_status where slug = $`) {
		t.Fatalf("expected scalar status subquery, got query: %s", query)
	}
	if strings.Contains(query, "SELECT * FROM") {
		t.Fatalf("expected subquery to be used as a value, got query: %s", query)
	}
	if len(args) != 8 {
		t.Fatalf("expected 8 query args, got %d: %#v", len(args), args)
	}
	if args[1] != "" {
		t.Fatalf("expected empty description arg, got %#v", args[1])
	}
}

func TestUpdateAdPriceQueryStoresFinalPriceAndPretendent(t *testing.T) {
	query, args, err := updateAdPriceQuery(domain.UpdateAdPriceInput{
		AdID:         "ad-id",
		Price:        105,
		PretendentID: 42,
		Now:          time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(query, `final_price * 105`) {
		t.Fatalf("expected final_price calculation to stay out of SQL, got query: %s", query)
	}
	if !strings.Contains(query, `"final_price"=$`) {
		t.Fatalf("expected final_price to be set from an argument, got query: %s", query)
	}
	if !strings.Contains(query, `"pretendent_id"=$`) {
		t.Fatalf("expected pretendent_id to be updated, got query: %s", query)
	}
	if !strings.Contains(query, `RETURNING id::text, "final_price"`) {
		t.Fatalf("expected only ad id and final_price to be returned, got query: %s", query)
	}
	if !strings.Contains(query, `"expires_at" >`) || !strings.Contains(query, `(select id from ad_status where slug = $`) {
		t.Fatalf("expected active-ad guards, got query: %s", query)
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 query args, got %d: %#v", len(args), args)
	}
	if fmt.Sprint(args[0]) != "105" {
		t.Fatalf("expected final_price arg, got %#v", args[0])
	}
	if fmt.Sprint(args[1]) != "42" {
		t.Fatalf("expected pretendent_id arg, got %#v", args[1])
	}
	if args[2] != "ad-id" {
		t.Fatalf("expected ad id arg, got %#v", args[2])
	}
}

func TestUpdateAdStatusQueryUsesProvidedStatus(t *testing.T) {
	query, args, err := updateAdStatusQuery("ad-id", domain.AdStatusPublished)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(query, `(select id from ad_status where slug = $`) {
		t.Fatalf("expected scalar status subquery, got query: %s", query)
	}
	if !strings.Contains(query, `"updated_at"=now()`) {
		t.Fatalf("expected updated_at to be updated, got query: %s", query)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 query args, got %d: %#v", len(args), args)
	}
	if fmt.Sprint(args[0]) != string(domain.AdStatusPublished) {
		t.Fatalf("expected published status arg, got %#v", args[0])
	}
	if args[1] != "ad-id" {
		t.Fatalf("expected ad id arg, got %#v", args[1])
	}
}
