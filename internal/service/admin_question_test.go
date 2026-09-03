package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

type fakeAdminQuestionRepo struct {
	search func(context.Context, domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error)
}

func (f fakeAdminQuestionRepo) SearchAdminQuestions(ctx context.Context, params domain.AdminQuestionSearch) ([]domain.AdminQuestionSummary, error) {
	return f.search(ctx, params)
}

type fakeAdminQuestionDetails struct {
	find func(context.Context, string) (domain.AdminQuestion, error)
	list func(context.Context, string) ([]domain.ReviewEntry, error)
}

func (f fakeAdminQuestionDetails) FindQuestionFull(ctx context.Context, id string) (domain.AdminQuestion, error) {
	return f.find(ctx, id)
}

func (f fakeAdminQuestionDetails) ListReviewHistory(ctx context.Context, id string) ([]domain.ReviewEntry, error) {
	return f.list(ctx, id)
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

func TestAdminQuestionGet(t *testing.T) {
	created := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	svc := NewAdminQuestion(fakeAdminQuestionRepo{}, fakeAdminQuestionDetails{
		find: func(_ context.Context, id string) (domain.AdminQuestion, error) {
			return domain.AdminQuestion{ID: id, CorrectKeys: []string{"a"}}, nil
		},
		list: func(_ context.Context, _ string) ([]domain.ReviewEntry, error) {
			return []domain.ReviewEntry{{ID: "r1", CreatedAt: created}}, nil
		},
	})

	got, err := svc.Get(context.Background(), "q1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "q1" || len(got.ReviewHistory) != 1 || got.ReviewHistory[0].ID != "r1" {
		t.Errorf("question = %+v", got)
	}
}

func TestAdminQuestionGetTranslatesNotFound(t *testing.T) {
	svc := NewAdminQuestion(fakeAdminQuestionRepo{}, fakeAdminQuestionDetails{
		find: func(context.Context, string) (domain.AdminQuestion, error) {
			return domain.AdminQuestion{}, repository.ErrNotFound
		},
		list: func(context.Context, string) ([]domain.ReviewEntry, error) { return nil, nil },
	})

	_, err := svc.Get(context.Background(), "q1")
	if !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("error = %v, want ErrQuestionNotFound", err)
	}
}
