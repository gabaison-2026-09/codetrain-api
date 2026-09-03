package service

import (
	"context"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
)

type reviewQueueCursor struct {
	QueuedAt time.Time `json:"queued_at"`
	ID       string    `json:"id"`
}

// ReviewQueue は未レビュー問題キューのユースケース。
type ReviewQueue struct {
	reviews ReviewQueueRepository
}

func NewReviewQueue(reviews ReviewQueueRepository) *ReviewQueue {
	return &ReviewQueue{reviews: reviews}
}

// List は queued_at 昇順で未レビュー問題を1頁返す。
func (s *ReviewQueue) List(ctx context.Context, params ReviewQueueParams) (ReviewQueueList, error) {
	var cursorQueuedAt *time.Time
	var cursorID string
	if params.Cursor != "" {
		var cursor reviewQueueCursor
		if err := paging.Decode(params.Cursor, &cursor); err != nil {
			return ReviewQueueList{}, err
		}
		if cursor.ID == "" {
			return ReviewQueueList{}, apperr.Validation("cursor が不正です")
		}
		cursorQueuedAt = &cursor.QueuedAt
		cursorID = cursor.ID
	}

	items, err := s.reviews.ListReviewQueue(ctx, cursorQueuedAt, cursorID, params.Limit)
	if err != nil {
		return ReviewQueueList{}, err
	}

	page, next, err := paging.Page(items, params.Limit, func(item domain.ReviewQueueItem) any {
		return reviewQueueCursor{QueuedAt: item.QueuedAt, ID: item.ReviewID}
	})
	if err != nil {
		return ReviewQueueList{}, err
	}
	return ReviewQueueList{Items: page, NextCursor: next}, nil
}
