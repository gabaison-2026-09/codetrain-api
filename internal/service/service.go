// Package service は配信 API のユースケースを持つ。
//
// HTTP（Echo）も SQL（pgx）も知らない層で、Phase 2 の出題組立・回答記録・SRS の
// ロジックはここに置く（DESIGN.md §8 Phase 2）。
//
// 依存する repository は **この層で interface として定義する**（Go の作法どおり、
// 消費側が必要なメソッドだけを宣言する）。実体は internal/repository が満たし、
// 配線は cmd/api/main.go が行う。テストではフェイクを差し込む。
package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// HealthRepository は /healthz が使う DB 疎通確認。
type HealthRepository interface {
	Ping(ctx context.Context) error
}

// SkillRepository はスキルツリーの取得。行を返すだけで、組み立ては service が行う。
type SkillRepository interface {
	ListSkills(ctx context.Context) ([]domain.Skill, error)
	ListSkillNodes(ctx context.Context) ([]domain.SkillNode, error)
}

// UserLookupRepository は認証基盤上の識別子からユーザーを解決する。
type UserLookupRepository interface {
	FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error)
}

// UserRepository はユーザーと進捗の取得・作成。
type UserRepository interface {
	UserLookupRepository
	InsertUser(ctx context.Context, externalID, displayName string, avatarURL *string) (domain.User, domain.Progress, error)
}

// TaskOptionRepository は認証ユーザーの解決とタスク候補の取得。
type TaskOptionRepository interface {
	UserLookupRepository
	ListTaskOptions(ctx context.Context) ([]domain.TaskOption, error)
}
