package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// SearchAdminQuestions は status を問わない問題を条件検索する。
// Limit+1 件返す（次頁判定は service）。
func (p *Postgres) SearchAdminQuestions(ctx context.Context, q domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error) {
	var b strings.Builder
	args := make([]any, 0, 7)
	n := 0

	b.WriteString(`
SELECT q.id, q.status, q.type, q.difficulty, q.title, q.created_at
  FROM question q`)
	if q.SkillID != "" {
		b.WriteString(` JOIN skill_node n ON n.id = q.skill_node_id`)
	}
	b.WriteString(` WHERE 1=1`)

	if q.Status != "" {
		n++
		args = append(args, q.Status)
		fmt.Fprintf(&b, ` AND q.status = $%d`, n)
	}
	if q.Type != "" {
		n++
		args = append(args, q.Type)
		fmt.Fprintf(&b, ` AND q.type = $%d`, n)
	}
	if q.Language != "" {
		n++
		args = append(args, q.Language)
		fmt.Fprintf(&b, ` AND q.code_language = $%d`, n)
	}
	if q.SkillID != "" {
		n++
		args = append(args, q.SkillID)
		fmt.Fprintf(&b, ` AND n.skill_id = $%d`, n)
	}
	if q.Query != "" {
		n++
		args = append(args, "%"+escapeLike(q.Query)+"%")
		fmt.Fprintf(&b, ` AND (q.title ILIKE $%d ESCAPE '\' OR q.body ILIKE $%d ESCAPE '\')`, n, n)
	}
	if q.CursorCreatedAt != nil {
		n++
		args = append(args, *q.CursorCreatedAt)
		n++
		args = append(args, q.CursorID)
		fmt.Fprintf(&b, ` AND (q.created_at, q.id) < ($%d, $%d)`, n-1, n)
	}

	limit := q.Limit
	if limit < 1 {
		limit = 20
	}
	n++
	args = append(args, limit+1)
	fmt.Fprintf(&b, ` ORDER BY q.created_at DESC, q.id DESC LIMIT $%d`, n)

	rows, err := p.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.AdminQuestionSummary, error) {
		var item domain.AdminQuestionSummary
		err := r.Scan(&item.ID, &item.Status, &item.Type, &item.Difficulty, &item.Title, &item.CreatedAt)
		return item, err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.AdminQuestionSummary{}
	}
	return items, nil
}
