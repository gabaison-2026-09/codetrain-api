package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ListTypeStats は user_type_stat の全行を種別×言語別に返す。
// Accuracy は埋めず、corrects/attempts の算出は service 層が行う。
// language = '' の行（言語を問わない集計）もそのまま含める。
func (p *Postgres) ListTypeStats(ctx context.Context, userID string) ([]domain.TypeStat, error) {
	rows, err := p.pool.Query(ctx, `
SELECT question_type, language, attempts, corrects, last_difficulty
  FROM user_type_stat
 WHERE user_id = $1
 ORDER BY question_type, language`, userID)
	if err != nil {
		return nil, err
	}
	stats, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.TypeStat, error) {
		var s domain.TypeStat
		err := r.Scan(&s.QuestionType, &s.Language, &s.Attempts, &s.Corrects, &s.LastDifficulty)
		return s, err
	})
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []domain.TypeStat{}
	}
	return stats, nil
}
