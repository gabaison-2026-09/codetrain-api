package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
	"github.com/labstack/echo/v4"
)

func (h *Handler) Calendar(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}
	from, to := c.QueryParam("from"), c.QueryParam("to")
	if !validDateRange(from, to) {
		return validationError("from/to は YYYY-MM-DD の有効な期間で指定してください")
	}
	result, err := h.calendar.Get(c.Request().Context(), sub, from, to)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "カレンダーの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, result)
}

func validDateRange(from, to string) bool {
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || from == "" || to == "" || f.After(t) {
		return false
	}
	return t.Sub(f).Hours()/24+1 <= 366
}
