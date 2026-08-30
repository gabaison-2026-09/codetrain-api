package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// FindUserByExternalID は認証基盤上の識別子からユーザーと進捗を引く。
// 進捗の行が無い場合は COALESCE により初期値（XP・streak・hearts が 0）が入る。
// 該当ユーザーがいない場合は ErrNotFound を返す。
func (p *Postgres) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	var user domain.User
	var progress domain.Progress
	var email *string
	var lastStudied *string

	err := p.pool.QueryRow(ctx, `
		SELECT u.id, u.external_id, u.display_name, u.email, u.created_at,
		       COALESCE(p.xp, 0), COALESCE(p.streak_days, 0),
		       to_char(p.last_studied_on, 'YYYY-MM-DD'),
		       COALESCE(p.hearts, 0), p.current_skill_node_id
		  FROM app_user u
		  LEFT JOIN user_progress p ON p.user_id = u.id
		 WHERE u.external_id = $1`, externalID).
		Scan(&user.ID, &user.ExternalID, &user.DisplayName, &email, &user.CreatedAt,
			&progress.XP, &progress.StreakDays, &lastStudied,
			&progress.Hearts, &progress.CurrentSkillNodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.Progress{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, domain.Progress{}, err
	}

	if email != nil {
		user.Email = *email
	}
	progress.LastStudiedOn = lastStudied
	return user, progress, nil
}
