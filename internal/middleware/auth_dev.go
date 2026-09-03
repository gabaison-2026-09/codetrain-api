//go:build dev_auth

// ローカル開発専用の認証。**本番ビルドには含まれない。**
//
// ビルドタグ dev_auth を付けたときだけコンパイルされる（LOCAL_DEV.md §5.4）。
// Cognito を起動せずにアプリ機能の開発を進められるようにするためのもので、
// 署名検証を一切行わない。
package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// devUserHeader に入っている値を、そのまま認証済みユーザーの sub として扱う。
//
//	curl -H "X-Dev-User: seed-user-01" http://localhost:8080/v1/me
const devUserHeader = "X-Dev-User"

const devEmailHeader = "X-Dev-Email"

// ExtraCORSHeaders は dev モードで追加する CORS 許可ヘッダを返す。
// admin（ブラウザ）から X-Dev-User を付けて API を叩けるようにするためのもので、
// この文字列自体も本番バイナリに残さない。
func ExtraCORSHeaders() []string {
	return []string{devUserHeader, devEmailHeader}
}

func newDevAuth(_ config.Config) (echo.MiddlewareFunc, error) {
	slog.Warn("AUTH_MODE=dev で起動しています。署名検証は行われません（ローカル開発専用）",
		"header", devUserHeader)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sub := c.Request().Header.Get(devUserHeader)
			if sub == "" {
				return ErrUnauthorized
			}
			setSubject(c, sub)
			if email := c.Request().Header.Get(devEmailHeader); email != "" {
				setEmail(c, email)
			}
			return next(c)
		}
	}, nil
}
