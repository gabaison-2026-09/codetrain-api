package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type submitAttemptRequest struct {
	SelectedKeys []string `json:"selected_keys"`
	DurationMS   *int     `json:"duration_ms"`
}

// SubmitAttempt は POST /v1/questions/:id/attempts。認証必須。
func (h *Handler) SubmitAttempt(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return apperr.Validation("id は uuid 形式で指定してください")
	}

	var req submitAttemptRequest
	if err := c.Bind(&req); err != nil {
		return apperr.Validation("リクエストボディが不正です")
	}
	if len(req.SelectedKeys) == 0 {
		return apperr.Validation("selected_keys は1要素以上指定してください")
	}
	if req.DurationMS != nil && *req.DurationMS < 0 {
		return apperr.Validation("duration_ms は0以上で指定してください")
	}

	result, err := h.attempts.Submit(c.Request().Context(), sub, id, service.SubmitAttemptInput{
		SelectedKeys: req.SelectedKeys, DurationMS: req.DurationMS,
	})
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if errors.Is(err, service.ErrQuestionNotFound) {
		return apperr.New(apperr.CodeQuestionNotFound, http.StatusNotFound, "問題が見つかりません")
	}
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return err
		}
		return internalError(c, "回答の記録に失敗しました", err)
	}
	return c.JSON(http.StatusCreated, result)
}
