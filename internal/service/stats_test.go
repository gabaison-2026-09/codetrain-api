package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

type fakeMeStatsRepo struct {
	list func(ctx context.Context, userID string) ([]domain.TypeStat, error)
}

func (f fakeMeStatsRepo) ListTypeStats(ctx context.Context, userID string) ([]domain.TypeStat, error) {
	return f.list(ctx, userID)
}

func TestMeStatsComputesAccuracy(t *testing.T) {
	var gotUserID string
	svc := NewMeStats(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeMeStatsRepo{
		list: func(_ context.Context, userID string) ([]domain.TypeStat, error) {
			gotUserID = userID
			return []domain.TypeStat{
				{QuestionType: domain.QuestionTypeCodeReading, Language: "typescript", Attempts: 42, Corrects: 35},
				{QuestionType: domain.QuestionTypeOutputPrediction, Language: "", Attempts: 0, Corrects: 0},
			}, nil
		},
	})

	got, err := svc.Stats(context.Background(), "seed-user-03")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if gotUserID != "u1" {
		t.Fatalf("repository に渡した userID = %q, want %q", gotUserID, "u1")
	}
	if math.Abs(got[0].Accuracy-35.0/42.0) > 1e-9 {
		t.Errorf("accuracy = %v, want %v", got[0].Accuracy, 35.0/42.0)
	}
	if got[1].Accuracy != 0 {
		t.Errorf("attempts=0 の accuracy = %v, want 0 (ゼロ除算しない)", got[1].Accuracy)
	}
}

func TestMeStatsUserNotFoundTranslation(t *testing.T) {
	svc := NewMeStats(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{}, domain.Progress{}, repository.ErrNotFound
		},
	}, fakeMeStatsRepo{
		list: func(context.Context, string) ([]domain.TypeStat, error) {
			t.Fatal("ユーザー未解決なのに repository が呼ばれた")
			return nil, nil
		},
	})

	_, err := svc.Stats(context.Background(), "no-such-user")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
	if errors.Is(err, repository.ErrNotFound) {
		t.Errorf("repository.ErrNotFound が翻訳されず漏れている")
	}
}

func TestMeStatsEmptyIsArray(t *testing.T) {
	svc := NewMeStats(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeMeStatsRepo{
		list: func(context.Context, string) ([]domain.TypeStat, error) { return nil, nil },
	})

	got, err := svc.Stats(context.Background(), "seed-user-03")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("stats = %#v, err = %v", got, err)
	}
}

func TestMeStatsRepoErrorPassesThrough(t *testing.T) {
	sentinel := errors.New("DB 障害")
	svc := NewMeStats(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeMeStatsRepo{
		list: func(context.Context, string) ([]domain.TypeStat, error) { return nil, sentinel },
	})

	_, err := svc.Stats(context.Background(), "seed-user-03")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
