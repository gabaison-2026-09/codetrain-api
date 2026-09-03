package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// FindUserByExternalID は認証基盤上の識別子からユーザーと進捗を引く。
// 進捗の行が無い場合は COALESCE により初期値（XP・streak・hearts が 0）が入る。
// 該当ユーザーがいない場合は ErrNotFound を返す。
func (p *Postgres) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	var user domain.User
	var progress domain.Progress
	var email *string
	var avatarURL *string
	var lastStudied *string

	err := p.pool.QueryRow(ctx, `
		SELECT u.id, u.external_id, u.display_name, u.email, u.avatar_url, u.created_at,
		       COALESCE(p.xp, 0), COALESCE(p.level, 1), COALESCE(p.streak_days, 0),
		       to_char(p.last_studied_on, 'YYYY-MM-DD'),
		       COALESCE(p.hearts, 0), p.current_skill_node_id
		  FROM app_user u
		  LEFT JOIN user_progress p ON p.user_id = u.id
		 WHERE u.external_id = $1`, externalID).
		Scan(&user.ID, &user.ExternalID, &user.DisplayName, &email, &avatarURL, &user.CreatedAt,
			&progress.XP, &progress.Level, &progress.StreakDays, &lastStudied,
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
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	progress.LastStudiedOn = lastStudied
	return user, progress, nil
}

// InsertUser は app_user と user_progress（初期値）を同一トランザクションで作成し、
// GET /v1/me 相当の内容を返す。
// external_id の UNIQUE 違反は ErrAlreadyExists に翻訳する。
func (p *Postgres) InsertUser(ctx context.Context, externalID, displayName string, avatarURL *string) (domain.User, domain.Progress, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Progress{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit 成功後は no-op

	var user domain.User
	var email *string
	err = tx.QueryRow(ctx, `
		INSERT INTO app_user (external_id, display_name, avatar_url)
		VALUES ($1, $2, $3)
		RETURNING id, external_id, display_name, email, created_at`,
		externalID, displayName, avatarURL).
		Scan(&user.ID, &user.ExternalID, &user.DisplayName, &email, &user.CreatedAt)
	if isUniqueViolation(err) {
		return domain.User{}, domain.Progress{}, ErrAlreadyExists
	}
	if err != nil {
		return domain.User{}, domain.Progress{}, err
	}
	if email != nil {
		user.Email = *email
	}

	// 初期値: xp 0 / level 1 / streak 0 / hearts 5（B-3 確定まで固定）
	var progress domain.Progress
	var lastStudied *string
	err = tx.QueryRow(ctx, `
		INSERT INTO user_progress (user_id, xp, level, streak_days, hearts)
		VALUES ($1, 0, 1, 0, 5)
		RETURNING xp, level, streak_days,
		          to_char(last_studied_on, 'YYYY-MM-DD'),
		          hearts, current_skill_node_id`,
		user.ID).
		Scan(&progress.XP, &progress.Level, &progress.StreakDays, &lastStudied,
			&progress.Hearts, &progress.CurrentSkillNodeID)
	if err != nil {
		return domain.User{}, domain.Progress{}, err
	}
	progress.LastStudiedOn = lastStudied

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Progress{}, err
	}
	return user, progress, nil
}

// isUniqueViolation は PostgreSQL の UNIQUE 制約違反（SQLSTATE 23505）かを判定する。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
