package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// RequireReviewer は認証済み subject がレビュアーであることを要求する。
//
// 現在は REVIEWER_SUBS の許可リストで判定する。
// TODO(C-5/D-2): 権限モデル確定後に本実装へ差し替える。
func RequireReviewer(cfg config.Config) echo.MiddlewareFunc {
	reviewerSubs := make(map[string]struct{}, len(cfg.ReviewerSubs))
	for _, sub := range cfg.ReviewerSubs {
		reviewerSubs[sub] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sub, ok := Subject(c)
			if !ok {
				return ErrUnauthorized
			}
			if _, ok := reviewerSubs[sub]; !ok {
				return echo.NewHTTPError(http.StatusForbidden, "レビュアー権限が必要です")
			}
			return next(c)
		}
	}
}
