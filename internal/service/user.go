package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// User はユーザー情報のユースケース。
type User struct {
	repo UserRepository
}

func NewUser(repo UserRepository) *User { return &User{repo: repo} }

// Me は認証済みユーザーの sub（external_id）からユーザーと進捗を返す。
// 該当ユーザーがいない場合は ErrUserNotFound を返す。
func (s *User) Me(ctx context.Context, externalID string) (UserWithProgress, error) {
	user, progress, err := s.repo.FindUserByExternalID(ctx, externalID)
	if errors.Is(err, repository.ErrNotFound) {
		return UserWithProgress{}, ErrUserNotFound
	}
	if err != nil {
		return UserWithProgress{}, err
	}
	return UserWithProgress{User: user, Progress: progress}, nil
}
