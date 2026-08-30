package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// ListSkills はスキルを表示順で返す。ノードはぶら下げない（service が組み立てる）。
func (p *Postgres) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, slug, name, COALESCE(description, ''), display_order
		  FROM skill
		 ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Skill, error) {
		var s domain.Skill
		err := r.Scan(&s.ID, &s.Slug, &s.Name, &s.Description, &s.DisplayOrder)
		return s, err
	})
}

// ListSkillNodes は全スキルのノードを skill_id・表示順で返す。
func (p *Postgres) ListSkillNodes(ctx context.Context) ([]domain.SkillNode, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, skill_id, parent_id, slug, name, COALESCE(description, ''),
		       difficulty, display_order
		  FROM skill_node
		 ORDER BY skill_id, display_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.SkillNode, error) {
		var n domain.SkillNode
		err := r.Scan(&n.ID, &n.SkillID, &n.ParentID, &n.Slug, &n.Name,
			&n.Description, &n.Difficulty, &n.DisplayOrder)
		return n, err
	})
}
