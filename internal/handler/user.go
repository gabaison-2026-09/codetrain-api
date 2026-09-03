package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

// Me は GET /v1/me。認証必須。
// AUTH_MODE=dev では X-Dev-User の値が、cognito では JWT の sub が external_id になる。
func (h *Handler) Me(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	me, err := h.users.Me(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		// dev モードでは存在しない ID を渡せてしまうため、404 で明示的に返す。
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "ユーザーの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, me)
}

type createMeRequest struct {
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// CreateMe は POST /v1/me。認証必須。
// GET /v1/me が USER_NOT_FOUND を返したあとに 1 回だけ呼ぶ JIT プロビジョニング。
func (h *Handler) CreateMe(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	var req createMeRequest
	if err := c.Bind(&req); err != nil {
		return validationError("リクエストボディが不正です")
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return validationError("display_name は必須です")
	}

	in := service.CreateUserInput{
		DisplayName: displayName,
		AvatarURL:   normalizeOptionalString(req.AvatarURL),
	}
	me, err := h.users.Create(c.Request().Context(), sub, in)
	if errors.Is(err, service.ErrUserAlreadyProvisioned) {
		return apperr.New(apperr.CodeUserAlreadyProvisioned, http.StatusConflict, "ユーザーは既に作成済みです")
	}
	if err != nil {
		return internalError(c, "ユーザーの作成に失敗しました", err)
	}
	return c.JSON(http.StatusCreated, me)
}

// normalizeOptionalString は空文字を nil に揃える（未指定と同じ扱い）。
func normalizeOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
}
