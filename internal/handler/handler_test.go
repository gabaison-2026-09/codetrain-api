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

// 各 service の interface に対するフェイク。関数フィールドをテストごとに差し替える。

type fakeHealth struct {
	check func(ctx context.Context) error
}

func (f fakeHealth) Check(ctx context.Context) error { return f.check(ctx) }

type fakeSkills struct {
	list func(ctx context.Context) ([]domain.Skill, error)
}

func (f fakeSkills) List(ctx context.Context) ([]domain.Skill, error) { return f.list(ctx) }

type fakeUsers struct {
	me func(ctx context.Context, externalID string) (service.UserWithProgress, error)
}

func (f fakeUsers) Me(ctx context.Context, externalID string) (service.UserWithProgress, error) {
	return f.me(ctx, externalID)
}

// serve はハンドラを1回呼び、Echo のエラーハンドラを通したうえで結果を返す。
// sub が空でなければ認証済みのコンテキストとして組み立てる。
func serve(t *testing.T, h echo.HandlerFunc, method, path, sub string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.HTTPErrorHandler = apperr.HTTPErrorHandler
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if sub != "" {
		c.Set(middleware.SubjectKey, sub)
	}

	if err := h(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

// Health の契約: DB に到達できれば 200 と db:"ok"、できなければ 503 と db:"error"。
func TestHealth(t *testing.T) {
	tests := []struct {
		name            string
		pingErr         error
		wantStatus      int
		wantStatusField string
		wantDB          string
	}{
		{name: "DB 到達可", pingErr: nil, wantStatus: http.StatusOK, wantStatusField: "ok", wantDB: "ok"},
		{name: "DB 不通", pingErr: errors.New("接続断"), wantStatus: http.StatusServiceUnavailable, wantStatusField: "degraded", wantDB: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(fakeHealth{check: func(context.Context) error { return tt.pingErr }}, nil, nil)
			rec := serve(t, h.Health, http.MethodGet, "/healthz", "")

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("レスポンスの解析に失敗: %v (body=%s)", err, rec.Body.String())
			}
			if body["status"] != tt.wantStatusField {
				t.Errorf(`body["status"] = %q, want %q`, body["status"], tt.wantStatusField)
			}
			if body["db"] != tt.wantDB {
				t.Errorf(`body["db"] = %q, want %q`, body["db"], tt.wantDB)
			}
		})
	}
}

// ListSkills の契約: 200 で {"skills":[...]}。service が失敗したら 500。
func TestListSkills(t *testing.T) {
	t.Run("スキルを返す", func(t *testing.T) {
		h := New(nil, fakeSkills{
			list: func(context.Context) ([]domain.Skill, error) {
				return []domain.Skill{{ID: "s1", Slug: "go", Name: "Go"}}, nil
			},
		}, nil)
		rec := serve(t, h.ListSkills, http.MethodGet, "/v1/skills", "")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var body struct {
			Skills []domain.Skill `json:"skills"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("レスポンスの解析に失敗: %v (body=%s)", err, rec.Body.String())
		}
		if len(body.Skills) != 1 || body.Skills[0].Slug != "go" {
			t.Errorf("skills = %+v, want slug=go の1件", body.Skills)
		}
	})

	t.Run("スキルが0件でも skills キーは配列", func(t *testing.T) {
		h := New(nil, fakeSkills{
			list: func(context.Context) ([]domain.Skill, error) { return []domain.Skill{}, nil },
		}, nil)
		rec := serve(t, h.ListSkills, http.MethodGet, "/v1/skills", "")

		if got := rec.Body.String(); got != "{\"skills\":[]}\n" {
			t.Errorf("body = %q, want %q", got, "{\"skills\":[]}\n")
		}
	})

	t.Run("service が失敗したら 500", func(t *testing.T) {
		h := New(nil, fakeSkills{
			list: func(context.Context) ([]domain.Skill, error) { return nil, errors.New("DB 障害") },
		}, nil)
		rec := serve(t, h.ListSkills, http.MethodGet, "/v1/skills", "")

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

// Me の契約:
//
//	認証済みでなければ 401、ユーザーがいなければ 404、それ以外の失敗は 500。
func TestMe(t *testing.T) {
	okUser := service.UserWithProgress{
		User:     domain.User{ID: "u1", ExternalID: "seed-user-01", DisplayName: "テスト"},
		Progress: domain.Progress{XP: 120},
	}

	tests := []struct {
		name       string
		sub        string
		me         func(ctx context.Context, externalID string) (service.UserWithProgress, error)
		wantStatus int
		wantCode   string // 失敗時のエラーエンベロープ code。空なら検証しない。
	}{
		{
			name:       "認証済みならユーザーを返す",
			sub:        "seed-user-01",
			me:         func(context.Context, string) (service.UserWithProgress, error) { return okUser, nil },
			wantStatus: http.StatusOK,
		},
		{
			name:       "sub が無ければ 401",
			sub:        "",
			me:         func(context.Context, string) (service.UserWithProgress, error) { return okUser, nil },
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthorized,
		},
		{
			name: "ユーザーが無ければ 404",
			sub:  "no-such-user",
			me: func(context.Context, string) (service.UserWithProgress, error) {
				return service.UserWithProgress{}, service.ErrUserNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeUserNotFound,
		},
		{
			name: "その他の失敗は 500",
			sub:  "seed-user-01",
			me: func(context.Context, string) (service.UserWithProgress, error) {
				return service.UserWithProgress{}, errors.New("DB 障害")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(nil, nil, fakeUsers{me: tt.me})
			rec := serve(t, h.Me, http.MethodGet, "/v1/me", tt.sub)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if tt.wantCode == "" {
					return
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
				if env.Error.Status != tt.wantStatus {
					t.Errorf("error.status = %d, want %d", env.Error.Status, tt.wantStatus)
				}
				if env.Error.Code != tt.wantCode {
					t.Errorf("error.code = %q, want %q", env.Error.Code, tt.wantCode)
				}
				if env.Error.Message == "" {
					t.Errorf("error.message が空")
				}
				if tt.wantStatus == http.StatusInternalServerError && strings.Contains(env.Error.Message, "DB 障害") {
					t.Errorf("500 応答に原因文字列が漏れている: %q", env.Error.Message)
				}
				return
			}
			var body service.UserWithProgress
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("レスポンスの解析に失敗: %v (body=%s)", err, rec.Body.String())
			}
			if body.User.ExternalID != "seed-user-01" || body.Progress.XP != 120 {
				t.Errorf("body = %+v, want %+v", body, okUser)
			}
		})
	}
}

// sub の受け渡しが middleware と handler で噛み合っていることを確認する。
// Subject が読める形でしか Me は service を呼ばない。
func TestMeUsesSubjectAsExternalID(t *testing.T) {
	var got string
	h := New(nil, nil, fakeUsers{
		me: func(_ context.Context, externalID string) (service.UserWithProgress, error) {
			got = externalID
			return service.UserWithProgress{}, nil
		},
	})

	serve(t, h.Me, http.MethodGet, "/v1/me", "seed-user-02")

	if got != "seed-user-02" {
		t.Errorf("service に渡した external_id = %q, want %q", got, "seed-user-02")
	}
}
