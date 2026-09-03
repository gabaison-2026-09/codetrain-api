// Package handler は配信 API の HTTP ハンドラを持つ。
//
// この層の責務は HTTP の関心だけ —— リクエストからの値の取り出し、
// service 層の呼び出し、エラーからステータスコードへの変換、JSON 化。
// ビジネスロジックは service 層に置き、ここには書かない。
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
	Create(ctx context.Context, externalID string, in service.CreateUserInput) (service.UserWithProgress, error)
	Update(
		ctx context.Context,
		externalID string,
		patch service.UserPatch,
	) (domain.User, error)
}

type QuestionLister interface {
	List(ctx context.Context, externalID string, params service.QuestionSearchParams) (service.QuestionList, error)
	GetForUser(ctx context.Context, externalID, questionID string) (domain.QuestionDetail, error)
}

type AdminQuestionLister interface {
	List(ctx context.Context, params service.AdminQuestionSearchParams) (service.AdminQuestionList, error)
}

type ReviewQueueLister interface {
	List(ctx context.Context, params service.ReviewQueueParams) (service.ReviewQueueList, error)
}

type AttemptSubmitter interface {
	Submit(ctx context.Context, externalID, questionID string, in service.SubmitAttemptInput) (domain.AttemptResult, error)
}

type SRSDueLister interface {
	ListDue(ctx context.Context, externalID string, limit int) ([]domain.SRSDueItem, error)
}

type TaskSlotLister interface {
	ListSlots(ctx context.Context, externalID string) ([]domain.TaskConfig, error)
	DeleteSlot(ctx context.Context, externalID string, slotNo int) error
}

type TaskOptionLister interface {
	List(ctx context.Context, externalID string) ([]domain.TaskOption, error)
	SetSlot(ctx context.Context, externalID string, slot domain.TaskConfig) (domain.TaskConfig, error)
}

type CalendarGetter interface {
	Get(ctx context.Context, externalID, from, to string) (domain.Calendar, error)
}
type HomeGetter interface {
	Get(ctx context.Context, externalID string) (domain.Home, error)
}

type Handler struct {
	health      HealthChecker
	skills      SkillLister
	users       UserFinder
	questions   QuestionLister
	admin       AdminQuestionLister
	reviewQueue ReviewQueueLister
	srs         SRSDueLister
	taskSlots   TaskSlotLister
	taskOptions TaskOptionLister
	calendar    CalendarGetter
	attempts    AttemptSubmitter
	home        HomeGetter
}

// Deps は Handler が必要とする service 群。service を増やす際はここにフィールドを
// 足すだけで済み、既存の New 呼び出し（テスト含む）を壊さない。
type Deps struct {
	Health      HealthChecker
	Skills      SkillLister
	Users       UserFinder
	Questions   QuestionLister
	Admin       AdminQuestionLister
	ReviewQueue ReviewQueueLister
	SRS         SRSDueLister
	TaskSlots   TaskSlotLister
	TaskOptions TaskOptionLister
	Calendar    CalendarGetter
	Attempts    AttemptSubmitter
	Home        HomeGetter
}

func New(deps Deps) *Handler {
	return &Handler{
		health:      deps.Health,
		skills:      deps.Skills,
		users:       deps.Users,
		questions:   deps.Questions,
		admin:       deps.Admin,
		reviewQueue: deps.ReviewQueue,
		srs:         deps.SRS,
		taskSlots:   deps.TaskSlots,
		taskOptions: deps.TaskOptions,
		calendar:    deps.Calendar,
		attempts:    deps.Attempts,
		home:        deps.Home,
	}
}
