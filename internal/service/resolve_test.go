package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// userResolver の契約:
//
//	sub（external_id）から app_user 行を解決する。
//	repository.ErrNotFound は ErrUserNotFound に翻訳する（repository のエラーを漏らさない）。
//	それ以外のエラーはそのまま伝播する。
func TestUserResolver(t *testing.T) {
	t.Run("resolveUser がユーザーと進捗を返す", func(t *testing.T) {
		var gotSub string
		r := userResolver{repo: fakeUserRepo{
			find: func(_ context.Context, sub string) (domain.User, domain.Progress, error) {
				gotSub = sub
				return domain.User{ID: "u1", ExternalID: sub},
					domain.Progress{XP: 42},
					nil
			},
		}}

		user, progress, err := r.resolveUser(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("resolveUser: %v", err)
		}
		if gotSub != "seed-user-01" {
			t.Errorf("repository に渡した sub = %q, want %q", gotSub, "seed-user-01")
		}
		if user.ID != "u1" || progress.XP != 42 {
			t.Errorf("user=%+v progress=%+v", user, progress)
		}
	})

	t.Run("resolveUserID が app_user.id を返す", func(t *testing.T) {
		r := userResolver{repo: fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
		}}

		id, err := r.resolveUserID(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("resolveUserID: %v", err)
		}
		if id != "u1" {
			t.Errorf("id = %q, want %q", id, "u1")
		}
	})

	t.Run("ErrNotFound を ErrUserNotFound に翻訳する", func(t *testing.T) {
		r := userResolver{repo: fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
		}}

		for _, tc := range []struct {
			name string
			call func() error
		}{
			{"resolveUser", func() error {
				_, _, err := r.resolveUser(context.Background(), "no-such-user")
				return err
			}},
			{"resolveUserID", func() error {
				_, err := r.resolveUserID(context.Background(), "no-such-user")
				return err
			}},
		} {
			err := tc.call()
			if !errors.Is(err, ErrUserNotFound) {
				t.Errorf("%s: err = %v, want %v", tc.name, err, ErrUserNotFound)
			}
			if errors.Is(err, repository.ErrNotFound) {
				t.Errorf("%s: repository のエラーがそのまま漏れている", tc.name)
			}
		}
	})

	t.Run("その他のエラーは伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		r := userResolver{repo: fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, wantErr
			},
		}}

		if _, _, err := r.resolveUser(context.Background(), "seed-user-01"); !errors.Is(err, wantErr) {
			t.Errorf("resolveUser: err = %v, want %v", err, wantErr)
		}
		if _, err := r.resolveUserID(context.Background(), "seed-user-01"); !errors.Is(err, wantErr) {
			t.Errorf("resolveUserID: err = %v, want %v", err, wantErr)
		}
	})
}
