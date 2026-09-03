// Package apperr は API の失敗レスポンスを一元管理する。
//
// すべてのエラーは Document/API_DESIGN.md §1.1 / §4 が定める共通エンベロープ
//
//	{"error":{"code":"STRING_CODE","message":"日本語の説明文"}}
//
// で返す。handler 層は *APIError を return するだけでよく、実際の描画は
// HTTPErrorHandler が行う（server.New が Echo に登録する）。
package apperr

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// エラーコード（API_DESIGN.md §4）。
const (
	// 共通コード
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeInternalError   = "INTERNAL_ERROR"

	// 業務固有コード
	CodeUserNotFound           = "USER_NOT_FOUND"
	CodeUserAlreadyProvisioned = "USER_ALREADY_PROVISIONED"
	CodeQuestionNotFound       = "QUESTION_NOT_FOUND"
	CodeTaskSlotNoInvalid      = "TASK_SLOT_NO_INVALID"
	CodeTaskSlotOptionInvalid  = "TASK_SLOT_OPTION_INVALID"
	CodeNoAvailableQuestion    = "NO_AVAILABLE_QUESTION"
	CodeReviewAlreadyDecided   = "REVIEW_ALREADY_DECIDED"
)

// internalMessage は 500 応答でクライアントに見せる固定文言。
// 原因（元の error）はログにのみ出し、詳細は返さない。
const internalMessage = "サーバ内部エラーが発生しました"

// APIError はクライアントに返す業務エラー。code / HTTP ステータス / 表示文言を持つ。
type APIError struct {
	Code       string
	HTTPStatus int
	Message    string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// New は任意のコード・ステータス・メッセージで APIError を作る。
func New(code string, status int, msg string) *APIError {
	return &APIError{Code: code, HTTPStatus: status, Message: msg}
}

// よく使うコンストラクタ。

func Validation(msg string) *APIError { return New(CodeValidationError, http.StatusBadRequest, msg) }
func Unauthorized(msg string) *APIError {
	return New(CodeUnauthorized, http.StatusUnauthorized, msg)
}
func Forbidden(msg string) *APIError { return New(CodeForbidden, http.StatusForbidden, msg) }
func NotFound(msg string) *APIError  { return New(CodeNotFound, http.StatusNotFound, msg) }
func Conflict(msg string) *APIError  { return New(CodeConflict, http.StatusConflict, msg) }

// Internal は原因を伏せた 500。メッセージは固定。
func Internal() *APIError {
	return New(CodeInternalError, http.StatusInternalServerError, internalMessage)
}

// envelope はレスポンス JSON の形。
type envelope struct {
	Error body `json:"error"`
}

type body struct {
	Status  int    `json:"status"` // HTTP ステータスコード（数値）
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorHandler は echo.HTTPErrorHandler として登録する。
// すべての失敗レスポンスを共通エンベロープに統一する。
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status, code, msg := classify(err)

	// 500 系は原因をログにのみ残す。
	if status >= http.StatusInternalServerError {
		c.Logger().Errorf("internal error: %v", err)
	}

	var writeErr error
	if c.Request().Method == http.MethodHead {
		writeErr = c.NoContent(status)
	} else {
		writeErr = c.JSON(status, envelope{Error: body{Status: status, Code: code, Message: msg}})
	}
	if writeErr != nil {
		c.Logger().Errorf("エラーレスポンスの送信に失敗: %v", writeErr)
	}
}

// classify は error からステータス・コード・表示文言を決める。
func classify(err error) (status int, code, msg string) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPStatus >= http.StatusInternalServerError {
			return apiErr.HTTPStatus, CodeInternalError, internalMessage
		}
		return apiErr.HTTPStatus, apiErr.Code, apiErr.Message
	}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		status = he.Code
		msg = messageOf(he)
		if status >= http.StatusInternalServerError {
			return status, CodeInternalError, internalMessage
		}
		return status, codeForStatus(status), msg
	}

	// 素の error は 500 扱い。
	return http.StatusInternalServerError, CodeInternalError, internalMessage
}

// messageOf は echo.HTTPError の Message を文字列として取り出す。
func messageOf(he *echo.HTTPError) string {
	if s, ok := he.Message.(string); ok && s != "" {
		return s
	}
	return http.StatusText(he.Code)
}

// codeForStatus は個別コードが無い場合に HTTP ステータスから共通コードを引く。
func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeValidationError
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	default:
		return CodeInternalError
	}
}
