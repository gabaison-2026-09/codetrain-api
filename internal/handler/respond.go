package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// internalError は「原因はログに残し、クライアントには詳細を出さない 500」をまとめたもの。
// 各ハンドラで同じログ出力とエラー生成を繰り返さないための共通化。
func internalError(c echo.Context, msg string, err error) error {
	c.Logger().Errorf("%s: %v", msg, err)
	return echo.NewHTTPError(http.StatusInternalServerError, msg)
}
