package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type fakeAdminQuestions struct {
	list func(context.Context, service.AdminQuestionSearchParams) (service.AdminQuestionList, error)
}

func (f fakeAdminQuestions) List(ctx context.Context, params service.AdminQuestionSearchParams) (service.AdminQuestionList, error) {
	return f.list(ctx, params)
}

func TestAdminListQuestions(t *testing.T) {
	var got service.AdminQuestionSearchParams
	h := New(Deps{Admin: fakeAdminQuestions{
		list: func(_ context.Context, params service.AdminQuestionSearchParams) (service.AdminQuestionList, error) {
			got = params
			return service.AdminQuestionList{
				Questions: []domain.AdminQuestionSummary{{ID: "q1", Status: domain.QuestionStatusNeedsReview}},
			}, nil
		},
	}})

	rec := serve(t, h.AdminListQuestions, http.MethodGet,
		"/v1/admin/questions?status=needs_review&type=bug_finding&language=go&skill_id=00000000-0000-0000-0000-000000000001&q=off-by-one&limit=10",
		"reviewer-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got.Status != domain.QuestionStatusNeedsReview || got.Type != domain.QuestionTypeBugFinding ||
		got.Language != "go" || got.Q != "off-by-one" || got.Limit != 10 {
		t.Errorf("params = %+v", got)
	}
}
