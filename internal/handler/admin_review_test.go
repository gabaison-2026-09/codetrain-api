package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type fakeReviewQueue struct {
	list func(context.Context, service.ReviewQueueParams) (service.ReviewQueueList, error)
}

func (f fakeReviewQueue) List(
	ctx context.Context,
	params service.ReviewQueueParams,
) (service.ReviewQueueList, error) {
	return f.list(ctx, params)
}

func TestAdminReviewQueue(t *testing.T) {
	var got service.ReviewQueueParams
	h := New(Deps{ReviewQueue: fakeReviewQueue{
		list: func(_ context.Context, params service.ReviewQueueParams) (service.ReviewQueueList, error) {
			got = params
			return service.ReviewQueueList{
				Items: []domain.ReviewQueueItem{{ReviewID: "r1", QuestionID: "q1"}},
			}, nil
		},
	}})

	rec := serve(t, h.AdminReviewQueue, http.MethodGet, "/v1/admin/review-queue?limit=10", "reviewer-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got.Limit != 10 {
		t.Errorf("params = %+v", got)
	}
}

func TestAdminReviewQueueRequiresAuthentication(t *testing.T) {
	called := false
	h := New(Deps{ReviewQueue: fakeReviewQueue{
		list: func(context.Context, service.ReviewQueueParams) (service.ReviewQueueList, error) {
			called = true
			return service.ReviewQueueList{}, nil
		},
	}})

	rec := serve(t, h.AdminReviewQueue, http.MethodGet, "/v1/admin/review-queue", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Error("認証失敗時に service が呼ばれた")
	}
	if rec.Body.String() == "" {
		t.Error("エラーレスポンスが空")
	}
}
