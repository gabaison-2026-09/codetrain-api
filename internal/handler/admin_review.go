package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type adminReviewRequest struct {
	Decision string `json:"decision"`
	Notes    string `json:"notes"`
}

// AdminReview は POST /v1/admin/questions/:id/review。認証とレビュアー権限が必須。
func (h *Handler) AdminReview(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return apperr.Validation("id は uuid 形式で指定してください")
	}

	var req adminReviewRequest
	if err := c.Bind(&req); err != nil {
		return apperr.Validation("リクエストボディが不正です")
	}
	decision := domain.ReviewDecision(req.Decision)
	switch decision {
	case domain.ReviewDecisionApproved, domain.ReviewDecisionRejected, domain.ReviewDecisionNeedsEdit:
	default:
		return apperr.Validation("decision が不正です")
	}

	result, err := h.reviewer.Decide(c.Request().Context(), sub, id, service.AdminReviewInput{
		Decision: decision,
		Notes:    req.Notes,
	})
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "レビュアーが見つかりません")
	}
	if errors.Is(err, service.ErrQuestionNotFound) {
		return apperr.New(apperr.CodeQuestionNotFound, http.StatusNotFound, "問題が見つかりません")
	}
	if errors.Is(err, service.ErrReviewAlreadyDecided) {
		return apperr.New(apperr.CodeReviewAlreadyDecided, http.StatusConflict, "レビューは既に判定済みです")
	}
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return apiErr
		}
		return internalError(c, "レビュー判定の記録に失敗しました", err)
	}
	return c.JSON(http.StatusCreated, result)
}

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
