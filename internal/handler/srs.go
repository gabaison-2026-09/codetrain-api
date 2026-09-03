package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// SRSDue は GET /v1/srs/due。認証必須。
func (h *Handler) SRSDue(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}
	params, err := paging.ParseParams(c)
	if err != nil {
		return err
	}
	items, err := h.srs.ListDue(c.Request().Context(), sub, params.Limit)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "復習対象問題の取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"questions": items})
}
