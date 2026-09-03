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

type fakeMeStats struct {
	stats func(ctx context.Context, externalID string) ([]domain.TypeStat, error)
}

func (f fakeMeStats) Stats(ctx context.Context, externalID string) ([]domain.TypeStat, error) {
	return f.stats(ctx, externalID)
}

func intPtr(v int) *int { return &v }

// MeStats の契約:
//
//	認証済みでなければ 401、ユーザーがいなければ 404、それ以外の失敗は 500。
//	成功時は {"stats":[...]} を API_DESIGN.md §3 の形で返す。
func TestMeStats(t *testing.T) {
	okStats := []domain.TypeStat{
		{QuestionType: domain.QuestionTypeCodeReading, Language: "typescript", Attempts: 42, Corrects: 35, Accuracy: 0.83, LastDifficulty: intPtr(3)},
		{QuestionType: domain.QuestionTypeOutputPrediction, Language: "", Attempts: 10, Corrects: 6, Accuracy: 0.6, LastDifficulty: nil},
	}

	tests := []struct {
		name       string
		sub        string
		stats      func(ctx context.Context, externalID string) ([]domain.TypeStat, error)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "認証済みなら統計を返す",
			sub:        "seed-user-01",
			stats:      func(context.Context, string) ([]domain.TypeStat, error) { return okStats, nil },
			wantStatus: http.StatusOK,
		},
		{
			name:       "sub が無ければ 401",
			sub:        "",
			stats:      func(context.Context, string) ([]domain.TypeStat, error) { return okStats, nil },
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name: "ユーザーが無ければ 404",
			sub:  "no-such-user",
			stats: func(context.Context, string) ([]domain.TypeStat, error) {
				return nil, service.ErrUserNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name: "その他の失敗は 500",
			sub:  "seed-user-01",
			stats: func(context.Context, string) ([]domain.TypeStat, error) {
				return nil, errors.New("DB 障害")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Deps{Stats: fakeMeStats{stats: tt.stats}})
			rec := serve(t, h.MeStats, http.MethodGet, "/v1/me/stats", tt.sub)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				var env struct {
					Error struct {
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
				if tt.wantStatus == http.StatusInternalServerError && strings.Contains(env.Error.Message, "DB 障害") {
					t.Errorf("500 応答に原因文字列が漏れている: %q", env.Error.Message)
				}
				return
			}
			var body struct {
				Stats []domain.TypeStat `json:"stats"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("レスポンスの解析に失敗: %v (body=%s)", err, rec.Body.String())
			}
			if len(body.Stats) != 2 {
				t.Fatalf("stats = %+v, want 2 件", body.Stats)
			}
			if body.Stats[0].QuestionType != domain.QuestionTypeCodeReading ||
				body.Stats[0].Language != "typescript" ||
				body.Stats[0].Attempts != 42 ||
				body.Stats[0].Corrects != 35 ||
				body.Stats[0].LastDifficulty == nil || *body.Stats[0].LastDifficulty != 3 {
				t.Errorf("stats[0] = %+v", body.Stats[0])
			}
			if body.Stats[1].Language != "" || body.Stats[1].LastDifficulty != nil {
				t.Errorf("stats[1] = %+v, want language=\"\" last_difficulty=null", body.Stats[1])
			}
		})
	}
}

func TestMeStatsEmptyIsArray(t *testing.T) {
	h := New(Deps{Stats: fakeMeStats{
		stats: func(context.Context, string) ([]domain.TypeStat, error) { return []domain.TypeStat{}, nil },
	}})
	rec := serve(t, h.MeStats, http.MethodGet, "/v1/me/stats", "seed-user-01")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "{\"stats\":[]}\n" {
		t.Errorf("body = %q, want %q", got, "{\"stats\":[]}\n")
	}
}

func TestMeStatsUsesSubjectAsExternalID(t *testing.T) {
	var got string
	h := New(Deps{Stats: fakeMeStats{
		stats: func(_ context.Context, externalID string) ([]domain.TypeStat, error) {
			got = externalID
			return []domain.TypeStat{}, nil
		},
	}})

	serve(t, h.MeStats, http.MethodGet, "/v1/me/stats", "seed-user-02")

	if got != "seed-user-02" {
		t.Errorf("service に渡した external_id = %q, want %q", got, "seed-user-02")
	}
}
