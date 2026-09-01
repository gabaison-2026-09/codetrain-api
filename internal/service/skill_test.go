package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// fakeSkillRepo は SkillRepository のフェイク。関数フィールドをテストごとに差し替える。
type fakeSkillRepo struct {
	skills func(ctx context.Context) ([]domain.Skill, error)
	nodes  func(ctx context.Context) ([]domain.SkillNode, error)
}

func (f fakeSkillRepo) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	return f.skills(ctx)
}

func (f fakeSkillRepo) ListSkillNodes(ctx context.Context) ([]domain.SkillNode, error) {
	return f.nodes(ctx)
}

// Skill.List の契約:
//
//	repository から取った skill と skill_node を、skill_id で紐づけて返す。
//	skill が0件なら nil ではなく空スライス（レスポンスを {"skills":[]} にするため）。
func TestSkillList(t *testing.T) {
	t.Run("ノードがスキルにぶら下がる", func(t *testing.T) {
		repo := fakeSkillRepo{
			skills: func(context.Context) ([]domain.Skill, error) {
				return []domain.Skill{
					{ID: "s1", Slug: "go", Name: "Go"},
					{ID: "s2", Slug: "sql", Name: "SQL"},
				}, nil
			},
			nodes: func(context.Context) ([]domain.SkillNode, error) {
				return []domain.SkillNode{
					{ID: "n10", SkillID: "s1", Slug: "go-basics"},
					{ID: "n11", SkillID: "s1", Slug: "go-slices"},
					{ID: "n20", SkillID: "s2", Slug: "sql-select"},
				}, nil
			},
		}

		got, err := NewSkill(repo).List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("skills = %d 件, want 2", len(got))
		}
		if len(got[0].Nodes) != 2 {
			t.Errorf("skills[0].Nodes = %d 件, want 2", len(got[0].Nodes))
		}
		if len(got[1].Nodes) != 1 {
			t.Errorf("skills[1].Nodes = %d 件, want 1", len(got[1].Nodes))
		}
		if got[1].Nodes[0].Slug != "sql-select" {
			t.Errorf("skills[1].Nodes[0].Slug = %q, want %q", got[1].Nodes[0].Slug, "sql-select")
		}
	})

	t.Run("ノードが無いスキルは Nodes が空", func(t *testing.T) {
		repo := fakeSkillRepo{
			skills: func(context.Context) ([]domain.Skill, error) {
				return []domain.Skill{{ID: "s1", Slug: "go"}}, nil
			},
			nodes: func(context.Context) ([]domain.SkillNode, error) {
				return nil, nil
			},
		}

		got, err := NewSkill(repo).List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got[0].Nodes) != 0 {
			t.Errorf("Nodes = %v, want 空", got[0].Nodes)
		}
	})

	t.Run("スキルが0件なら空スライス", func(t *testing.T) {
		nodesCalled := false
		repo := fakeSkillRepo{
			skills: func(context.Context) ([]domain.Skill, error) { return nil, nil },
			nodes: func(context.Context) ([]domain.SkillNode, error) {
				nodesCalled = true
				return nil, nil
			},
		}

		got, err := NewSkill(repo).List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("List() = nil, want 空スライス")
		}
		if len(got) != 0 {
			t.Errorf("skills = %d 件, want 0", len(got))
		}
		if nodesCalled {
			t.Error("スキルが0件のときはノードを引かないこと")
		}
	})

	t.Run("repository のエラーを伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")

		t.Run("skill の取得で失敗", func(t *testing.T) {
			repo := fakeSkillRepo{
				skills: func(context.Context) ([]domain.Skill, error) { return nil, wantErr },
				nodes:  func(context.Context) ([]domain.SkillNode, error) { return nil, nil },
			}
			if _, err := NewSkill(repo).List(context.Background()); !errors.Is(err, wantErr) {
				t.Errorf("err = %v, want %v", err, wantErr)
			}
		})

		t.Run("skill_node の取得で失敗", func(t *testing.T) {
			repo := fakeSkillRepo{
				skills: func(context.Context) ([]domain.Skill, error) {
					return []domain.Skill{{ID: "s1"}}, nil
				},
				nodes: func(context.Context) ([]domain.SkillNode, error) { return nil, wantErr },
			}
			if _, err := NewSkill(repo).List(context.Background()); !errors.Is(err, wantErr) {
				t.Errorf("err = %v, want %v", err, wantErr)
			}
		})
	})
}
