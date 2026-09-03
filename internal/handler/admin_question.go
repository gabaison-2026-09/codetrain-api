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

// AdminListQuestions は GET /v1/admin/questions。認証とレビュアー権限が必須。
func (h *Handler) AdminListQuestions(c echo.Context) error {
	if _, ok := middleware.Subject(c); !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	params, err := parseAdminQuestionSearch(c)
	if err != nil {
		return err
	}

	list, err := h.admin.List(c.Request().Context(), params)
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return apiErr
		}
		return internalError(c, "管理者向け問題の検索に失敗しました", err)
	}
	return c.JSON(http.StatusOK, list)
}

// AdminGetQuestion は GET /v1/admin/questions/:id。認証とレビュアー権限が必須。
func (h *Handler) AdminGetQuestion(c echo.Context) error {
	if _, ok := middleware.Subject(c); !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return apperr.Validation("id は uuid 形式で指定してください")
	}

	question, err := h.adminGetter.Get(c.Request().Context(), id)
	if errors.Is(err, service.ErrQuestionNotFound) {
		return apperr.New(apperr.CodeQuestionNotFound, http.StatusNotFound, "問題が見つかりません")
	}
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return apiErr
		}
		return internalError(c, "管理者向け問題の取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, question)
}

func parseAdminQuestionSearch(c echo.Context) (service.AdminQuestionSearchParams, error) {
	page, err := paging.ParseParams(c)
	if err != nil {
		return service.AdminQuestionSearchParams{}, err
	}

	p := service.AdminQuestionSearchParams{
		Language: c.QueryParam("language"),
		SkillID:  c.QueryParam("skill_id"),
		Q:        c.QueryParam("q"),
		Cursor:   page.Cursor,
		Limit:    page.Limit,
	}

	if p.SkillID != "" && !uuidPattern.MatchString(p.SkillID) {
		return service.AdminQuestionSearchParams{}, apperr.Validation("skill_id は uuid 形式で指定してください")
	}
	if raw := c.QueryParam("status"); raw != "" {
		if !domain.ValidQuestionStatus(raw) {
			return service.AdminQuestionSearchParams{}, apperr.Validation("status が不正です")
		}
		p.Status = domain.QuestionStatus(raw)
	}
	if raw := c.QueryParam("type"); raw != "" {
		if !domain.ValidQuestionType(raw) {
			return service.AdminQuestionSearchParams{}, apperr.Validation("type が不正です")
		}
		p.Type = domain.QuestionType(raw)
	}
	return p, nil
}
