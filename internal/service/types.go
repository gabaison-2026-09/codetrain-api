package service

import "github.com/gabaison-2026-09/codetrain-core/pkg/domain"

// UserWithProgress は /v1/me が返す内容。
type UserWithProgress struct {
	User     domain.User     `json:"user"`
	Progress domain.Progress `json:"progress"`
}
