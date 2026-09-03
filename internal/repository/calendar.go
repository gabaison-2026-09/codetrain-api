package repository

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) DailyConsumption(ctx context.Context, userID string, from, to string) ([]domain.CalendarDay, error) {
	rows, err := p.pool.Query(ctx, `
SELECT to_char(activity_date, 'YYYY-MM-DD'), count(*)::int, count(completed_at)::int
FROM daily_task
WHERE user_id = $1 AND activity_date BETWEEN $2 AND $3
GROUP BY activity_date ORDER BY activity_date`, userID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.CalendarDay, error) {
		var d domain.CalendarDay
		if err := r.Scan(&d.Date, &d.TotalSlots, &d.CompletedSlots); err != nil {
			return d, err
		}
		d.Completed = d.TotalSlots > 0 && d.TotalSlots == d.CompletedSlots
		return d, nil
	})
}

func (p *Postgres) Streak(ctx context.Context, userID string) (int, *string, error) {
	var days int
	var last *string
	err := p.pool.QueryRow(ctx, `
WITH completed_days AS (
  SELECT activity_date::date AS day
  FROM daily_task WHERE user_id = $1
  GROUP BY activity_date
  HAVING count(*) = count(completed_at)
), ranked AS (
  SELECT day, day - (row_number() OVER (ORDER BY day))::int AS grp
  FROM completed_days
), latest AS (SELECT max(day) AS day FROM completed_days)
SELECT COALESCE((SELECT count(*) FROM ranked r, latest l
  WHERE l.day >= CURRENT_DATE - 1 AND r.grp = (
    SELECT grp FROM ranked WHERE day = l.day)), 0)::int,
       (SELECT day::text FROM latest)`, userID).Scan(&days, &last)
	return days, last, err
}
