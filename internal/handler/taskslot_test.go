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
