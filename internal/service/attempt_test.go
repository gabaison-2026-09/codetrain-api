package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
)

type fakeAttemptRepo struct {
	record func(context.Context, domain.Attempt, domain.Question, int) (domain.AttemptResult, error)
}

func (f fakeAttemptRepo) RecordAttempt(ctx context.Context, a domain.Attempt, q domain.Question, xp int) (domain.AttemptResult, error) {
	return f.record(ctx, a, q, xp)
}

func TestAttemptSubmit(t *testing.T) {
	question := domain.Question{
		ID: "q1", Type: domain.QuestionTypeCodeReading, Difficulty: 2,
		CodeLanguage: "go", Choices: []domain.Choice{{Key: "a"}, {Key: "b"}},
		CorrectKeys: []string{"b"}, Explanation: "説明",
	}
	users := fakeUserRepo{find: func(context.Context, string) (domain.User, domain.Progress, error) {
		return domain.User{ID: "u1"}, domain.Progress{}, nil
	}}
	questions := fakeQuestionRepo{findPublishedBy: func(context.Context, string, string) (domain.Question, bool, error) {
		return question, false, nil
	}}

	t.Run("集合が一致する正解を採点してXPを付与する", func(t *testing.T) {
		called := false
		svc := NewAttempt(users, questions, fakeAttemptRepo{record: func(_ context.Context, a domain.Attempt, q domain.Question, xp int) (domain.AttemptResult, error) {
			called = true
			if a.UserID != "u1" || a.QuestionID != "q1" || !a.IsCorrect || xp != 10 {
				t.Errorf("attempt=%+v xp=%d", a, xp)
			}
			return domain.AttemptResult{AttemptID: "a1", IsCorrect: true, XPGained: xp}, nil
		}})
		got, err := svc.Submit(context.Background(), "sub", "q1", SubmitAttemptInput{SelectedKeys: []string{"b", "b"}})
		if err != nil || !called || !got.IsCorrect || got.XPGained != 10 {
			t.Fatalf("got=%+v called=%v err=%v", got, called, err)
		}
	})

	t.Run("不正解はXPゼロ", func(t *testing.T) {
		svc := NewAttempt(users, questions, fakeAttemptRepo{record: func(_ context.Context, a domain.Attempt, _ domain.Question, xp int) (domain.AttemptResult, error) {
			if a.IsCorrect || xp != 0 {
				t.Errorf("isCorrect=%v xp=%d", a.IsCorrect, xp)
			}
			return domain.AttemptResult{IsCorrect: a.IsCorrect, XPGained: xp}, nil
		}})
		if _, err := svc.Submit(context.Background(), "sub", "q1", SubmitAttemptInput{SelectedKeys: []string{"a"}}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("存在しない選択肢はVALIDATION_ERROR", func(t *testing.T) {
		svc := NewAttempt(users, questions, fakeAttemptRepo{record: func(context.Context, domain.Attempt, domain.Question, int) (domain.AttemptResult, error) {
			t.Fatal("validation error では記録しない")
			return domain.AttemptResult{}, nil
		}})
		_, err := svc.Submit(context.Background(), "sub", "q1", SubmitAttemptInput{SelectedKeys: []string{"zzz"}})
		var apiErr *apperr.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != apperr.CodeValidationError {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("問題が無ければErrQuestionNotFound", func(t *testing.T) {
		svc := NewAttempt(users, fakeQuestionRepo{findPublishedBy: func(context.Context, string, string) (domain.Question, bool, error) {
			return domain.Question{}, false, repository.ErrNotFound
		}}, fakeAttemptRepo{})
		_, err := svc.Submit(context.Background(), "sub", "q1", SubmitAttemptInput{SelectedKeys: []string{"a"}})
		if !errors.Is(err, ErrQuestionNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
