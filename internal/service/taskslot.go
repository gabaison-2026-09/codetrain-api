package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// TaskSlot はユーザーのタスクスロット設定を扱うユースケース。
type TaskSlot struct {
	userResolver
	repo    TaskSlotRepository
	options TaskOptionRepository
}

func NewTaskSlot(users UserRepository, repo TaskSlotRepository) *TaskSlot {
	return &TaskSlot{userResolver: userResolver{repo: users}, repo: repo}
}

// ListSlots は認証済みユーザーのタスクスロット設定を slot_no 順で返す。
func (s *TaskSlot) ListSlots(ctx context.Context, externalID string) ([]domain.TaskConfig, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	slots, err := s.repo.ListUserTasks(ctx, userID)
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = []domain.TaskConfig{}
	}
	return slots, nil
}

// TaskSlot はタスクスロットの候補を扱うユースケース。
// List は認証ユーザーの存在を確認してから、選択可能な候補を返す。
func (s *TaskSlot) List(ctx context.Context, externalID string) ([]domain.TaskOption, error) {
	if _, err := s.resolveUserID(ctx, externalID); err != nil {
		return nil, err
	}

	options, err := s.options.ListTaskOptions(ctx)
	if err != nil {
		return nil, err
	}
	if options == nil {
		options = []domain.TaskOption{}
	}
	return options, nil
}

// NewTaskOptions はタスク候補取得用のサービスを作る。
func NewTaskOptions(repo TaskOptionRepository) *TaskSlot {
	return &TaskSlot{userResolver: userResolver{repo: repo}, options: repo}
}
