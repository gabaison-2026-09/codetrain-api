package service

import (
	"context"
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
