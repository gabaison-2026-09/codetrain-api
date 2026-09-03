package service

import (
	"context"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
)

// questionCursor は created_at DESC, id DESC の並び替えキー。
type questionCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// Question は問題検索のユースケース。
type Question struct {
	userResolver
	questions QuestionRepository
}

func NewQuestion(users UserRepository, questions QuestionRepository) *Question {
	return &Question{userResolver: userResolver{repo: users}, questions: questions}
}

// List は published 問題を条件検索して1頁返す。
// 未登録ユーザーは ErrUserNotFound。不正な cursor は VALIDATION_ERROR。
func (s *Question) List(ctx context.Context, externalID string, params QuestionSearchParams) (QuestionList, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return QuestionList{}, err
	}

	q := domain.QuestionSearch{
		SkillNodeID:    params.SkillNodeID,
		Type:           params.Type,
		Language:       params.Language,
		Difficulty:     params.Difficulty,
		Tags:           params.Tags,
		Query:          params.Q,
		UnansweredOnly: params.UnansweredOnly,
		Limit:          params.Limit,
	}
	if params.Cursor != "" {
		var cur questionCursor
		if err := paging.Decode(params.Cursor, &cur); err != nil {
			return QuestionList{}, err
		}
		if cur.ID == "" {
			return QuestionList{}, apperr.Validation("cursor が不正です")
		}
		q.CursorCreatedAt = &cur.CreatedAt
		q.CursorID = cur.ID
	}

	items, err := s.questions.SearchQuestions(ctx, userID, q)
	if err != nil {
		return QuestionList{}, err
	}

	page, next, err := paging.Page(items, params.Limit, func(item domain.QuestionSummary) any {
		return questionCursor{CreatedAt: item.CreatedAt, ID: item.ID}
	})
	if err != nil {
		return QuestionList{}, err
	}
	return QuestionList{Questions: page, NextCursor: next}, nil
}
