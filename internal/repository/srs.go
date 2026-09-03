package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ListDue は今日以前が期限の published 問題を期限順に返す。
func (p *Postgres) ListDue(ctx context.Context, userID string, limit int) ([]domain.SRSDueItem, error) {
	rows, err := p.pool.Query(ctx, `
SELECT q.id, q.type, q.difficulty, q.title, COALESCE(q.code_language, ''),
       q.tags, to_char(s.due_on, 'YYYY-MM-DD')
  FROM srs_state s
  JOIN question q ON q.id = s.question_id
 WHERE s.user_id = $1
   AND s.due_on <= CURRENT_DATE
   AND q.status = 'published'
 ORDER BY s.due_on ASC, q.id
 LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.SRSDueItem, error) {
		var item domain.SRSDueItem
		err := r.Scan(&item.ID, &item.Type, &item.Difficulty, &item.Title,
			&item.CodeLanguage, &item.Tags, &item.DueOn)
		if item.Tags == nil {
			item.Tags = []string{}
		}
		return item, err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.SRSDueItem{}
	}
	return items, nil
}
