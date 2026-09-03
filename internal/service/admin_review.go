package service

import (
	"context"
	"errors"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
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

// AdminReview はレビュー判定を記録するユースケース。
type AdminReview struct {
	reviews AdminReviewRepository
	users   UserLookupRepository
}

func NewAdminReview(reviews AdminReviewRepository, users UserLookupRepository) *AdminReview {
	return &AdminReview{reviews: reviews, users: users}
}

// Decide はレビュアーの sub を app_user.id に解決して判定を記録する。
func (s *AdminReview) Decide(
	ctx context.Context,
	reviewerSub, questionID string,
	in AdminReviewInput,
) (domain.ReviewResult, error) {
	switch in.Decision {
	case domain.ReviewDecisionApproved, domain.ReviewDecisionRejected, domain.ReviewDecisionNeedsEdit:
	default:
		return domain.ReviewResult{}, apperr.Validation("decision が不正です")
	}

	reviewerID, err := (userResolver{repo: s.users}).resolveUserID(ctx, reviewerSub)
	if err != nil {
		return domain.ReviewResult{}, err
	}

	result, err := s.reviews.DecideReview(ctx, reviewerID, questionID, in.Decision, in.Notes)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.ReviewResult{}, ErrQuestionNotFound
	}
	if errors.Is(err, repository.ErrAlreadyExists) {
		return domain.ReviewResult{}, ErrReviewAlreadyDecided
	}
	if err != nil {
		return domain.ReviewResult{}, err
	}
	return result, nil
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
