// Package handler は配信 API のハンドラを持つ。
//
// 今回のスコープはローカル環境が立ち上がったことを確認できる最小限
// （/healthz・/v1/skills・/v1/me）。MVP ループの実装は Phase 2 で追加する。
package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/store"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler { return &Handler{store: s} }

// Health は GET /healthz。
// LOCAL_DEV.md §5.2 のスモークテストが期待する {"status":"ok","db":"ok"} を返す。
// DB に到達できない場合は 503 と db:"error" を返し、起動しているだけの状態と区別する。
func (h *Handler) Health(c echo.Context) error {
	if err := h.store.Ping(c.Request().Context()); err != nil {
		c.Logger().Errorf("healthz: DB に到達できません: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "degraded",
			"db":     "error",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"db":     "ok",
	})
}

// ListSkills は GET /v1/skills。認証不要。
func (h *Handler) ListSkills(c echo.Context) error {
	skills, err := h.store.ListSkills(c.Request().Context())
	if err != nil {
		c.Logger().Errorf("skills の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "スキルの取得に失敗しました")
	}
	return c.JSON(http.StatusOK, map[string]any{"skills": skills})
}

// Me は GET /v1/me。認証必須。
// AUTH_MODE=dev では X-Dev-User の値が、cognito では JWT の sub が external_id になる。
func (h *Handler) Me(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return middleware.ErrUnauthorized
	}

	me, err := h.store.FindUserByExternalID(c.Request().Context(), sub)
	if errors.Is(err, store.ErrNotFound) {
		// dev モードでは存在しない ID を渡せてしまうため、404 で明示的に返す。
		return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		c.Logger().Errorf("me の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの取得に失敗しました")
	}
	return c.JSON(http.StatusOK, me)
}
