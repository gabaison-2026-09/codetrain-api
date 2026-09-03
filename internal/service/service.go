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

// UserPatch はユーザープロフィールの部分更新内容。
// nilのフィールドは更新しない。
type UserPatch struct {
	DisplayName *string
	AvatarURL   *string
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
	UpdateUser(
		ctx context.Context,
		externalID string,
		displayName *string,
		avatarURL *string,
	) (domain.User, error)
}

// QuestionRepository は published 問題の検索・取得。行を返すだけで、ページングは service が行う。
type QuestionRepository interface {
	SearchQuestions(ctx context.Context, userID string, q domain.QuestionSearch) ([]domain.QuestionSummary, error)
	// FindPublishedByID は status=published の問題を1件返す。answered は
	// 当該ユーザーの attempt が存在するかどうか。該当行がなければ repository.ErrNotFound。
	FindPublishedByID(ctx context.Context, userID, questionID string) (domain.Question, bool, error)
}

// SRSRepository は復習期限の問題を取得する。
type SRSRepository interface {
	ListDue(ctx context.Context, userID string, limit int) ([]domain.SRSDueItem, error)
}

// TaskSlotRepository はユーザーのタスクスロット設定を取得する。
type TaskSlotRepository interface {
	ListUserTasks(ctx context.Context, userID string) ([]domain.TaskConfig, error)
}

// TaskOptionRepository は認証ユーザーの解決とタスク候補の取得。
type TaskOptionRepository interface {
	UserLookupRepository
	ListTaskOptions(ctx context.Context) ([]domain.TaskOption, error)
}
