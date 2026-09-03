package service

import (
	"context"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type fakeSRSDueRepo struct {
	list func(context.Context, string, int) ([]domain.SRSDueItem, error)
}

func (f fakeSRSDueRepo) ListDue(ctx context.Context, userID string, limit int) ([]domain.SRSDueItem, error) {
	return f.list(ctx, userID, limit)
}

func TestSRSListDue(t *testing.T) {
	var gotUserID string
	var gotLimit int
	svc := NewSRS(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeSRSDueRepo{
		list: func(_ context.Context, userID string, limit int) ([]domain.SRSDueItem, error) {
			gotUserID, gotLimit = userID, limit
			return []domain.SRSDueItem{{ID: "q1", DueOn: "2026-09-01"}}, nil
		},
	})

	got, err := svc.ListDue(context.Background(), "seed-user-03", 20)
	if err != nil {
		t.Fatalf("ListDue: %v", err)
	}
	if gotUserID != "u1" || gotLimit != 20 {
		t.Fatalf("repository args = %q, %d", gotUserID, gotLimit)
	}
	if len(got) != 1 || got[0].DueOn != "2026-09-01" {
		t.Fatalf("items = %+v", got)
	}
}

func TestSRSListDueEmptyIsArray(t *testing.T) {
	svc := NewSRS(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeSRSDueRepo{
		list: func(context.Context, string, int) ([]domain.SRSDueItem, error) { return nil, nil },
	})
	got, err := svc.ListDue(context.Background(), "seed-user-03", 20)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("items = %#v, err = %v", got, err)
	}
}
