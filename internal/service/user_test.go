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
	find          func(ctx context.Context, externalID string) (domain.User, domain.Progress, error)
	insert        func(ctx context.Context, externalID, displayName string, email *string, avatarURL *string) (domain.User, domain.Progress, error)
	update        func(
		ctx context.Context,
		externalID string,
		displayName *string,
		avatarURL *string,
	) (domain.User, error)
	backfillEmail func(ctx context.Context, externalID, email string) error
}

func (f fakeUserRepo) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return f.find(ctx, externalID)
}
func (f fakeUserRepo) UpdateUser(
	ctx context.Context,
	externalID string,
	displayName *string,
	avatarURL *string,
) (domain.User, error) {
	return f.update(ctx, externalID, displayName, avatarURL)
}

func (f fakeUserRepo) InsertUser(ctx context.Context, externalID, displayName string, email *string, avatarURL *string) (domain.User, domain.Progress, error) {
	return f.insert(ctx, externalID, displayName, email, avatarURL)
}

func (f fakeUserRepo) BackfillEmail(ctx context.Context, externalID, email string) error {
	if f.backfillEmail != nil {
		return f.backfillEmail(ctx, externalID, email)
	}
	return nil
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

		got, err := NewUser(repo).Me(context.Background(), "seed-user-01", "")
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

		_, err := NewUser(repo).Me(context.Background(), "no-such-user", "")
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

		_, err := NewUser(repo).Me(context.Background(), "seed-user-01", "")
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
		var gotEmail *string
		var gotAvatar *string
		repo := fakeUserRepo{
			insert: func(_ context.Context, externalID, displayName string, email *string, avatarURL *string) (domain.User, domain.Progress, error) {
				gotExt, gotName, gotEmail, gotAvatar = externalID, displayName, email, avatarURL
				return domain.User{ID: "u1", ExternalID: externalID, DisplayName: displayName},
					domain.Progress{XP: 0, Level: 1, StreakDays: 0, Hearts: 5},
					nil
			},
		}

		avatar := "https://example.com/a.jpg"
		emailStr := "test@example.com"
		got, err := NewUser(repo).Create(context.Background(), "brand-new-user", CreateUserInput{
			DisplayName: "新規太郎",
			Email:       &emailStr,
			AvatarURL:   &avatar,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if gotExt != "brand-new-user" || gotName != "新規太郎" {
			t.Errorf("InsertUser args = (%q, %q), want (brand-new-user, 新規太郎)", gotExt, gotName)
		}
		if gotEmail == nil || *gotEmail != emailStr {
			t.Errorf("email = %v, want %q", gotEmail, emailStr)
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
			insert: func(context.Context, string, string, *string, *string) (domain.User, domain.Progress, error) {
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
			insert: func(context.Context, string, string, *string, *string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, wantErr
			},
		}

		_, err := NewUser(repo).Create(
			context.Background(),
			"brand-new-user",
			CreateUserInput{DisplayName: "新規"},
		)
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestUserUpdate(t *testing.T) {
	t.Run("指定されたプロフィールを更新して返す", func(t *testing.T) {
		displayName := "変更後"
		avatarURL := "https://example.com/new-avatar.jpg"

		var gotExternalID string
		var gotDisplayName *string
		var gotAvatarURL *string

		repo := fakeUserRepo{
			find: func(
				context.Context,
				string,
			) (domain.User, domain.Progress, error) {
				return domain.User{
					ID:          "u1",
					ExternalID:  "seed-user-01",
					DisplayName: "変更前",
				}, domain.Progress{}, nil
			},
			update: func(
				_ context.Context,
				externalID string,
				displayName *string,
				avatarURL *string,
			) (domain.User, error) {
				gotExternalID = externalID
				gotDisplayName = displayName
				gotAvatarURL = avatarURL

				return domain.User{
					ID:          "u1",
					ExternalID:  externalID,
					DisplayName: *displayName,
					AvatarURL:   *avatarURL,
				}, nil
			},
		}

		got, err := NewUser(repo).Update(
			context.Background(),
			"seed-user-01",
			UserPatch{
				DisplayName: &displayName,
				AvatarURL:   &avatarURL,
			},
		)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}

		if gotExternalID != "seed-user-01" {
			t.Errorf(
				"externalID = %q, want %q",
				gotExternalID,
				"seed-user-01",
			)
		}
		if gotDisplayName == nil || *gotDisplayName != "変更後" {
			t.Errorf("displayName = %v, want %q", gotDisplayName, "変更後")
		}
		if gotAvatarURL == nil || *gotAvatarURL != avatarURL {
			t.Errorf("avatarURL = %v, want %q", gotAvatarURL, avatarURL)
		}
		if got.DisplayName != "変更後" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "変更後")
		}
		if got.AvatarURL != avatarURL {
			t.Errorf("AvatarURL = %q, want %q", got.AvatarURL, avatarURL)
		}
	})

	t.Run("空パッチなら更新せず現在のユーザーを返す", func(t *testing.T) {
		updateCalled := false
		currentUser := domain.User{
			ID:          "u1",
			ExternalID:  "seed-user-01",
			DisplayName: "現在の名前",
			AvatarURL:   "https://example.com/current.jpg",
		}

		repo := fakeUserRepo{
			find: func(
				context.Context,
				string,
			) (domain.User, domain.Progress, error) {
				return currentUser, domain.Progress{}, nil
			},
			update: func(
				context.Context,
				string,
				*string,
				*string,
			) (domain.User, error) {
				updateCalled = true
				return domain.User{}, nil
			},
		}

		got, err := NewUser(repo).Update(
			context.Background(),
			"seed-user-01",
			UserPatch{},
		)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}

		if updateCalled {
			t.Error("空パッチでUpdateUserが呼ばれた")
		}
		if got.DisplayName != currentUser.DisplayName {
			t.Errorf(
				"DisplayName = %q, want %q",
				got.DisplayName,
				currentUser.DisplayName,
			)
		}
		if got.AvatarURL != currentUser.AvatarURL {
			t.Errorf(
				"AvatarURL = %q, want %q",
				got.AvatarURL,
				currentUser.AvatarURL,
			)
		}
	})

	t.Run("未登録ユーザーならErrUserNotFoundを返す", func(t *testing.T) {
		updateCalled := false

		repo := fakeUserRepo{
			find: func(
				context.Context,
				string,
			) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
			update: func(
				context.Context,
				string,
				*string,
				*string,
			) (domain.User, error) {
				updateCalled = true
				return domain.User{}, nil
			},
		}

		displayName := "変更後"
		_, err := NewUser(repo).Update(
			context.Background(),
			"no-such-user",
			UserPatch{DisplayName: &displayName},
		)

		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
		if updateCalled {
			t.Error("未登録ユーザーでUpdateUserが呼ばれた")
		}
	})

	t.Run("更新時のErrNotFoundをErrUserNotFoundに翻訳する", func(t *testing.T) {
		repo := fakeUserRepo{
			find: func(
				context.Context,
				string,
			) (domain.User, domain.Progress, error) {
				return domain.User{
					ID:         "u1",
					ExternalID: "seed-user-01",
				}, domain.Progress{}, nil
			},
			update: func(
				context.Context,
				string,
				*string,
				*string,
			) (domain.User, error) {
				return domain.User{}, repository.ErrNotFound
			},
		}

		displayName := "変更後"
		_, err := NewUser(repo).Update(
			context.Background(),
			"seed-user-01",
			UserPatch{DisplayName: &displayName},
		)

		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
		if errors.Is(err, repository.ErrNotFound) {
			t.Error("repositoryのエラーがそのまま漏れている")
		}
	})

	t.Run("更新時のその他のエラーは伝播する", func(t *testing.T) {
		wantErr := errors.New("DB障害")

		repo := fakeUserRepo{
			find: func(
				context.Context,
				string,
			) (domain.User, domain.Progress, error) {
				return domain.User{
					ID:         "u1",
					ExternalID: "seed-user-01",
				}, domain.Progress{}, nil
			},
			update: func(
				context.Context,
				string,
				*string,
				*string,
			) (domain.User, error) {
				return domain.User{}, wantErr
			},
		}

		displayName := "変更後"
		_, err := NewUser(repo).Update(
			context.Background(),
			"seed-user-01",
			UserPatch{DisplayName: &displayName},
		)

		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
