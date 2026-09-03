package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// FindQuestionFull は管理者向けに問題の全項目と出典を取得する。
func (p *Postgres) FindQuestionFull(ctx context.Context, questionID string) (domain.AdminQuestion, error) {
	var q domain.AdminQuestion
	var code, codeLanguage, explanation *string
	var skillNodeID *string
	var rawSourceID string
	var promptVersion, modelID *string
	var genTokens *int

	err := p.pool.QueryRow(ctx, `
SELECT q.id, q.status, q.type, q.difficulty, q.title, q.body,
       q.code, q.code_language, q.choices, q.correct_keys, q.explanation, q.tags,
       q.skill_node_id, q.raw_source_id,
       q.prompt_version, q.model_id, q.gen_tokens, q.generated_at
  FROM question q
  JOIN raw_source rs ON rs.id = q.raw_source_id
 WHERE q.id = $1`,
		questionID,
	).Scan(
		&q.ID, &q.Status, &q.Type, &q.Difficulty, &q.Title, &q.Body,
		&code, &codeLanguage, &q.Choices, &q.CorrectKeys, &explanation, &q.Tags,
		&skillNodeID, &rawSourceID,
		&promptVersion, &modelID, &genTokens, &q.GeneratedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AdminQuestion{}, ErrNotFound
	}
	if err != nil {
		return domain.AdminQuestion{}, err
	}

	q.SkillNodeID = skillNodeID
	q.RawSourceID = rawSourceID
	if code != nil {
		q.Code = *code
	}
	if codeLanguage != nil {
		q.CodeLanguage = *codeLanguage
	}
	if explanation != nil {
		q.Explanation = *explanation
	}
	if promptVersion != nil {
		q.PromptVersion = *promptVersion
	}
	if modelID != nil {
		q.ModelID = *modelID
	}
	q.GenTokens = genTokens
	if q.Choices == nil {
		q.Choices = []domain.Choice{}
	}
	if q.CorrectKeys == nil {
		q.CorrectKeys = []string{}
	}
	if q.Tags == nil {
		q.Tags = []string{}
	}

	q.ReviewHistory = []domain.ReviewEntry{}

	return q, nil
}

// ListReviewHistory は問題に紐づくレビュー履歴を新しい順で取得する。
func (p *Postgres) ListReviewHistory(ctx context.Context, questionID string) ([]domain.ReviewEntry, error) {
	rows, err := p.pool.Query(ctx, `
SELECT id, reviewer_id, decision, notes, created_at
  FROM review_queue
 WHERE question_id = $1
 ORDER BY created_at DESC`,
		questionID,
	)
	if err != nil {
		return nil, err
	}

	history, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.ReviewEntry, error) {
		var entry domain.ReviewEntry
		if err := r.Scan(&entry.ID, &entry.ReviewerID, &entry.Decision, &entry.Notes, &entry.CreatedAt); err != nil {
			return domain.ReviewEntry{}, err
		}
		return entry, nil
	})
	if err != nil {
		return nil, err
	}
	if history == nil {
		history = []domain.ReviewEntry{}
	}
	return history, nil
}

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
