package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// TaskSlot はユーザーのタスクスロット設定を扱うユースケース。
type TaskSlot struct {
	userResolver
	repo TaskSlotRepository
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
