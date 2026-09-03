package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ListTaskOptions は published な問題に対応する選択可能な組み合わせを返す。
func (p *Postgres) ListTaskOptions(ctx context.Context) ([]domain.TaskOption, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT question_type, language, difficulty
		  FROM available_task_option
		 ORDER BY question_type, language, difficulty`)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.TaskOption, error) {
		var option domain.TaskOption
		err := r.Scan(&option.QuestionType, &option.Language, &option.Difficulty)
		return option, err
	})
}
