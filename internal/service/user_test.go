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
	find   func(ctx context.Context, externalID string) (domain.User, domain.Progress, error)
	insert func(ctx context.Context, externalID, displayName string, avatarURL *string) (domain.User, domain.Progress, error)
}

func (f fakeUserRepo) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return f.find(ctx, externalID)
}

func (f fakeUserRepo) InsertUser(ctx context.Context, externalID, displayName string, avatarURL *string) (domain.User, domain.Progress, error) {
	return f.insert(ctx, externalID, displayName, avatarURL)
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
				return domain.User{ID: "u1", ExternalID: externalID, DisplayName: "テスト"},
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

// User.Create の契約:
//
//	repository.InsertUser の結果を UserWithProgress にまとめる。
//	repository.ErrAlreadyExists は ErrUserAlreadyProvisioned に翻訳する。
//	それ以外のエラーはそのまま伝播する。
func TestUserCreate(t *testing.T) {
	t.Run("ユーザーと初期進捗を返す", func(t *testing.T) {
		var gotExt, gotName string
		var gotAvatar *string
		repo := fakeUserRepo{
			insert: func(_ context.Context, externalID, displayName string, avatarURL *string) (domain.User, domain.Progress, error) {
				gotExt, gotName, gotAvatar = externalID, displayName, avatarURL
				return domain.User{ID: "u1", ExternalID: externalID, DisplayName: displayName},
					domain.Progress{XP: 0, Level: 1, StreakDays: 0, Hearts: 5},
					nil
			},
		}

		avatar := "https://example.com/a.jpg"
		got, err := NewUser(repo).Create(context.Background(), "brand-new-user", CreateUserInput{
			DisplayName: "新規太郎",
			AvatarURL:   &avatar,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if gotExt != "brand-new-user" || gotName != "新規太郎" {
			t.Errorf("InsertUser args = (%q, %q), want (brand-new-user, 新規太郎)", gotExt, gotName)
		}
		if gotAvatar == nil || *gotAvatar != avatar {
			t.Errorf("avatarURL = %v, want %q", gotAvatar, avatar)
		}
		if got.User.DisplayName != "新規太郎" || got.Progress.Hearts != 5 || got.Progress.Level != 1 {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("ErrAlreadyExists を ErrUserAlreadyProvisioned に翻訳する", func(t *testing.T) {
		repo := fakeUserRepo{
			insert: func(context.Context, string, string, *string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrAlreadyExists
			},
		}

		_, err := NewUser(repo).Create(context.Background(), "seed-user-01", CreateUserInput{DisplayName: "既存"})
		if !errors.Is(err, ErrUserAlreadyProvisioned) {
			t.Errorf("err = %v, want %v", err, ErrUserAlreadyProvisioned)
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			t.Error("repository のエラーがそのまま漏れている")
		}
	})

	t.Run("その他のエラーは伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		repo := fakeUserRepo{
			insert: func(context.Context, string, string, *string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, wantErr
			},
		}

		_, err := NewUser(repo).Create(context.Background(), "brand-new-user", CreateUserInput{DisplayName: "新規"})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
