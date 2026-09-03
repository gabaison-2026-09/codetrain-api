package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

func TestRequireReviewer(t *testing.T) {
	mw := RequireReviewer(config.Config{ReviewerSubs: []string{"reviewer-01"}})

	tests := []struct {
		name       string
		sub        string
		setSubject bool
		wantStatus int
	}{
		{name: "許可リスト内", sub: "reviewer-01", setSubject: true, wantStatus: http.StatusOK},
		{name: "許可リスト外", sub: "user-01", setSubject: true, wantStatus: http.StatusForbidden},
		{name: "認証なし", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/v1/admin/questions", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.setSubject {
				c.Set(SubjectKey, tt.sub)
			}

			nextCalled := false
			h := mw(func(c echo.Context) error {
				nextCalled = true
				return c.NoContent(http.StatusOK)
			})
			if err := h(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != (tt.wantStatus == http.StatusOK) {
				t.Errorf("nextCalled = %t, want %t", nextCalled, tt.wantStatus == http.StatusOK)
			}
		})
	}
}
