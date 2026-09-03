package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// ListTaskSlots は GET /v1/task-slots。認証必須。
func (h *Handler) ListTaskSlots(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	slots, err := h.taskSlots.ListSlots(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(
			apperr.CodeUserNotFound,
			http.StatusNotFound,
			"ユーザーが見つかりません: "+sub,
		)
	}
	if err != nil {
		return internalError(c, "タスクスロットの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"slots": slots})
}
