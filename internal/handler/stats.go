package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// MeStats は GET /v1/me/stats。認証必須。
// 種別×言語別の累計回答数・正答率を返す（プロフィール画面用）。
func (h *Handler) MeStats(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}
	items, err := h.stats.Stats(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "統計の取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"stats": items})
}
