package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// RecordAttempt は回答履歴と全ての派生状態を同一トランザクションで更新する。
func (p *Postgres) RecordAttempt(ctx context.Context, attempt domain.Attempt, question domain.Question, xpGained int) (domain.AttemptResult, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.AttemptResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit 後は no-op

	selectedKeys, err := json.Marshal(attempt.SelectedKeys)
	if err != nil {
		return domain.AttemptResult{}, err
	}
	err = tx.QueryRow(ctx, `
INSERT INTO attempt (user_id, question_id, selected_keys, is_correct, duration_ms)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, answered_at`, attempt.UserID, attempt.QuestionID, selectedKeys,
		attempt.IsCorrect, attempt.DurationMS).Scan(&attempt.ID, &attempt.AnsweredAt)
	if err != nil {
		return domain.AttemptResult{}, err
	}

	correctIncrement := 0
	if attempt.IsCorrect {
		correctIncrement = 1
	}
	_, err = tx.Exec(ctx, `
INSERT INTO user_type_stat
  (user_id, question_type, language, attempts, corrects, last_difficulty)
VALUES ($1, $2, $3, 1, $4, $5)
ON CONFLICT (user_id, question_type, language) DO UPDATE SET
  attempts = user_type_stat.attempts + 1,
  corrects = user_type_stat.corrects + EXCLUDED.corrects,
  last_difficulty = EXCLUDED.last_difficulty,
  updated_at = now()`, attempt.UserID, question.Type, question.CodeLanguage,
		correctIncrement, question.Difficulty)
	if err != nil {
		return domain.AttemptResult{}, err
	}

	// TODO(B-4): 仕様確定後、正式な SM-2 計算に置き換える。
	_, err = tx.Exec(ctx, `
INSERT INTO srs_state
  (user_id, question_id, interval_days, repetitions, due_on, last_reviewed_at)
VALUES ($1, $2, 1, CASE WHEN $3 THEN 1 ELSE 0 END, `+jstToday+` + 1, now())
ON CONFLICT (user_id, question_id) DO UPDATE SET
  repetitions = CASE WHEN $3 THEN srs_state.repetitions + 1 ELSE 0 END,
  interval_days = CASE
    WHEN NOT $3 THEN 1
    WHEN srs_state.interval_days < 1 THEN 1
    WHEN srs_state.interval_days = 1 THEN 3
    ELSE greatest(1, round(srs_state.interval_days * srs_state.easiness)::int)
  END,
  due_on = `+jstToday+` + CASE
    WHEN NOT $3 THEN 1
    WHEN srs_state.interval_days < 1 THEN 1
    WHEN srs_state.interval_days = 1 THEN 3
    ELSE greatest(1, round(srs_state.interval_days * srs_state.easiness)::int)
  END,
  last_reviewed_at = now(), updated_at = now()`, attempt.UserID, attempt.QuestionID, attempt.IsCorrect)
	if err != nil {
		return domain.AttemptResult{}, err
	}

	var completed *domain.DailyTaskRef
	var ref domain.DailyTaskRef
	err = tx.QueryRow(ctx, `
UPDATE daily_task SET completed_at = now(), attempt_id = $3
WHERE id = (
  SELECT id FROM daily_task
  WHERE user_id = $1 AND activity_date = `+jstToday+`
    AND question_id = $2 AND completed_at IS NULL
  ORDER BY slot_no LIMIT 1 FOR UPDATE
)
RETURNING slot_no, activity_date::text`, attempt.UserID, attempt.QuestionID, attempt.ID).
		Scan(&ref.SlotNo, &ref.ActivityDate)
	if err == nil {
		completed = &ref
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.AttemptResult{}, err
	}

	var progress domain.Progress
	var lastStudied *string
	err = tx.QueryRow(ctx, `
WITH completed_days AS (
  SELECT activity_date::date AS day FROM daily_task WHERE user_id = $1
  GROUP BY activity_date HAVING count(*) = count(completed_at)
), ranked AS (
  SELECT day, day - (row_number() OVER (ORDER BY day))::int AS grp FROM completed_days
), latest AS (SELECT max(day) AS day FROM completed_days), calculated AS (
  SELECT COALESCE((SELECT count(*) FROM ranked r, latest l
    WHERE l.day >= `+jstToday+` - 1 AND r.grp =
      (SELECT grp FROM ranked WHERE day = l.day)), 0)::int AS streak_days,
    (SELECT day FROM latest) AS last_studied_on
)
UPDATE user_progress p SET
  xp = p.xp + $2,
  streak_days = calculated.streak_days,
  last_studied_on = calculated.last_studied_on,
  updated_at = now()
FROM calculated WHERE p.user_id = $1
RETURNING p.xp, p.level, p.streak_days, to_char(p.last_studied_on, 'YYYY-MM-DD'),
          p.hearts, p.current_skill_node_id`, attempt.UserID, xpGained).
		Scan(&progress.XP, &progress.Level, &progress.StreakDays, &lastStudied,
			&progress.Hearts, &progress.CurrentSkillNodeID)
	if err != nil {
		return domain.AttemptResult{}, err
	}
	progress.LastStudiedOn = lastStudied

	if err := tx.Commit(ctx); err != nil {
		return domain.AttemptResult{}, err
	}
	return domain.AttemptResult{
		AttemptID: attempt.ID, IsCorrect: attempt.IsCorrect,
		CorrectKeys: question.CorrectKeys, Explanation: question.Explanation,
		XPGained: xpGained, Progress: progress, DailyTaskCompleted: completed,
	}, nil
}
