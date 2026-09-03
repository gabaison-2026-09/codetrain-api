package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"
	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
)

type putTaskSlotRequest struct {
	QuestionType domain.QuestionType `json:"question_type"`
	Language     string              `json:"language"`
	Difficulty   *int                `json:"difficulty"`
}

// ListTaskSlots は GET /v1/task-slots。認証必須。
func (h *Handler) ListTaskSlots(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	slots, err := h.taskSlots.ListSlots(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "タスクスロットの取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"slots": slots})
}

// TaskOptions は GET /v1/task-slots/options。認証必須。
func (h *Handler) TaskOptions(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	options, err := h.taskOptions.List(c.Request().Context(), sub)
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if err != nil {
		return internalError(c, "タスク候補の取得に失敗しました", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"options": options})
}

// PutTaskSlot は PUT /v1/task-slots/:slot_no。認証必須。
func (h *Handler) PutTaskSlot(c echo.Context) error {
	sub, ok := middleware.Subject(c)
	if !ok {
		return apperr.Unauthorized("認証が必要です")
	}

	slotNo, err := strconv.Atoi(c.Param("slot_no"))
	if err != nil || slotNo < 1 || slotNo > 5 {
		return apperr.New(apperr.CodeTaskSlotNoInvalid, http.StatusBadRequest, "slot_no は 1〜5 で指定してください")
	}

	var req putTaskSlotRequest
	if err := c.Bind(&req); err != nil {
		return apperr.Validation("リクエストボディが不正です")
	}
	if !domain.ValidQuestionType(string(req.QuestionType)) {
		return apperr.Validation("question_type が不正です")
	}
	if req.Difficulty != nil && (*req.Difficulty < 1 || *req.Difficulty > 5) {
		return apperr.Validation("difficulty は null または 1〜5 で指定してください")
	}

	slot, err := h.taskOptions.SetSlot(c.Request().Context(), sub, domain.TaskConfig{
		SlotNo:       slotNo,
		QuestionType: req.QuestionType,
		Language:     req.Language,
		Difficulty:   req.Difficulty,
	})
	if errors.Is(err, service.ErrUserNotFound) {
		return apperr.New(apperr.CodeUserNotFound, http.StatusNotFound, "ユーザーが見つかりません: "+sub)
	}
	if errors.Is(err, service.ErrTaskSlotNoInvalid) {
		return apperr.New(apperr.CodeTaskSlotNoInvalid, http.StatusBadRequest, "slot_no は 1〜5 で指定してください")
	}
	if errors.Is(err, service.ErrTaskSlotOptionInvalid) {
		return apperr.New(apperr.CodeTaskSlotOptionInvalid, http.StatusUnprocessableEntity, "指定されたタスク条件は利用できません")
	}
	if err != nil {
		return internalError(c, "タスクスロットの保存に失敗しました", err)
	}
	return c.JSON(http.StatusOK, slot)
}
