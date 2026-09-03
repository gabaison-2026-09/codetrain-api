package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

type fakeQuestionRepo struct {
	search func(ctx context.Context, userID string, q domain.QuestionSearch) ([]domain.QuestionSummary, error)
}

func (f fakeQuestionRepo) SearchQuestions(ctx context.Context, userID string, q domain.QuestionSearch) ([]domain.QuestionSummary, error) {
	return f.search(ctx, userID, q)
}

func TestQuestionList(t *testing.T) {
	user := fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1", ExternalID: "seed-user-01"}, domain.Progress{}, nil
		},
	}
	t1 := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(-time.Hour)
	summaries := []domain.QuestionSummary{
		{ID: "q1", Type: domain.QuestionTypeCodeReading, Title: "A", CreatedAt: t1},
		{ID: "q2", Type: domain.QuestionTypeBugFinding, Title: "B", CreatedAt: t2},
	}

	t.Run("ユーザーを解決して検索し1頁を返す", func(t *testing.T) {
		var gotUserID string
		var gotQ domain.QuestionSearch
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(_ context.Context, userID string, q domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				gotUserID = userID
				gotQ = q
				return summaries, nil
			},
		})

		lang := "javascript"
		diff := 2
		got, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{
			Type:           domain.QuestionTypeCodeReading,
			Language:       lang,
			Difficulty:     &diff,
			Tags:           []string{"array"},
			Q:              "配列",
			UnansweredOnly: true,
			Limit:          20,
		})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if gotUserID != "u1" {
			t.Errorf("repository に渡した userID = %q, want u1", gotUserID)
		}
		if gotQ.Type != domain.QuestionTypeCodeReading || gotQ.Language != lang || gotQ.Query != "配列" {
			t.Errorf("search = %+v", gotQ)
		}
		if gotQ.Difficulty == nil || *gotQ.Difficulty != 2 {
			t.Errorf("Difficulty = %v, want 2", gotQ.Difficulty)
		}
		if !gotQ.UnansweredOnly {
			t.Error("UnansweredOnly = false, want true")
		}
		if len(got.Questions) != 2 || got.Questions[0].ID != "q1" {
			t.Errorf("questions = %+v", got.Questions)
		}
		if got.NextCursor != nil {
			t.Errorf("next_cursor = %v, want nil", *got.NextCursor)
		}
	})

	t.Run("limit+1 件なら next_cursor を付ける", func(t *testing.T) {
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(context.Context, string, domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				return summaries, nil
			},
		})

		got, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{Limit: 1})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got.Questions) != 1 || got.Questions[0].ID != "q1" {
			t.Errorf("questions = %+v, want q1 の1件", got.Questions)
		}
		if got.NextCursor == nil {
			t.Fatal("next_cursor = nil, want 非 nil")
		}
		var cur questionCursor
		if err := paging.Decode(*got.NextCursor, &cur); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if cur.ID != "q1" || !cur.CreatedAt.Equal(t1) {
			t.Errorf("cursor = %+v, want q1 / t1", cur)
		}
	})

	t.Run("0件なら空スライス", func(t *testing.T) {
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(context.Context, string, domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				return nil, nil
			},
		})

		got, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{Limit: 20})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got.Questions == nil {
			t.Error("Questions = nil, want 空スライス")
		}
		if len(got.Questions) != 0 {
			t.Errorf("questions = %d 件, want 0", len(got.Questions))
		}
		if got.NextCursor != nil {
			t.Errorf("next_cursor = %v, want nil", *got.NextCursor)
		}
	})

	t.Run("cursor をデコードして repository に渡す", func(t *testing.T) {
		encoded, err := paging.Encode(questionCursor{CreatedAt: t1, ID: "q1"})
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		var gotQ domain.QuestionSearch
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(_ context.Context, _ string, q domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				gotQ = q
				return []domain.QuestionSummary{}, nil
			},
		})

		if _, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{
			Cursor: encoded,
			Limit:  20,
		}); err != nil {
			t.Fatalf("List: %v", err)
		}
		if gotQ.CursorCreatedAt == nil || !gotQ.CursorCreatedAt.Equal(t1) || gotQ.CursorID != "q1" {
			t.Errorf("cursor = created_at=%v id=%q", gotQ.CursorCreatedAt, gotQ.CursorID)
		}
	})

	t.Run("不正な cursor は VALIDATION_ERROR", func(t *testing.T) {
		searchCalled := false
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(context.Context, string, domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				searchCalled = true
				return nil, nil
			},
		})

		_, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{
			Cursor: "!!!not-base64!!!",
			Limit:  20,
		})
		var apiErr *apperr.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != apperr.CodeValidationError {
			t.Errorf("err = %v, want VALIDATION_ERROR", err)
		}
		if searchCalled {
			t.Error("不正 cursor のときは repository を呼ばないこと")
		}
	})

	t.Run("ErrNotFound を ErrUserNotFound に翻訳する", func(t *testing.T) {
		svc := NewQuestion(fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
		}, fakeQuestionRepo{
			search: func(context.Context, string, domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				t.Fatal("未登録なら検索しない")
				return nil, nil
			},
		})

		_, err := svc.List(context.Background(), "no-such-user", QuestionSearchParams{Limit: 20})
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
	})

	t.Run("repository のエラーを伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		svc := NewQuestion(user, fakeQuestionRepo{
			search: func(context.Context, string, domain.QuestionSearch) ([]domain.QuestionSummary, error) {
				return nil, wantErr
			},
		})

		_, err := svc.List(context.Background(), "seed-user-01", QuestionSearchParams{Limit: 20})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
