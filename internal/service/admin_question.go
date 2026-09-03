package service

import (
	"context"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
)

type adminQuestionCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// AdminQuestion は管理者向け問題検索のユースケース。
type AdminQuestion struct {
	questions AdminQuestionRepository
}

func NewAdminQuestion(questions AdminQuestionRepository) *AdminQuestion {
	return &AdminQuestion{questions: questions}
}

// List は status を問わず問題を条件検索して1頁返す。
func (s *AdminQuestion) List(ctx context.Context, params AdminQuestionSearchParams) (AdminQuestionList, error) {
	var cursorCreatedAt *time.Time
	var cursorID string
	if params.Cursor != "" {
		var cursor adminQuestionCursor
		if err := paging.Decode(params.Cursor, &cursor); err != nil {
			return AdminQuestionList{}, err
		}
		if cursor.ID == "" {
			return AdminQuestionList{}, apperr.Validation("cursor が不正です")
		}
		cursorCreatedAt = &cursor.CreatedAt
		cursorID = cursor.ID
	}

	items, err := s.questions.SearchAdminQuestions(ctx, domain.AdminQuestionSearch{
		Status:          params.Status,
		Type:            params.Type,
		Language:        params.Language,
		SkillID:         params.SkillID,
		Query:           params.Q,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           params.Limit,
	})
	if err != nil {
		return AdminQuestionList{}, err
	}

	page, next, err := paging.Page(items, params.Limit, func(item domain.AdminQuestionSummary) any {
		return adminQuestionCursor{CreatedAt: item.CreatedAt, ID: item.ID}
	})
	if err != nil {
		return AdminQuestionList{}, err
	}
	return AdminQuestionList{Questions: page, NextCursor: next}, nil
}
