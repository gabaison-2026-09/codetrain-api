package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/paging"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// GetQuestion は GET /v1/questions/:id。認証必須。
func (h *Handler) GetQuestion(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		return apperr.Validation("id は uuid 形式で指定してください")
	}

	detail, err := h.questions.GetForUser(c.Request().Context(), sub, id)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if errors.Is(err, service.ErrQuestionNotFound) {
		return apperr.New(apperr.CodeQuestionNotFound, http.StatusNotFound, "問題が見つかりません")
	}
	if err != nil {
		return internalError(c, "問題の取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, detail)
}

// ListQuestions は GET /v1/questions。認証必須。
func (h *Handler) ListQuestions(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	params, err := parseQuestionSearch(c)
	if err != nil {
		return err
	}

	list, err := h.questions.List(c.Request().Context(), sub, params)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		var apiErr *apperr.APIError
		if errors.As(err, &apiErr) {
			return err
		}
		return internalError(c, "問題の検索に失敗しました", err)
	}
	return c.JSON(http.StatusOK, list)
}

func parseQuestionSearch(c echo.Context) (service.QuestionSearchParams, error) {
	page, err := paging.ParseParams(c)
	if err != nil {
		return service.QuestionSearchParams{}, err
	}

	p := service.QuestionSearchParams{
		SkillNodeID: c.QueryParam("skill_node_id"),
		Language:    c.QueryParam("language"),
		Q:           c.QueryParam("q"),
		Tags:        compactStrings(c.QueryParams()["tag"]),
		Cursor:      page.Cursor,
		Limit:       page.Limit,
	}

	if p.SkillNodeID != "" && !uuidPattern.MatchString(p.SkillNodeID) {
		return service.QuestionSearchParams{}, apperr.Validation("skill_node_id は uuid 形式で指定してください")
	}

	if raw := c.QueryParam("type"); raw != "" {
		if !domain.ValidQuestionType(raw) {
			return service.QuestionSearchParams{}, apperr.Validation("type が不正です")
		}
		p.Type = domain.QuestionType(raw)
	}

	if raw := c.QueryParam("difficulty"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 5 {
			return service.QuestionSearchParams{}, apperr.Validation("difficulty は 1〜5 で指定してください")
		}
		p.Difficulty = &n
	}

	if raw := c.QueryParam("unanswered_only"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return service.QuestionSearchParams{}, apperr.Validation("unanswered_only は true または false で指定してください")
		}
		p.UnansweredOnly = b
	}

	return p, nil
}

func compactStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
