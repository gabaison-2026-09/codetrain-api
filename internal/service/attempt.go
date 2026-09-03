package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

// Attempt は回答の検証と採点を行い、永続化を repository に委譲する。
type Attempt struct {
	userResolver
	questions QuestionRepository
	attempts  AttemptRepository
}

func NewAttempt(users UserRepository, questions QuestionRepository, attempts AttemptRepository) *Attempt {
	return &Attempt{userResolver: userResolver{repo: users}, questions: questions, attempts: attempts}
}

func (s *Attempt) Submit(ctx context.Context, externalID, questionID string, in SubmitAttemptInput) (domain.AttemptResult, error) {
	userID, err := s.resolveUserID(ctx, externalID)
	if err != nil {
		return domain.AttemptResult{}, err
	}

	q, _, err := s.questions.FindPublishedByID(ctx, userID, questionID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.AttemptResult{}, ErrQuestionNotFound
	}
	if err != nil {
		return domain.AttemptResult{}, err
	}

	valid := make(map[string]struct{}, len(q.Choices))
	for _, choice := range q.Choices {
		valid[choice.Key] = struct{}{}
	}
	for _, key := range in.SelectedKeys {
		if _, ok := valid[key]; !ok {
			return domain.AttemptResult{}, apperr.Validation("selected_keys に問題の選択肢に存在しない key が含まれています")
		}
	}

	isCorrect := sameStringSet(in.SelectedKeys, q.CorrectKeys)
	xpGained := 0
	if isCorrect {
		xpGained = 10 // TODO(B-3): XP 配点表の確定後に置き換える。
	}

	return s.attempts.RecordAttempt(ctx, domain.Attempt{
		UserID: userID, QuestionID: questionID, SelectedKeys: in.SelectedKeys,
		IsCorrect: isCorrect, DurationMS: in.DurationMS,
	}, q, xpGained)
}

func sameStringSet(a, b []string) bool {
	aSet := make(map[string]struct{}, len(a))
	bSet := make(map[string]struct{}, len(b))
	for _, value := range a {
		aSet[value] = struct{}{}
	}
	for _, value := range b {
		bSet[value] = struct{}{}
	}
	if len(aSet) != len(bSet) {
		return false
	}
	for value := range aSet {
		if _, ok := bSet[value]; !ok {
			return false
		}
	}
	return true
}
