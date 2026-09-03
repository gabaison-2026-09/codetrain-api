package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// TaskSlot はタスクスロットの候補を扱うユースケース。
type TaskSlot struct {
	userResolver
	repo TaskOptionRepository
}

func NewTaskSlot(repo TaskOptionRepository) *TaskSlot {
	return &TaskSlot{userResolver: userResolver{repo: repo}, repo: repo}
}

// List は認証ユーザーの存在を確認してから、選択可能な候補を返す。
func (s *TaskSlot) List(ctx context.Context, externalID string) ([]domain.TaskOption, error) {
	if _, err := s.resolveUserID(ctx, externalID); err != nil {
		return nil, err
	}

	options, err := s.repo.ListTaskOptions(ctx)
	if err != nil {
		return nil, err
	}
	if options == nil {
		options = []domain.TaskOption{}
	}
	return options, nil
}

// SetSlot は認証ユーザーのタスクスロットを新規作成または更新する。
func (s *TaskSlot) SetSlot(ctx context.Context, externalID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
	if slot.SlotNo < 1 || slot.SlotNo > 5 {
		return domain.TaskConfig{}, ErrTaskSlotNoInvalid
	}

	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return domain.TaskConfig{}, err
	}

	exists, err := s.repo.OptionExists(ctx, slot.QuestionType, slot.Language, slot.Difficulty)
	if err != nil {
		return domain.TaskConfig{}, err
	}
	if !exists {
		return domain.TaskConfig{}, ErrTaskSlotOptionInvalid
	}

	return s.repo.UpsertUserTask(ctx, userID, slot)
}
