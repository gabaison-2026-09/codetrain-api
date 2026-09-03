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

// AdminReviewQueue は GET /v1/admin/review-queue。認証とレビュアー権限が必須。
func (h *Handler) AdminReviewQueue(c echo.Context) error {
	if _, ok := middleware.Subject(c); !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	page, err := paging.ParseParams(c)
	if err != nil {
		return err
	}
	list, err := h.reviewQueue.List(c.Request().Context(), service.ReviewQueueParams{
		Cursor: page.Cursor,
		Limit:  page.Limit,
	})
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return apiErr
		}
		return internalError(c, "レビューキューの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, list)
}
