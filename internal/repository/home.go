package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) PickUnansweredPublished(ctx context.Context, userID string, typ domain.QuestionType, language string, difficulty int) (domain.Question, error) {
	var q domain.Question
	var choices []byte
	err := p.pool.QueryRow(ctx, `SELECT q.id,q.type,q.difficulty,q.title,q.body,COALESCE(q.code,''),COALESCE(q.code_language,''),q.choices
		FROM question q WHERE q.status='published' AND q.type=$2 AND COALESCE(q.code_language,'')=$3 AND q.difficulty=$4
		AND NOT EXISTS (SELECT 1 FROM attempt a WHERE a.user_id=$1 AND a.question_id=q.id) ORDER BY random() LIMIT 1`,
		userID, typ, language, difficulty).Scan(&q.ID, &q.Type, &q.Difficulty, &q.Title, &q.Body, &q.Code, &q.CodeLanguage, &choices)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return q, ErrNotFound
		}
		return q, err
	}
	_ = json.Unmarshal(choices, &q.Choices)
	return q, nil
}

func (p *Postgres) UpsertDailyTask(ctx context.Context, userID string, slot domain.TaskConfig, questionID string) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO daily_task(user_id,activity_date,slot_no,question_type,language,difficulty,question_id)
		VALUES($1,`+jstToday+`,$2,$3,$4,$5,$6) ON CONFLICT(user_id,activity_date,slot_no) DO NOTHING`,
		userID, slot.SlotNo, slot.QuestionType, slot.Language, slot.Difficulty, questionID)
	return err
}

func (p *Postgres) GetTodayHome(ctx context.Context, userID string) ([]domain.HomeTask, error) {
	rows, err := p.pool.Query(ctx, `SELECT d.id,d.activity_date::text,d.slot_no,d.question_type,d.language,d.difficulty,d.question_id,d.completed_at,
		q.id,q.type,q.difficulty,q.title,q.body,COALESCE(q.code,''),COALESCE(q.code_language,''),q.choices
		FROM daily_task d JOIN question q ON q.id=d.question_id
		WHERE d.user_id=$1 AND d.activity_date=`+jstToday+` ORDER BY d.slot_no`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.HomeTask
	for rows.Next() {
		var t domain.HomeTask
		var date string
		var choices []byte
		if err := rows.Scan(&t.ID, &date, &t.SlotNo, &t.QuestionType, &t.Language, &t.Difficulty, &t.QuestionID, &t.CompletedAt,
			&t.Question.ID, &t.Question.Type, &t.Question.Difficulty, &t.Question.Title, &t.Question.Body, &t.Question.Code, &t.Question.CodeLanguage, &choices); err != nil {
			return nil, err
		}
		t.ActivityDate = date
		_ = json.Unmarshal(choices, &t.Question.Choices)
		out = append(out, t)
	}
	return out, rows.Err()
}
