package handler

import (
	"errors"
	"net/http"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
	"github.com/labstack/echo/v4"
)

func (h *Handler) Home(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}
	result, err := h.home.Get(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "ホームの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, result)
}
