package handler

import (
	"context"
	"errors"
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

type fakeAdminQuestionGetter struct {
	get func(context.Context, string) (domain.AdminQuestion, error)
}

func (f fakeAdminQuestionGetter) Get(ctx context.Context, id string) (domain.AdminQuestion, error) {
	return f.get(ctx, id)
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

func TestAdminGetQuestion(t *testing.T) {
	var gotID string
	h := New(Deps{AdminGetter: fakeAdminQuestionGetter{
		get: func(_ context.Context, id string) (domain.AdminQuestion, error) {
			gotID = id
			return domain.AdminQuestion{
				ID:          id,
				CorrectKeys: []string{"a"},
				ReviewHistory: []domain.ReviewEntry{
					{ID: "review-1"},
				},
			}, nil
		},
	}})

	rec := serveWithParam(t, h.AdminGetQuestion, http.MethodGet,
		"/v1/admin/questions/:id", "00000000-0000-0000-0000-000000000001", "reviewer-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotID == "" {
		t.Fatal("question id が service に渡されていない")
	}
}

func TestAdminGetQuestionErrors(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		sub        string
		get        func(context.Context, string) (domain.AdminQuestion, error)
		wantStatus int
	}{
		{
			name:       "認証なし",
			id:         "00000000-0000-0000-0000-000000000001",
			get:        func(context.Context, string) (domain.AdminQuestion, error) { return domain.AdminQuestion{}, nil },
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "uuid不正",
			id:         "not-a-uuid",
			sub:        "reviewer-01",
			get:        func(context.Context, string) (domain.AdminQuestion, error) { return domain.AdminQuestion{}, nil },
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "問題なし",
			id:   "00000000-0000-0000-0000-000000000001",
			sub:  "reviewer-01",
			get: func(context.Context, string) (domain.AdminQuestion, error) {
				return domain.AdminQuestion{}, service.ErrQuestionNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "内部エラー",
			id:   "00000000-0000-0000-0000-000000000001",
			sub:  "reviewer-01",
			get: func(context.Context, string) (domain.AdminQuestion, error) {
				return domain.AdminQuestion{}, errors.New("DB障害")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{AdminGetter: fakeAdminQuestionGetter{get: tt.get}})
			rec := serveWithParam(t, h.AdminGetQuestion, http.MethodGet,
				"/v1/admin/questions/:id", tt.id, tt.sub)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
