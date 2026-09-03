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

type adminQuestionCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// AdminQuestion は管理者向け問題検索のユースケース。
type AdminQuestion struct {
	questions AdminQuestionRepository
	details   AdminQuestionDetailRepository
	updater   AdminQuestionUpdateRepository
}

// AdminQuestionPatch は問題の部分更新内容。
type AdminQuestionPatch = domain.AdminQuestionPatch

func NewAdminQuestion(questions AdminQuestionRepository, details ...AdminQuestionDetailRepository) *AdminQuestion {
	var detailRepo AdminQuestionDetailRepository
	if len(details) > 0 {
		detailRepo = details[0]
	}
	return &AdminQuestion{questions: questions, details: detailRepo}
}

func NewAdminQuestionWithUpdater(
	questions AdminQuestionRepository,
	details AdminQuestionDetailRepository,
	updater AdminQuestionUpdateRepository,
) *AdminQuestion {
	return &AdminQuestion{questions: questions, details: details, updater: updater}
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

// Get は問題の全項目とレビュー履歴を返す。
func (s *AdminQuestion) Get(ctx context.Context, questionID string) (domain.AdminQuestion, error) {
	if s.details == nil {
		return domain.AdminQuestion{}, ErrQuestionNotFound
	}

	question, err := s.details.FindQuestionFull(ctx, questionID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.AdminQuestion{}, ErrQuestionNotFound
	}
	if err != nil {
		return domain.AdminQuestion{}, err
	}

	history, err := s.details.ListReviewHistory(ctx, questionID)
	if err != nil {
		return domain.AdminQuestion{}, err
	}
	if history == nil {
		history = []domain.ReviewEntry{}
	}
	question.ReviewHistory = history
	return question, nil
}

// Update は指定されたフィールドだけ問題を更新し、更新後の詳細とレビュー履歴を返す。
func (s *AdminQuestion) Update(ctx context.Context, questionID string, patch domain.AdminQuestionPatch) (domain.AdminQuestion, error) {
	if s.updater == nil || s.details == nil {
		return domain.AdminQuestion{}, ErrQuestionNotFound
	}

	current, err := s.details.FindQuestionFull(ctx, questionID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.AdminQuestion{}, ErrQuestionNotFound
	}
	if err != nil {
		return domain.AdminQuestion{}, err
	}

	choices := current.Choices
	if patch.Choices != nil {
		choices = *patch.Choices
	}
	correctKeys := current.CorrectKeys
	if patch.CorrectKeys != nil {
		correctKeys = *patch.CorrectKeys
	}
	validChoices := make(map[string]struct{}, len(choices))
	for _, choice := range choices {
		validChoices[choice.Key] = struct{}{}
	}
	for _, key := range correctKeys {
		if _, ok := validChoices[key]; !ok {
			return domain.AdminQuestion{}, apperr.Validation("correct_keys に choices に存在しない key が含まれています")
		}
	}

	updated, err := s.updater.UpdateQuestion(ctx, questionID, patch)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.AdminQuestion{}, ErrQuestionNotFound
	}
	if err != nil {
		return domain.AdminQuestion{}, err
	}
	history, err := s.details.ListReviewHistory(ctx, questionID)
	if err != nil {
		return domain.AdminQuestion{}, err
	}
	if history == nil {
		history = []domain.ReviewEntry{}
	}
	updated.ReviewHistory = history
	return updated, nil
}
