package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

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
