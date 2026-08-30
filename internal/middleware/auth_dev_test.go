//go:build dev_auth

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// dev モードの契約:
//
//	X-Dev-User があれば、その値を sub として通す。無ければ 401。
func TestDevAuth(t *testing.T) {
	mw, err := newDevAuth(config.Config{AuthMode: "dev"})
	if err != nil {
		t.Fatalf("newDevAuth: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantSub    string
	}{
		{name: "ヘッダあり", header: "seed-user-01", wantStatus: http.StatusOK, wantSub: "seed-user-01"},
		{name: "ヘッダなし", header: "", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
			if tt.header != "" {
				req.Header.Set(devUserHeader, tt.header)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var gotSub string
			h := mw(func(c echo.Context) error {
				gotSub, _ = Subject(c)
				return c.NoContent(http.StatusOK)
			})

			err := h(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if gotSub != tt.wantSub {
				t.Errorf("subject = %q, want %q", gotSub, tt.wantSub)
			}
		})
	}
}

// dev_auth タグ付きのビルドでは、CORS 許可ヘッダに X-Dev-User が足される。
// タグなしのビルドでは空になること（auth_dev_disabled_test.go）と対にしている。
func TestExtraCORSHeadersWithDevAuth(t *testing.T) {
	got := ExtraCORSHeaders()
	if len(got) != 1 || got[0] != devUserHeader {
		t.Errorf("ExtraCORSHeaders() = %v, want [%s]", got, devUserHeader)
	}
}
