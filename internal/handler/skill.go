package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListSkills は GET /v1/skills。認証不要。
func (h *Handler) ListSkills(c echo.Context) error {
	skills, err := h.skills.List(c.Request().Context())
	if err != nil {
		return internalError(c, "スキルの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"skills": skills})
}
