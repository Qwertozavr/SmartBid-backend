package postgres

import (
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
