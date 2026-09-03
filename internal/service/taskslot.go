package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
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

// DeleteSlot は認証ユーザーの指定されたタスクスロットを削除する。
func (s *TaskSlot) DeleteSlot(ctx context.Context, externalID string, slotNo int) error {
	if slotNo < 1 || slotNo > 5 {
		return ErrTaskSlotNoInvalid
	}

	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteUserTask(ctx, userID, slotNo); errors.Is(err, repository.ErrNotFound) {
		return ErrTaskSlotNoInvalid
	} else if err != nil {
		return err
	}
	return nil
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

// SetSlot は認証ユーザーのタスクスロットを新規作成または更新する。
func (s *TaskSlot) SetSlot(ctx context.Context, externalID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
	if slot.SlotNo < 1 || slot.SlotNo > 5 {
		return domain.TaskConfig{}, ErrTaskSlotNoInvalid
	}

	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return domain.TaskConfig{}, err
	}

	exists, err := s.options.OptionExists(ctx, slot.QuestionType, slot.Language, slot.Difficulty)
	if err != nil {
		return domain.TaskConfig{}, err
	}
	if !exists {
		return domain.TaskConfig{}, ErrTaskSlotOptionInvalid
	}

	return s.options.UpsertUserTask(ctx, userID, slot)
}
