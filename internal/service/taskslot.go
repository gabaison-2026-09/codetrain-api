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
