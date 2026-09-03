package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type fakeTaskOptions struct {
	list func(context.Context, string) ([]domain.TaskOption, error)
}

func (f fakeTaskOptions) List(ctx context.Context, externalID string) ([]domain.TaskOption, error) {
	return f.list(ctx, externalID)
}

func TestTaskOptions(t *testing.T) {
	tests := []struct {
		name       string
		sub        string
		list       func(context.Context, string) ([]domain.TaskOption, error)
		wantStatus int
		wantCode   string
	}{
		{
			name: "候補を返す",
			sub:  "seed-user-01",
			list: func(_ context.Context, externalID string) ([]domain.TaskOption, error) {
				if externalID != "seed-user-01" {
					t.Errorf("external_id = %q", externalID)
				}
				return []domain.TaskOption{{QuestionType: domain.QuestionTypeCodeReading, Language: "typescript", Difficulty: 1}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "sub が無ければ 401",
			sub:        "",
			list:       func(context.Context, string) ([]domain.TaskOption, error) { return nil, nil },
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name: "ユーザーが無ければ 404",
			sub:  "no-such-user",
			list: func(context.Context, string) ([]domain.TaskOption, error) {
				return nil, service.ErrUserNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name: "その他の失敗は 500",
			sub:  "seed-user-01",
			list: func(context.Context, string) ([]domain.TaskOption, error) {
				return nil, errors.New("DB 障害")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{TaskOptions: fakeTaskOptions{list: tt.list}})
			rec := serve(t, h.TaskOptions, http.MethodGet, "/v1/task-slots/options", tt.sub)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var body struct {
					Options []domain.TaskOption `json:"options"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("response: %v", err)
				}
				if len(body.Options) != 1 || body.Options[0].Difficulty != 1 {
					t.Errorf("options = %+v", body.Options)
				}
				return
			}

			var env struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("response: %v", err)
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", env.Error.Code, tt.wantCode)
			}
		})
	}

	t.Run("0件でも options は配列", func(t *testing.T) {
		h := New(Deps{TaskOptions: fakeTaskOptions{list: func(context.Context, string) ([]domain.TaskOption, error) {
			return []domain.TaskOption{}, nil
		}}})
		rec := serve(t, h.TaskOptions, http.MethodGet, "/v1/task-slots/options", "seed-user-01")
		if got := rec.Body.String(); got != "{\"options\":[]}\n" {
			t.Errorf("body = %q", got)
		}
	})
}
