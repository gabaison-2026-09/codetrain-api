package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type fakeTaskSlotRepo struct {
	list func(context.Context, string) ([]domain.TaskConfig, error)
}

func (f fakeTaskSlotRepo) ListUserTasks(ctx context.Context, userID string) ([]domain.TaskConfig, error) {
	return f.list(ctx, userID)
}

func TestTaskSlotListSlots(t *testing.T) {
	var gotUserID string
	difficulty := 3
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeTaskSlotRepo{
		list: func(_ context.Context, userID string) ([]domain.TaskConfig, error) {
			gotUserID = userID
			return []domain.TaskConfig{
				{SlotNo: 1, QuestionType: domain.QuestionTypeCodeReading, Language: "typescript"},
				{SlotNo: 2, QuestionType: domain.QuestionTypeOutputPrediction, Difficulty: &difficulty},
			}, nil
		},
	})

	got, err := svc.ListSlots(context.Background(), "seed-user-01")
	if err != nil {
		t.Fatalf("ListSlots: %v", err)
	}
	if gotUserID != "u1" {
		t.Fatalf("repository userID = %q, want %q", gotUserID, "u1")
	}
	if len(got) != 2 || got[1].Difficulty == nil || *got[1].Difficulty != 3 {
		t.Fatalf("slots = %+v", got)
	}
}

func TestTaskSlotListSlotsEmptyIsArray(t *testing.T) {
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeTaskSlotRepo{
		list: func(context.Context, string) ([]domain.TaskConfig, error) { return nil, nil },
	})

	got, err := svc.ListSlots(context.Background(), "seed-user-01")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("slots = %#v, err = %v", got, err)
	}
}

func TestTaskSlotListSlotsResolvesUserError(t *testing.T) {
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{}, domain.Progress{}, errors.New("DB障害")
		},
	}, fakeTaskSlotRepo{
		list: func(context.Context, string) ([]domain.TaskConfig, error) { t.Fatal("not called"); return nil, nil },
	})

	if _, err := svc.ListSlots(context.Background(), "seed-user-01"); err == nil {
		t.Fatal("expected error")
	}
}
