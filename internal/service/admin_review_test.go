package service

import (
	"context"
	"testing"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type fakeReviewQueueRepo struct {
	list func(context.Context, *time.Time, string, int) ([]domain.ReviewQueueItem, error)
}

func (f fakeReviewQueueRepo) ListReviewQueue(
	ctx context.Context,
	cursorQueuedAt *time.Time,
	cursorID string,
	limit int,
) ([]domain.ReviewQueueItem, error) {
	return f.list(ctx, cursorQueuedAt, cursorID, limit)
}

func TestReviewQueueList(t *testing.T) {
	queuedAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	var gotCursor *time.Time
	var gotID string
	svc := NewReviewQueue(fakeReviewQueueRepo{
		list: func(_ context.Context, cursor *time.Time, id string, _ int) ([]domain.ReviewQueueItem, error) {
			gotCursor, gotID = cursor, id
			return []domain.ReviewQueueItem{
				{ReviewID: "r1", QuestionID: "q1", QueuedAt: queuedAt},
				{ReviewID: "r2", QuestionID: "q2", QueuedAt: queuedAt.Add(time.Hour)},
			}, nil
		},
	})

	list, err := svc.List(context.Background(), ReviewQueueParams{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ReviewID != "r1" {
		t.Errorf("items = %+v", list.Items)
	}
	if list.NextCursor == nil {
		t.Fatal("next_cursor = nil, want non-nil")
	}

	next, err := svc.List(context.Background(), ReviewQueueParams{Cursor: *list.NextCursor, Limit: 1})
	if err != nil {
		t.Fatalf("List next page: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].ReviewID != "r1" {
		t.Errorf("next items = %+v", next.Items)
	}
	if gotCursor == nil || !gotCursor.Equal(queuedAt) || gotID != "r1" {
		t.Errorf("cursor = (%v, %q), want (%v, %q)", gotCursor, gotID, queuedAt, "r1")
	}
}
