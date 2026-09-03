package service

import "github.com/gabaison-2026-09/codetrain-core/pkg/domain"

// UserWithProgress は /v1/me が返す内容。
type UserWithProgress struct {
	User     domain.User     `json:"user"`
	Progress domain.Progress `json:"progress"`
}

// CreateUserInput は POST /v1/me の作成内容。
// DisplayName は必須。AvatarURL は任意（未指定なら nil）。
type CreateUserInput struct {
	DisplayName string
	AvatarURL   *string
}
