package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type fakeQuestions struct {
	list       func(ctx context.Context, externalID string, params service.QuestionSearchParams) (service.QuestionList, error)
	getForUser func(ctx context.Context, externalID, questionID string) (domain.QuestionDetail, error)
}

func (f fakeQuestions) List(ctx context.Context, externalID string, params service.QuestionSearchParams) (service.QuestionList, error) {
	return f.list(ctx, externalID, params)
}

func (f fakeQuestions) GetForUser(ctx context.Context, externalID, questionID string) (domain.QuestionDetail, error) {
	return f.getForUser(ctx, externalID, questionID)
}

func TestListQuestions(t *testing.T) {
	okList := service.QuestionList{
		Questions: []domain.QuestionSummary{
			{ID: "q1", Type: domain.QuestionTypeCodeReading, Difficulty: 2, Title: "配列メソッドの挙動",
				CodeLanguage: "typescript", Tags: []string{"array"}, Answered: false},
		},
	}

	t.Run("認証済みなら問題一覧を返す", func(t *testing.T) {
		var gotSub string
		var gotParams service.QuestionSearchParams
		h := New(Deps{Questions: fakeQuestions{
			list: func(_ context.Context, externalID string, params service.QuestionSearchParams) (service.QuestionList, error) {
				gotSub = externalID
				gotParams = params
				return okList, nil
			},
		}})
		rec := serve(t, h.ListQuestions, http.MethodGet, "/v1/questions?type=code_reading&difficulty=2&limit=10", "seed-user-01")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if gotSub != "seed-user-01" {
			t.Errorf("external_id = %q, want seed-user-01", gotSub)
		}
		if gotParams.Type != domain.QuestionTypeCodeReading {
			t.Errorf("Type = %q, want code_reading", gotParams.Type)
		}
		if gotParams.Difficulty == nil || *gotParams.Difficulty != 2 {
			t.Errorf("Difficulty = %v, want 2", gotParams.Difficulty)
		}
		if gotParams.Limit != 10 {
			t.Errorf("Limit = %d, want 10", gotParams.Limit)
		}

		var body service.QuestionList
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("レスポンスの解析に失敗: %v (body=%s)", err, rec.Body.String())
		}
		if len(body.Questions) != 1 || body.Questions[0].Title != "配列メソッドの挙動" {
			t.Errorf("questions = %+v", body.Questions)
		}
		if body.NextCursor != nil {
			t.Errorf("next_cursor = %v, want null", *body.NextCursor)
		}

		var raw map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatalf("raw 解析: %v", err)
		}
		qs, _ := raw["questions"].([]any)
		if len(qs) == 0 {
			t.Fatal("questions が空")
		}
		q, _ := qs[0].(map[string]any)
		for _, key := range []string{"body", "code", "choices", "correct_keys", "explanation"} {
			if _, ok := q[key]; ok {
				t.Errorf("一覧に %s が含まれている", key)
			}
		}
	})

	t.Run("0件でも questions は配列", func(t *testing.T) {
		h := New(Deps{Questions: fakeQuestions{
			list: func(context.Context, string, service.QuestionSearchParams) (service.QuestionList, error) {
				return service.QuestionList{Questions: []domain.QuestionSummary{}}, nil
			},
		}})
		rec := serve(t, h.ListQuestions, http.MethodGet, "/v1/questions", "seed-user-01")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
		}
		var body struct {
			Questions  []domain.QuestionSummary `json:"questions"`
			NextCursor *string                  `json:"next_cursor"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("解析: %v", err)
		}
		if body.Questions == nil {
			t.Error("questions = null, want []")
		}
	})

	t.Run("tag は複数受け取れる", func(t *testing.T) {
		var gotTags []string
		h := New(Deps{Questions: fakeQuestions{
			list: func(_ context.Context, _ string, params service.QuestionSearchParams) (service.QuestionList, error) {
				gotTags = params.Tags
				return service.QuestionList{Questions: []domain.QuestionSummary{}}, nil
			},
		}})
		serve(t, h.ListQuestions, http.MethodGet, "/v1/questions?tag=array&tag=async", "seed-user-01")
		if len(gotTags) != 2 || gotTags[0] != "array" || gotTags[1] != "async" {
			t.Errorf("tags = %v, want [array async]", gotTags)
		}
	})

	t.Run("unanswered_only=true を渡す", func(t *testing.T) {
		var got bool
		h := New(Deps{Questions: fakeQuestions{
			list: func(_ context.Context, _ string, params service.QuestionSearchParams) (service.QuestionList, error) {
				got = params.UnansweredOnly
				return service.QuestionList{Questions: []domain.QuestionSummary{}}, nil
			},
		}})
		serve(t, h.ListQuestions, http.MethodGet, "/v1/questions?unanswered_only=true", "seed-user-01")
		if !got {
			t.Error("UnansweredOnly = false, want true")
		}
	})

	tests := []struct {
		name       string
		sub        string
		path       string
		list       func(ctx context.Context, externalID string, params service.QuestionSearchParams) (service.QuestionList, error)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "sub が無ければ 401",
			sub:        "",
			path:       "/v1/questions",
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name: "ユーザーが無ければ 404",
			sub:  "no-such-user",
			path: "/v1/questions",
			list: func(context.Context, string, service.QuestionSearchParams) (service.QuestionList, error) {
				return service.QuestionList{}, service.ErrUserNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name:       "type が不正なら 400",
			sub:        "seed-user-01",
			path:       "/v1/questions?type=unknown",
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name:       "difficulty が範囲外なら 400",
			sub:        "seed-user-01",
			path:       "/v1/questions?difficulty=9",
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name:       "skill_node_id が uuid でなければ 400",
			sub:        "seed-user-01",
			path:       "/v1/questions?skill_node_id=not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name: "不正 cursor は 400",
			sub:  "seed-user-01",
			path: "/v1/questions?cursor=!!!",
			list: func(context.Context, string, service.QuestionSearchParams) (service.QuestionList, error) {
				return service.QuestionList{}, apperr.Validation("cursor が不正です")
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name:       "limit が非数なら 400",
			sub:        "seed-user-01",
			path:       "/v1/questions?limit=abc",
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name: "その他の失敗は 500",
			sub:  "seed-user-01",
			path: "/v1/questions",
			list: func(context.Context, string, service.QuestionSearchParams) (service.QuestionList, error) {
				return service.QuestionList{}, errors.New("DB 障害")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{Questions: fakeQuestions{list: tt.list}})
			rec := serve(t, h.ListQuestions, http.MethodGet, tt.path, tt.sub)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var env struct {
				Error struct {
					Status  int    `json:"status"`
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("エンベロープの解析に失敗: %v (body=%s)", err, rec.Body.String())
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", env.Error.Code, tt.wantCode)
			}
			if env.Error.Message == "" {
				t.Error("error.message が空")
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "DB 障害") {
				t.Errorf("500 応答に原因文字列が漏れている: %s", rec.Body.String())
			}
		})
	}
}

func TestGetQuestion(t *testing.T) {
	explanation := "map は各要素に関数を適用した新配列を返す"
	correctKeys := []string{"b"}
	answeredDetail := domain.QuestionDetail{
		ID:          "q1",
		Type:        domain.QuestionTypeCodeReading,
		Difficulty:  2,
		Title:       "配列メソッドの挙動",
		Body:        "次のコードの出力は？",
		Choices:     []domain.Choice{{Key: "a", Text: "[1,2,3]"}, {Key: "b", Text: "[2,4,6]"}},
		Tags:        []string{"array"},
		Answered:    true,
		CorrectKeys: &correctKeys,
		Explanation: &explanation,
	}
	unansweredDetail := domain.QuestionDetail{
		ID:         "q1",
		Type:       domain.QuestionTypeCodeReading,
		Difficulty: 2,
		Title:      "配列メソッドの挙動",
		Body:       "次のコードの出力は？",
		Choices:    []domain.Choice{{Key: "a", Text: "[1,2,3]"}, {Key: "b", Text: "[2,4,6]"}},
		Tags:       []string{"array"},
		Answered:   false,
	}

	testQID := "d0000000-0000-0000-0000-000000000001"

	t.Run("未回答の問題で correct_keys/explanation が null", func(t *testing.T) {
		h := New(Deps{Questions: fakeQuestions{
			getForUser: func(_ context.Context, sub, qID string) (domain.QuestionDetail, error) {
				if sub != "seed-user-01" || qID != testQID {
					t.Errorf("args = (%q, %q)", sub, qID)
				}
				return unansweredDetail, nil
			},
		}})
		rec := serveWithParam(t, h.GetQuestion, http.MethodGet, "/v1/questions/:id", testQID, "seed-user-01")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var raw map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatalf("解析: %v", err)
		}
		if raw["answered"] != false {
			t.Errorf("answered = %v, want false", raw["answered"])
		}
		if raw["correct_keys"] != nil {
			t.Errorf("correct_keys = %v, want null", raw["correct_keys"])
		}
		if raw["explanation"] != nil {
			t.Errorf("explanation = %v, want null", raw["explanation"])
		}
		if raw["body"] == nil {
			t.Error("body が含まれていない")
		}
		if raw["choices"] == nil {
			t.Error("choices が含まれていない")
		}
	})

	t.Run("回答済みの問題で correct_keys/explanation が入る", func(t *testing.T) {
		h := New(Deps{Questions: fakeQuestions{
			getForUser: func(context.Context, string, string) (domain.QuestionDetail, error) {
				return answeredDetail, nil
			},
		}})
		rec := serveWithParam(t, h.GetQuestion, http.MethodGet, "/v1/questions/:id", testQID, "seed-user-01")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
		}
		var raw map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatalf("解析: %v", err)
		}
		if raw["answered"] != true {
			t.Errorf("answered = %v, want true", raw["answered"])
		}
		ck, ok := raw["correct_keys"].([]any)
		if !ok || len(ck) != 1 || ck[0] != "b" {
			t.Errorf("correct_keys = %v, want [b]", raw["correct_keys"])
		}
		if raw["explanation"] != explanation {
			t.Errorf("explanation = %v, want %q", raw["explanation"], explanation)
		}
	})

	errTests := []struct {
		name       string
		sub        string
		paramID    string
		getForUser func(ctx context.Context, externalID, questionID string) (domain.QuestionDetail, error)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "sub が無ければ 401",
			sub:        "",
			paramID:    "a0000000-0000-0000-0000-000000000001",
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name:       "id が uuid でなければ 400",
			sub:        "seed-user-01",
			paramID:    "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeValidationError,
		},
		{
			name:    "ユーザーが無ければ 404 USER_NOT_FOUND",
			sub:     "no-such-user",
			paramID: "a0000000-0000-0000-0000-000000000001",
			getForUser: func(context.Context, string, string) (domain.QuestionDetail, error) {
				return domain.QuestionDetail{}, service.ErrUserNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name:    "問題が無ければ 404 QUESTION_NOT_FOUND",
			sub:     "seed-user-01",
			paramID: "a0000000-0000-0000-0000-000000000001",
			getForUser: func(context.Context, string, string) (domain.QuestionDetail, error) {
				return domain.QuestionDetail{}, service.ErrQuestionNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeQuestionNotFound,
		},
		{
			name:    "その他の失敗は 500",
			sub:     "seed-user-01",
			paramID: "a0000000-0000-0000-0000-000000000001",
			getForUser: func(context.Context, string, string) (domain.QuestionDetail, error) {
				return domain.QuestionDetail{}, errors.New("DB 障害")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{Questions: fakeQuestions{getForUser: tt.getForUser}})
			rec := serveWithParam(t, h.GetQuestion, http.MethodGet, "/v1/questions/:id", tt.paramID, tt.sub)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var env struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("エンベロープ解析: %v (body=%s)", err, rec.Body.String())
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", env.Error.Code, tt.wantCode)
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "DB 障害") {
				t.Errorf("500 応答に原因文字列が漏れている: %s", rec.Body.String())
			}
		})
	}
}
