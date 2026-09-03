package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// User はユーザー情報のユースケース。
type User struct {
	userResolver
}

func NewUser(repo UserRepository) *User { return &User{userResolver{repo: repo}} }

// Me は認証済みユーザーの sub（external_id）からユーザーと進捗を返す。
// 該当ユーザーがいない場合は ErrUserNotFound を返す。
func (s *User) Me(ctx context.Context, externalID string) (UserWithProgress, error) {
	user, progress, err := s.resolveUser(ctx, externalID)
	if err != nil {
		return UserWithProgress{}, err
	}
	return UserWithProgress{User: user, Progress: progress}, nil
}

// Create は初回ログイン時の JIT プロビジョニング。
// app_user と user_progress を作成し、GET /v1/me と同形を返す。
// 既に同一 external_id が存在する場合は ErrUserAlreadyProvisioned を返す。
func (s *User) Create(ctx context.Context, externalID string, in CreateUserInput) (UserWithProgress, error) {
	user, progress, err := s.repo.InsertUser(ctx, externalID, in.DisplayName, in.AvatarURL)
	if errors.Is(err, repository.ErrAlreadyExists) {
		return UserWithProgress{}, ErrUserAlreadyProvisioned
	}
	if err != nil {
		return UserWithProgress{}, err
	}
	return UserWithProgress{User: user, Progress: progress}, nil
}
