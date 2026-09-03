package service

import (
	"context"
	"testing"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type fakeAdminQuestionRepo struct {
	search func(context.Context, domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error)
}

func (f fakeAdminQuestionRepo) SearchAdminQuestions(ctx context.Context, params domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error) {
	return f.search(ctx, params)
}

func TestAdminQuestionList(t *testing.T) {
	t1 := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	var got domain.AdminQuestionSearch
	svc := NewAdminQuestion(fakeAdminQuestionRepo{
		search: func(_ context.Context, params domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error) {
			got = params
			return []domain.AdminQuestionSummary{
				{ID: "q1", Status: domain.QuestionStatusNeedsReview, CreatedAt: t1},
				{ID: "q2", Status: domain.QuestionStatusDraft, CreatedAt: t1.Add(-time.Hour)},
			}, nil
		},
	})

	list, err := svc.List(context.Background(), AdminQuestionSearchParams{
		Status:   domain.QuestionStatusNeedsReview,
		Type:     domain.QuestionTypeBugFinding,
		Language: "go",
		SkillID:  "skill-1",
		Q:        "off-by-one",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Status != domain.QuestionStatusNeedsReview || got.Type != domain.QuestionTypeBugFinding ||
		got.Language != "go" || got.SkillID != "skill-1" || got.Query != "off-by-one" {
		t.Errorf("search = %+v", got)
	}
	if len(list.Questions) != 1 || list.Questions[0].ID != "q1" {
		t.Errorf("questions = %+v", list.Questions)
	}
	if list.NextCursor == nil {
		t.Fatal("next_cursor = nil, want non-nil")
	}
}
