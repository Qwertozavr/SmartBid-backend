package postgres

import (
	"fmt"
	"strings"
	"testing"

	"smartbid-backend/internal/domain"
)

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
	if len(args) != 3 {
		t.Fatalf("expected 3 query args, got %d: %#v", len(args), args)
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
