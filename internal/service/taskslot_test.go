package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

type fakeTaskOptionRepo struct {
	find func(context.Context, string) (domain.User, domain.Progress, error)
	list func(context.Context) ([]domain.TaskOption, error)
}

func (f fakeTaskOptionRepo) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return f.find(ctx, externalID)
}

func (f fakeTaskOptionRepo) ListTaskOptions(ctx context.Context) ([]domain.TaskOption, error) {
	return f.list(ctx)
}

func TestTaskSlotList(t *testing.T) {
	t.Run("ユーザー確認後に候補を返す", func(t *testing.T) {
		var gotExternalID string
		repo := fakeTaskOptionRepo{
			find: func(_ context.Context, externalID string) (domain.User, domain.Progress, error) {
				gotExternalID = externalID
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) {
				return []domain.TaskOption{{QuestionType: domain.QuestionTypeCodeReading, Language: "typescript", Difficulty: 1}}, nil
			},
		}

		got, err := NewTaskSlot(repo).List(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if gotExternalID != "seed-user-01" {
			t.Errorf("external_id = %q, want seed-user-01", gotExternalID)
		}
		if len(got) != 1 || got[0].Language != "typescript" {
			t.Errorf("options = %+v", got)
		}
	})

	t.Run("候補0件は nil ではなく空配列を返す", func(t *testing.T) {
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) { return nil, nil },
		}

		got, err := NewTaskSlot(repo).List(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("options = %#v, want non-nil empty slice", got)
		}
	})

	t.Run("未登録ユーザーは候補を取得せず ErrUserNotFound", func(t *testing.T) {
		listCalled := false
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
			list: func(context.Context) ([]domain.TaskOption, error) {
				listCalled = true
				return nil, nil
			},
		}

		_, err := NewTaskSlot(repo).List(context.Background(), "no-such-user")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
		if listCalled {
			t.Error("ユーザー確認失敗後に ListTaskOptions が呼ばれた")
		}
	})

	t.Run("候補取得エラーを伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) { return nil, wantErr },
		}

		_, err := NewTaskSlot(repo).List(context.Background(), "seed-user-01")
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
