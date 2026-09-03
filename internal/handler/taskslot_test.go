package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type fakeTaskOptions struct {
	list    func(context.Context, string) ([]domain.TaskOption, error)
	setSlot func(context.Context, string, domain.TaskConfig) (domain.TaskConfig, error)
}

func (f fakeTaskOptions) List(ctx context.Context, externalID string) ([]domain.TaskOption, error) {
	return f.list(ctx, externalID)
}

func (f fakeTaskOptions) SetSlot(ctx context.Context, externalID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
	return f.setSlot(ctx, externalID, slot)
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

func TestPutTaskSlot(t *testing.T) {
	t.Run("有効なスロットを保存して返す", func(t *testing.T) {
		h := New(Deps{TaskOptions: fakeTaskOptions{setSlot: func(_ context.Context, externalID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
			if externalID != "seed-user-01" {
				t.Errorf("externalID = %q", externalID)
			}
			if slot.SlotNo != 3 || slot.QuestionType != domain.QuestionTypeCodeReading || slot.Language != "typescript" || slot.Difficulty != nil {
				t.Errorf("slot = %+v", slot)
			}
			return slot, nil
		}}})

		rec := serveTaskSlot(t, h.PutTaskSlot, "3", "seed-user-01", `{"question_type":"code_reading","language":"typescript","difficulty":null}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var got domain.TaskConfig
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("response: %v", err)
		}
		if got.SlotNo != 3 || got.QuestionType != domain.QuestionTypeCodeReading || got.Language != "typescript" {
			t.Errorf("slot = %+v", got)
		}
	})

	tests := []struct {
		name       string
		sub        string
		slotNo     string
		body       string
		serviceErr error
		wantStatus int
		wantCode   string
	}{
		{name: "認証なしは 401", slotNo: "3", body: `{}`, wantStatus: http.StatusUnauthorized, wantCode: apperr.CodeUnauthorized},
		{name: "slot_no 6 は専用エラー", sub: "u", slotNo: "6", body: `{}`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeTaskSlotNoInvalid},
		{name: "slot_no が数値でなければ専用エラー", sub: "u", slotNo: "x", body: `{}`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeTaskSlotNoInvalid},
		{name: "JSON が不正なら 400", sub: "u", slotNo: "1", body: `{`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeValidationError},
		{name: "question_type が ENUM 外なら 400", sub: "u", slotNo: "1", body: `{"question_type":"essay"}`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeValidationError},
		{name: "difficulty 0 は 400", sub: "u", slotNo: "1", body: `{"question_type":"code_reading","difficulty":0}`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeValidationError},
		{name: "difficulty 6 は 400", sub: "u", slotNo: "1", body: `{"question_type":"code_reading","difficulty":6}`, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeValidationError},
		{name: "未登録ユーザーは 404", sub: "u", slotNo: "1", body: `{"question_type":"code_reading"}`, serviceErr: service.ErrUserNotFound, wantStatus: http.StatusNotFound, wantCode: apperr.CodeUserNotFound},
		{name: "候補に無い組み合わせは 422", sub: "u", slotNo: "1", body: `{"question_type":"code_reading"}`, serviceErr: service.ErrTaskSlotOptionInvalid, wantStatus: http.StatusUnprocessableEntity, wantCode: apperr.CodeTaskSlotOptionInvalid},
		{name: "service の slot_no エラーも 400", sub: "u", slotNo: "1", body: `{"question_type":"code_reading"}`, serviceErr: service.ErrTaskSlotNoInvalid, wantStatus: http.StatusBadRequest, wantCode: apperr.CodeTaskSlotNoInvalid},
		{name: "その他の失敗は 500", sub: "u", slotNo: "1", body: `{"question_type":"code_reading"}`, serviceErr: errors.New("DB 障害"), wantStatus: http.StatusInternalServerError, wantCode: apperr.CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := New(Deps{TaskOptions: fakeTaskOptions{setSlot: func(_ context.Context, _ string, slot domain.TaskConfig) (domain.TaskConfig, error) {
				called = true
				return slot, tt.serviceErr
			}}})
			rec := serveTaskSlot(t, h.PutTaskSlot, tt.slotNo, tt.sub, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.serviceErr == nil && tt.wantStatus != http.StatusOK && called {
				t.Error("入力エラー時に service が呼ばれた")
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

	t.Run("language 省略時は空文字", func(t *testing.T) {
		h := New(Deps{TaskOptions: fakeTaskOptions{setSlot: func(_ context.Context, _ string, slot domain.TaskConfig) (domain.TaskConfig, error) {
			if slot.Language != "" {
				t.Errorf("language = %q, want empty", slot.Language)
			}
			return slot, nil
		}}})
		rec := serveTaskSlot(t, h.PutTaskSlot, "1", "u", `{"question_type":"fill_in_blank","difficulty":null}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
	})
}

func serveTaskSlot(t *testing.T, h echo.HandlerFunc, slotNo, sub, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = apperr.HTTPErrorHandler
	req := httptest.NewRequest(http.MethodPut, "/v1/task-slots/"+slotNo, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/v1/task-slots/:slot_no")
	c.SetParamNames("slot_no")
	c.SetParamValues(slotNo)
	if sub != "" {
		c.Set(middleware.SubjectKey, sub)
	}
	if err := h(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

type fakeTaskSlotLister struct {
	list func(context.Context, string) ([]domain.TaskConfig, error)
}

func (f fakeTaskSlotLister) ListSlots(ctx context.Context, externalID string) ([]domain.TaskConfig, error) {
	return f.list(ctx, externalID)
}

func TestListTaskSlots(t *testing.T) {
	difficulty := 3
	h := New(Deps{TaskSlots: fakeTaskSlotLister{
		list: func(_ context.Context, externalID string) ([]domain.TaskConfig, error) {
			if externalID != "seed-user-01" {
				t.Errorf("externalID = %q", externalID)
			}
			return []domain.TaskConfig{
				{SlotNo: 1, QuestionType: domain.QuestionTypeCodeReading, Language: "typescript"},
				{SlotNo: 2, QuestionType: domain.QuestionTypeOutputPrediction, Difficulty: &difficulty},
			}, nil
		},
	}})
	rec := serve(t, h.ListTaskSlots, http.MethodGet, "/v1/task-slots", "seed-user-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Slots []struct {
			Difficulty *int `json:"difficulty"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Slots) != 2 || body.Slots[0].Difficulty != nil ||
		body.Slots[1].Difficulty == nil || *body.Slots[1].Difficulty != 3 {
		t.Fatalf("slots = %+v", body.Slots)
	}
}

func TestListTaskSlotsEmptyIsArray(t *testing.T) {
	h := New(Deps{TaskSlots: fakeTaskSlotLister{
		list: func(context.Context, string) ([]domain.TaskConfig, error) {
			return []domain.TaskConfig{}, nil
		},
	}})
	rec := serve(t, h.ListTaskSlots, http.MethodGet, "/v1/task-slots", "seed-user-01")
	if got := rec.Body.String(); got != "{\"slots\":[]}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestListTaskSlotsErrors(t *testing.T) {
	tests := []struct {
		name, sub string
		err       error
		status    int
		code      string
	}{
		{"unauthorized", "", nil, http.StatusUnauthorized, apperr.CodeUnauthorized},
		{"not found", "unknown", service.ErrUserNotFound, http.StatusNotFound, apperr.CodeUserNotFound},
		{"internal", "seed-user-01", errors.New("DB障害"), http.StatusInternalServerError, apperr.CodeInternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{TaskSlots: fakeTaskSlotLister{
				list: func(context.Context, string) ([]domain.TaskConfig, error) { return nil, tt.err },
			}})
			rec := serve(t, h.ListTaskSlots, http.MethodGet, "/v1/task-slots", tt.sub)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != tt.code {
				t.Errorf("error.code = %q, want %q", body.Error.Code, tt.code)
			}
		})
	}
}
