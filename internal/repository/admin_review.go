package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// DecideReview はレビュー判定と問題ステータス更新を同一トランザクションで行う。
func (p *Postgres) DecideReview(
	ctx context.Context,
	reviewerID, questionID string,
	decision domain.ReviewDecision,
	notes string,
) (domain.ReviewResult, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.ReviewResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit 成功後は no-op

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM question WHERE id = $1)`, questionID).Scan(&exists); err != nil {
		return domain.ReviewResult{}, err
	}
	if !exists {
		return domain.ReviewResult{}, ErrNotFound
	}

	var result domain.ReviewResult
	err = tx.QueryRow(ctx, `
UPDATE review_queue
   SET decision = $1, notes = $2, reviewer_id = $3, reviewed_at = now(), updated_at = now()
 WHERE question_id = $4 AND decision IS NULL
RETURNING id, reviewed_at`,
		decision, notes, reviewerID, questionID,
	).Scan(&result.ID, &result.ReviewedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReviewResult{}, ErrAlreadyExists
	}
	if err != nil {
		return domain.ReviewResult{}, err
	}

	if decision != domain.ReviewDecisionNeedsEdit {
		status := domain.QuestionStatusPublished
		if decision == domain.ReviewDecisionRejected {
			status = domain.QuestionStatusRejected
		}
		if _, err := tx.Exec(ctx, `
UPDATE question SET status = $1, updated_at = now() WHERE id = $2`,
			status, questionID,
		); err != nil {
			return domain.ReviewResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ReviewResult{}, err
	}
	result.QuestionID = questionID
	result.ReviewerID = reviewerID
	result.Decision = decision
	result.Notes = notes
	return result, nil
}

// ListReviewQueue は未レビュー問題を queued_at 昇順で取得する。
// Limit+1 件返す（次頁判定は service）。
func (p *Postgres) ListReviewQueue(
	ctx context.Context,
	cursorQueuedAt *time.Time,
	cursorID string,
	limit int,
) ([]domain.ReviewQueueItem, error) {
	args := make([]any, 0, 3)
	query := `
SELECT rq.id AS review_id, rq.question_id, q.title, q.type, q.difficulty,
       rq.created_at AS queued_at
  FROM review_queue rq
  JOIN question q ON q.id = rq.question_id
 WHERE rq.decision IS NULL`

	if cursorQueuedAt != nil {
		args = append(args, *cursorQueuedAt, cursorID)
		query += ` AND (rq.created_at, rq.id) > ($1, $2)`
	}

	if limit < 1 {
		limit = 20
	}
	args = append(args, limit+1)
	limitArg := len(args)
	query += ` ORDER BY rq.created_at ASC, rq.id ASC LIMIT $` + strconv.Itoa(limitArg)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.ReviewQueueItem, error) {
		var item domain.ReviewQueueItem
		err := r.Scan(
			&item.ReviewID,
			&item.QuestionID,
			&item.Title,
			&item.Type,
			&item.Difficulty,
			&item.QueuedAt,
		)
		return item, err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.ReviewQueueItem{}
	}
	return items, nil
}
