//go:build !dev_auth

// dev_auth タグなしのビルド（＝本番イメージ）向けのスタブ。
//
// AUTH_MODE=dev が設定されていても**起動時に失敗させる**。
// 認証を無効化する経路が本番バイナリに存在しないことを、コンパイル時に保証する。
package middleware

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// ExtraCORSHeaders は本番ビルドでは何も追加しない。
func ExtraCORSHeaders() []string { return nil }

func newDevAuth(_ config.Config) (echo.MiddlewareFunc, error) {
	return nil, errors.New(
		"AUTH_MODE=dev はこのビルドでは利用できません（dev_auth タグなしでビルドされています）")
}
