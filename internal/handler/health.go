package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Health は GET /healthz。
// LOCAL_DEV.md §5.2 のスモークテストが期待する {"status":"ok","db":"ok"} を返す。
// DB に到達できない場合は 503 と db:"error" を返し、起動しているだけの状態と区別する。
func (h *Handler) Health(c echo.Context) error {
	if err := h.health.Check(c.Request().Context()); err != nil {
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
