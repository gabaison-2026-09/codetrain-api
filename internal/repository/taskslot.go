package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ListUserTasks はユーザーのタスクスロット設定を slot_no 順で返す。
func (p *Postgres) ListUserTasks(ctx context.Context, userID string) ([]domain.TaskConfig, error) {
	rows, err := p.pool.Query(ctx, `
SELECT slot_no, question_type, language, difficulty
  FROM user_task
 WHERE user_id = $1
 ORDER BY slot_no`, userID)
	if err != nil {
		return nil, err
	}

	slots, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.TaskConfig, error) {
		var slot domain.TaskConfig
		err := r.Scan(&slot.SlotNo, &slot.QuestionType, &slot.Language, &slot.Difficulty)
		return slot, err
	})
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = []domain.TaskConfig{}
	}
	return slots, nil
}
