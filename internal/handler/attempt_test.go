package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
	"github.com/labstack/echo/v4"
)

type fakeAttempts struct {
	submit func(context.Context, string, string, service.SubmitAttemptInput) (domain.AttemptResult, error)
}

func serveJSONWithParam(t *testing.T, h echo.HandlerFunc, method, routePath, paramValue, sub, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = apperr.HTTPErrorHandler
	req := httptest.NewRequest(method, strings.Replace(routePath, ":id", paramValue, 1), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath(routePath)
	c.SetParamNames("id")
	c.SetParamValues(paramValue)
	if sub != "" {
		c.Set(middleware.SubjectKey, sub)
	}
	if err := h(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

func (f fakeAttempts) Submit(ctx context.Context, sub, questionID string, in service.SubmitAttemptInput) (domain.AttemptResult, error) {
	return f.submit(ctx, sub, questionID, in)
}

func TestSubmitAttempt(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	t.Run("有効な回答は201", func(t *testing.T) {
		h := New(Deps{Attempts: fakeAttempts{submit: func(_ context.Context, sub, qid string, in service.SubmitAttemptInput) (domain.AttemptResult, error) {
			if sub != "seed-user-01" || qid != id || len(in.SelectedKeys) != 1 || in.SelectedKeys[0] != "b" || in.DurationMS == nil || *in.DurationMS != 8200 {
				t.Errorf("sub=%q qid=%q in=%+v", sub, qid, in)
			}
			return domain.AttemptResult{AttemptID: "a1", IsCorrect: true, CorrectKeys: []string{"b"}, Explanation: "説明", XPGained: 10}, nil
		}}})
		rec := serveJSONWithParam(t, h.SubmitAttempt, http.MethodPost, "/v1/questions/:id/attempts", id, "seed-user-01", `{"selected_keys":["b"],"duration_ms":8200}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var got domain.AttemptResult
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.AttemptID != "a1" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	tests := []struct {
		name, body, sub, param string
		serviceErr             error
		status                 int
		code                   string
	}{
		{"未認証", `{}`, "", id, nil, 401, apperr.CodeUnauthorized},
		{"不正uuid", `{}`, "sub", "bad", nil, 400, apperr.CodeValidationError},
		{"空選択", `{"selected_keys":[]}`, "sub", id, nil, 400, apperr.CodeValidationError},
		{"負duration", `{"selected_keys":["a"],"duration_ms":-1}`, "sub", id, nil, 400, apperr.CodeValidationError},
		{"ユーザーなし", `{"selected_keys":["a"]}`, "sub", id, service.ErrUserNotFound, 404, apperr.CodeUserNotFound},
		{"問題なし", `{"selected_keys":["a"]}`, "sub", id, service.ErrQuestionNotFound, 404, apperr.CodeQuestionNotFound},
		{"DB障害", `{"selected_keys":["a"]}`, "sub", id, errors.New("db"), 500, apperr.CodeInternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{Attempts: fakeAttempts{submit: func(context.Context, string, string, service.SubmitAttemptInput) (domain.AttemptResult, error) {
				return domain.AttemptResult{}, tt.serviceErr
			}}})
			rec := serveJSONWithParam(t, h.SubmitAttempt, http.MethodPost, "/v1/questions/:id/attempts", tt.param, tt.sub, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.status, rec.Body.String())
			}
			var env struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error.Code != tt.code {
				t.Fatalf("code=%q err=%v", env.Error.Code, err)
			}
		})
	}
}
