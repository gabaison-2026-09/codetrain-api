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

// GetForUser は published 問題を1件返す。
// answered=false なら correct_keys / explanation を null にして返す。
// 未登録ユーザーは ErrUserNotFound、該当問題がなければ ErrQuestionNotFound。
func (s *Question) GetForUser(ctx context.Context, externalID, questionID string) (domain.QuestionDetail, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return domain.QuestionDetail{}, err
	}

	q, answered, err := s.questions.FindPublishedByID(ctx, userID, questionID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.QuestionDetail{}, ErrQuestionNotFound
	}
	if err != nil {
		return domain.QuestionDetail{}, err
	}

	detail := domain.QuestionDetail{
		ID:           q.ID,
		SkillNodeID:  q.SkillNodeID,
		Type:         q.Type,
		Difficulty:   q.Difficulty,
		Title:        q.Title,
		Body:         q.Body,
		Code:         q.Code,
		CodeLanguage: q.CodeLanguage,
		Choices:      q.Choices,
		Tags:         q.Tags,
		Attribution:  q.Attribution,
		Answered:     answered,
	}

	if answered {
		detail.CorrectKeys = &q.CorrectKeys
		detail.Explanation = &q.Explanation
	}
	// answered=false の場合 CorrectKeys / Explanation は nil → JSON null

	return detail, nil
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
