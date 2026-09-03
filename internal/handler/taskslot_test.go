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
				t.Errorf("externalID = %q, want %q", externalID, "seed-user-01")
			}
			return []domain.TaskConfig{
				{
					SlotNo:       1,
					QuestionType: domain.QuestionTypeCodeReading,
					Language:     "typescript",
				},
				{
					SlotNo:       2,
					QuestionType: domain.QuestionTypeOutputPrediction,
					Difficulty:   &difficulty,
				},
			}, nil
		},
	}})

	rec := serve(t, h.ListTaskSlots, http.MethodGet, "/v1/task-slots", "seed-user-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Slots []struct {
			SlotNo     int  `json:"slot_no"`
			Difficulty *int `json:"difficulty"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if len(body.Slots) != 2 || body.Slots[0].Difficulty != nil {
		t.Fatalf("slots = %+v", body.Slots)
	}
	if body.Slots[1].Difficulty == nil || *body.Slots[1].Difficulty != 3 {
		t.Fatalf("difficulty = %v, want 3", body.Slots[1].Difficulty)
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
		t.Fatalf("body = %q, want %q", got, "{\"slots\":[]}\n")
	}
}

func TestListTaskSlotsErrors(t *testing.T) {
	tests := []struct {
		name       string
		sub        string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unauthorized",
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name:       "user not found",
			sub:        "unknown",
			err:        service.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name:       "internal error",
			sub:        "seed-user-01",
			err:        errors.New("DB障害"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{TaskSlots: fakeTaskSlotLister{
				list: func(context.Context, string) ([]domain.TaskConfig, error) {
					return nil, tt.err
				},
			}})
			rec := serve(t, h.ListTaskSlots, http.MethodGet, "/v1/task-slots", tt.sub)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("error response JSON: %v", err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", body.Error.Code, tt.wantCode)
			}
		})
	}
}
