package service

import (
	"context"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// Skill はスキルツリーのユースケース。
type Skill struct {
	repo SkillRepository
}

func NewSkill(repo SkillRepository) *Skill { return &Skill{repo: repo} }

// List はスキルツリーをノード込みで返す。
// スキルが1件も無い場合は nil ではなく空スライスを返し、
// レスポンスが {"skills":[]} になるようにする。
func (s *Skill) List(ctx context.Context) ([]domain.Skill, error) {
	skills, err := s.repo.ListSkills(ctx)
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return []domain.Skill{}, nil
	}

	nodes, err := s.repo.ListSkillNodes(ctx)
	if err != nil {
		return nil, err
	}

	bySkill := make(map[string][]domain.SkillNode, len(skills))
	for _, n := range nodes {
		bySkill[n.SkillID] = append(bySkill[n.SkillID], n)
	}
	for i := range skills {
		skills[i].Nodes = bySkill[skills[i].ID]
	}
	return skills, nil
}
