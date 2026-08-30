// Package store は PostgreSQL へのアクセスをまとめる。
//
// スキーマの定義は codetrain-core（migrations）が持つ。api はそれを読むだけで、
// マイグレーションは打たない（REPOSITORIES.md §2.1）。
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ErrNotFound は対象のレコードが無いことを表す。
var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DB プールの作成に失敗: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Ping は /healthz の db 判定に使う。
func (s *Store) Ping(ctx context.Context) error {
	var one int
	return s.pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}

// ListSkills はスキルツリーをノード込みで返す。
// シードが入っているかの確認にも使う（LOCAL_DEV.md §5.2）。
func (s *Store) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, COALESCE(description, ''), display_order
		  FROM skill
		 ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	skills, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Skill, error) {
		var s domain.Skill
		err := r.Scan(&s.ID, &s.Slug, &s.Name, &s.Description, &s.DisplayOrder)
		return s, err
	})
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return []domain.Skill{}, nil
	}

	nodeRows, err := s.pool.Query(ctx, `
		SELECT id, skill_id, parent_id, slug, name, COALESCE(description, ''),
		       difficulty, display_order
		  FROM skill_node
		 ORDER BY skill_id, display_order, id`)
	if err != nil {
		return nil, err
	}
	nodes, err := pgx.CollectRows(nodeRows, func(r pgx.CollectableRow) (domain.SkillNode, error) {
		var n domain.SkillNode
		err := r.Scan(&n.ID, &n.SkillID, &n.ParentID, &n.Slug, &n.Name,
			&n.Description, &n.Difficulty, &n.DisplayOrder)
		return n, err
	})
	if err != nil {
		return nil, err
	}

	bySkill := make(map[int64][]domain.SkillNode, len(skills))
	for _, n := range nodes {
		bySkill[n.SkillID] = append(bySkill[n.SkillID], n)
	}
	for i := range skills {
		skills[i].Nodes = bySkill[skills[i].ID]
	}
	return skills, nil
}

// UserWithProgress は /v1/me が返す内容。
type UserWithProgress struct {
	User     domain.User     `json:"user"`
	Progress domain.Progress `json:"progress"`
}

// FindUserByExternalID は認証済みユーザーの sub からユーザーと進捗を引く。
// 進捗の行が無い場合は初期値を返す。
func (s *Store) FindUserByExternalID(ctx context.Context, externalID string) (UserWithProgress, error) {
	var out UserWithProgress
	var email *string
	var lastStudied *string
	var progressExists bool

	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.external_id, u.display_name, u.email, u.created_at,
		       p.user_id IS NOT NULL,
		       COALESCE(p.xp, 0), COALESCE(p.streak_days, 0),
		       to_char(p.last_studied_on, 'YYYY-MM-DD'),
		       COALESCE(p.hearts, 0), p.current_skill_node_id
		  FROM app_user u
		  LEFT JOIN user_progress p ON p.user_id = u.id
		 WHERE u.external_id = $1`, externalID).
		Scan(&out.User.ID, &out.User.ExternalID, &out.User.DisplayName, &email, &out.User.CreatedAt,
			&progressExists,
			&out.Progress.XP, &out.Progress.StreakDays, &lastStudied,
			&out.Progress.Hearts, &out.Progress.CurrentSkillNodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserWithProgress{}, ErrNotFound
	}
	if err != nil {
		return UserWithProgress{}, err
	}

	if email != nil {
		out.User.Email = *email
	}
	out.Progress.LastStudiedOn = lastStudied
	return out, nil
}
