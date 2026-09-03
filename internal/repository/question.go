package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// SearchQuestions は status = published の問題を条件検索する。
// userID は answered / unanswered_only の判定に使う。Limit+1 件返す（次頁判定は service）。
func (p *Postgres) SearchQuestions(ctx context.Context, userID string, q domain.QuestionSearch) ([]domain.QuestionSummary, error) {
	var b strings.Builder
	args := []any{userID}
	n := 1

	b.WriteString(`
SELECT q.id, q.type, q.difficulty, q.title, COALESCE(q.code_language, ''),
       q.tags, q.skill_node_id,
       EXISTS(SELECT 1 FROM attempt a WHERE a.user_id = $1 AND a.question_id = q.id) AS answered,
       q.created_at
  FROM question q
 WHERE q.status = 'published'`)

	if q.SkillNodeID != "" {
		n++
		args = append(args, q.SkillNodeID)
		fmt.Fprintf(&b, ` AND q.skill_node_id = $%d`, n)
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
	if q.Difficulty != nil {
		n++
		args = append(args, *q.Difficulty)
		fmt.Fprintf(&b, ` AND q.difficulty = $%d`, n)
	}
	if len(q.Tags) > 0 {
		n++
		args = append(args, q.Tags)
		fmt.Fprintf(&b, ` AND q.tags @> $%d`, n)
	}
	if q.Query != "" {
		n++
		args = append(args, "%"+escapeLike(q.Query)+"%")
		fmt.Fprintf(&b, ` AND (q.title ILIKE $%d ESCAPE '\' OR q.body ILIKE $%d ESCAPE '\')`, n, n)
	}
	if q.UnansweredOnly {
		b.WriteString(` AND NOT EXISTS (SELECT 1 FROM attempt a WHERE a.user_id = $1 AND a.question_id = q.id)`)
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
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.QuestionSummary, error) {
		var s domain.QuestionSummary
		err := r.Scan(&s.ID, &s.Type, &s.Difficulty, &s.Title, &s.CodeLanguage,
			&s.Tags, &s.SkillNodeID, &s.Answered, &s.CreatedAt)
		if s.Tags == nil {
			s.Tags = []string{}
		}
		return s, err
	})
}

// FindPublishedByID は status=published の問題を1件取得する。
// raw_source を JOIN して Attribution を組み立て、attempt の EXISTS で answered を判定する。
// 該当行がなければ ErrNotFound を返す。
func (p *Postgres) FindPublishedByID(ctx context.Context, userID, questionID string) (domain.Question, bool, error) {
	var q domain.Question
	var code, codeLang, explanation *string
	var repoFullName, repoURL, commitSHA, licenseSPDX string
	var filePath, licenseURL, authorAttribution *string
	var answered bool

	err := p.pool.QueryRow(ctx, `
SELECT q.id, q.skill_node_id, q.type, q.difficulty, q.title, q.body,
       q.code, q.code_language, q.choices, q.correct_keys, q.explanation, q.tags,
       rs.repo_full_name, rs.repo_url, rs.commit_sha, rs.file_path,
       rs.license_spdx, rs.license_url, rs.author_attribution,
       EXISTS(SELECT 1 FROM attempt a WHERE a.user_id = $1 AND a.question_id = q.id) AS answered
  FROM question q
  JOIN raw_source rs ON rs.id = q.raw_source_id
 WHERE q.id = $2 AND q.status = 'published'`,
		userID, questionID).
		Scan(&q.ID, &q.SkillNodeID, &q.Type, &q.Difficulty, &q.Title, &q.Body,
			&code, &codeLang, &q.Choices, &q.CorrectKeys, &explanation, &q.Tags,
			&repoFullName, &repoURL, &commitSHA, &filePath,
			&licenseSPDX, &licenseURL, &authorAttribution,
			&answered)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Question{}, false, ErrNotFound
	}
	if err != nil {
		return domain.Question{}, false, err
	}

	if code != nil {
		q.Code = *code
	}
	if codeLang != nil {
		q.CodeLanguage = *codeLang
	}
	if explanation != nil {
		q.Explanation = *explanation
	}
	if q.Tags == nil {
		q.Tags = []string{}
	}
	if q.CorrectKeys == nil {
		q.CorrectKeys = []string{}
	}
	if q.Choices == nil {
		q.Choices = []domain.Choice{}
	}

	q.Attribution = &domain.Attribution{
		RepoFullName: repoFullName,
		RepoURL:      repoURL,
		CommitSHA:    commitSHA,
		LicenseSPDX:  licenseSPDX,
	}
	if filePath != nil {
		q.Attribution.FilePath = *filePath
	}
	if licenseURL != nil {
		q.Attribution.LicenseURL = *licenseURL
	}
	if authorAttribution != nil {
		q.Attribution.Author = *authorAttribution
	}

	return q, answered, nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
