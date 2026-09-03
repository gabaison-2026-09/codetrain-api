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

// OptionExists は指定されたタスク条件が published な問題バンクに存在するか返す。
// difficulty が nil の場合は種別と言語の組み合わせだけを検証する。
func (p *Postgres) OptionExists(ctx context.Context, questionType domain.QuestionType, language string, difficulty *int) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM available_task_option
			 WHERE question_type = $1
			   AND language = $2
			   AND ($3::smallint IS NULL OR difficulty = $3)
		)`, questionType, language, difficulty).Scan(&exists)
	return exists, err
}

// UpsertUserTask はユーザーとスロット番号の組をキーにタスク条件を upsert する。
func (p *Postgres) UpsertUserTask(ctx context.Context, userID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
	var saved domain.TaskConfig
	err := p.pool.QueryRow(ctx, `
		INSERT INTO user_task (user_id, slot_no, question_type, language, difficulty)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, slot_no) DO UPDATE SET
			question_type = EXCLUDED.question_type,
			language = EXCLUDED.language,
			difficulty = EXCLUDED.difficulty,
			updated_at = now()
		RETURNING slot_no, question_type, language, difficulty`,
		userID, slot.SlotNo, slot.QuestionType, slot.Language, slot.Difficulty,
	).Scan(&saved.SlotNo, &saved.QuestionType, &saved.Language, &saved.Difficulty)
	return saved, err
}

// DeleteUserTask はユーザーの指定されたタスクスロットを削除する。
// 削除対象が存在しない場合は ErrNotFound を返す。
func (p *Postgres) DeleteUserTask(ctx context.Context, userID string, slotNo int) error {
	result, err := p.pool.Exec(ctx, `
		DELETE FROM user_task
		 WHERE user_id = $1
		   AND slot_no = $2`, userID, slotNo)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
