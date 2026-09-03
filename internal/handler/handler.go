// Package handler は配信 API の HTTP ハンドラを持つ。
//
// この層の責務は HTTP の関心だけ —— リクエストからの値の取り出し、
// service 層の呼び出し、エラーからステータスコードへの変換、JSON 化。
// ビジネスロジックは service 層に置き、ここには書かない。
//
// 今回のスコープはローカル環境が立ち上がったことを確認できる最小限
// （/healthz・/v1/skills・/v1/me）。MVP ループの実装は Phase 2 で追加する。
package handler

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// 依存する service は消費側であるこの層で interface として宣言する。
// テストではフェイクを差し込み、DB も Echo サーバも起動せずに検証する。

type HealthChecker interface {
	Check(ctx context.Context) error
}

type SkillLister interface {
	List(ctx context.Context) ([]domain.Skill, error)
}

type UserFinder interface {
	Me(ctx context.Context, externalID string) (service.UserWithProgress, error)
}

type Handler struct {
	health HealthChecker
	skills SkillLister
	users  UserFinder
}

// Deps は Handler が必要とする service 群。service を増やす際はここにフィールドを
// 足すだけで済み、既存の New 呼び出し（テスト含む）を壊さない。
type Deps struct {
	Health HealthChecker
	Skills SkillLister
	Users  UserFinder
}

func New(deps Deps) *Handler {
	return &Handler{health: deps.Health, skills: deps.Skills, users: deps.Users}
}
