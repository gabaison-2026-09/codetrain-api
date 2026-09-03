package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
)

// このファイルは handler 層のエラー返却を1箇所にまとめる。
//
// 方針: ハンドラは *apperr.APIError（または素の error）を return するだけ。
// 実際のレスポンス描画は server.New が登録する apperr.HTTPErrorHandler が行う。

// validationError は 400 VALIDATION_ERROR。
func validationError(msg string) *apperr.APIError { return apperr.Validation(msg) }

// notFoundError は 404 NOT_FOUND（個別コードが無い場合の汎用）。
func notFoundError(msg string) *apperr.APIError { return apperr.NotFound(msg) }

// internalError は「原因はログに残し、クライアントには詳細を出さない 500」をまとめたもの。
// 各ハンドラで同じログ出力とエラー生成を繰り返さないための共通化。
// 戻り値は INTERNAL_ERROR エンベロープに描画される（詳細文字列は返さない）。
func internalError(c echo.Context, msg string, err error) error {
	c.Logger().Errorf("%s: %v", msg, err)
	return apperr.Internal()
}
