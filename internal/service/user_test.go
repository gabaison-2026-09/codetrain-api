package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// fakeUserRepo は UserRepository のフェイク。
type fakeUserRepo struct {
	find func(ctx context.Context, externalID string) (domain.User, domain.Progress, error)
}

func (f fakeUserRepo) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return f.find(ctx, externalID)
}

// User.Me の契約:
//
//	repository の結果を UserWithProgress にまとめる。
//	repository.ErrNotFound は ErrUserNotFound に翻訳する（handler が 404 に変換できるように）。
//	それ以外のエラーはそのまま伝播する。
func TestUserMe(t *testing.T) {
	t.Run("ユーザーと進捗をまとめて返す", func(t *testing.T) {
		var gotExternalID string
		repo := fakeUserRepo{
			find: func(_ context.Context, externalID string) (domain.User, domain.Progress, error) {
				gotExternalID = externalID
				return domain.User{
					ID:          "u1",
					ExternalID:  externalID,
					DisplayName: "テスト",
					AvatarURL:   "https://example.com/avatar.jpg",
				},
					domain.Progress{XP: 120, StreakDays: 3, Hearts: 5},
					nil
			},
		}

		got, err := NewUser(repo).Me(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("Me: %v", err)
		}
		if gotExternalID != "seed-user-01" {
			t.Errorf("repository に渡した external_id = %q, want %q", gotExternalID, "seed-user-01")
		}
		if got.User.DisplayName != "テスト" {
			t.Errorf("User.DisplayName = %q, want %q", got.User.DisplayName, "テスト")
		}
		if got.User.AvatarURL != "https://example.com/avatar.jpg" {
			t.Errorf(
				"User.AvatarURL = %q, want %q",
				got.User.AvatarURL,
				"https://example.com/avatar.jpg",
			)
		}
		if got.Progress.XP != 120 {
			t.Errorf("Progress.XP = %d, want 120", got.Progress.XP)
		}
	})

	t.Run("ErrNotFound を ErrUserNotFound に翻訳する", func(t *testing.T) {
		repo := fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
		}

		_, err := NewUser(repo).Me(context.Background(), "no-such-user")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
		if errors.Is(err, repository.ErrNotFound) {
			t.Error("repository のエラーがそのまま漏れている")
		}
	})

	t.Run("その他のエラーは伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		repo := fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, wantErr
			},
		}

		_, err := NewUser(repo).Me(context.Background(), "seed-user-01")
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
