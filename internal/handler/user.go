package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// Me は GET /v1/me。認証必須。
// AUTH_MODE=dev では X-Dev-User の値が、cognito では JWT の sub が external_id になる。
func (h *Handler) Me(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return middleware.ErrUnauthorized
	}

	me, err := h.users.Me(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		// dev モードでは存在しない ID を渡せてしまうため、404 で明示的に返す。
		return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "ユーザーの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, me)
}
