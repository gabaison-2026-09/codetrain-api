// Package middleware は Echo のミドルウェアを提供する。
//
// 認証は LOCAL_DEV.md §5.4 / OPEN_ISSUES D-22 の方針に従い、2モードを持つ。
//
//	cognito : JWKS を取得して署名・aud・iss・期限を検証する（本番と同じコードパス）
//	dev     : 署名検証をせず X-Dev-User ヘッダを sub として扱う（ローカル専用）
//
// dev モードの実装は **ビルドタグ dev_auth で分離**されており、
// タグなしのビルド（＝本番イメージ）にはコンパイルされない。
// 「環境変数を間違えて本番で認証が無効化される」経路を、設定ミスではなく
// コンパイル時に潰すのが目的。
package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// SubjectKey は認証済みユーザーの sub を Echo のコンテキストに置くときのキー。
// 値の設定はこのパッケージの認証ミドルウェアだけが行い、読み出しは Subject を使う。
// 公開しているのは、handler のテストが認証済みのコンテキストを組み立てられるようにするため。
const SubjectKey = "codetrain.subject"

// EmailKey は認証トークンから取得した email を Echo のコンテキストに置くときのキー。
const EmailKey = "codetrain.email"

// ErrUnauthorized は認証に失敗したことを表す。
var ErrUnauthorized = echo.NewHTTPError(http.StatusUnauthorized, "認証が必要です")

// Subject は認証済みユーザーの sub を返す。
func Subject(c echo.Context) (string, bool) {
	v, ok := c.Get(SubjectKey).(string)
	return v, ok && v != ""
}

// Email は認証トークンから取得した email を返す。
// トークンに email が含まれない場合は空文字と false を返す。
func Email(c echo.Context) (string, bool) {
	v, ok := c.Get(EmailKey).(string)
	return v, ok && v != ""
}

func setSubject(c echo.Context, sub string) {
	c.Set(SubjectKey, sub)
}

func setEmail(c echo.Context, email string) {
	c.Set(EmailKey, email)
}

// NewAuth は AUTH_MODE に応じた認証ミドルウェアを返す。
func NewAuth(cfg config.Config) (echo.MiddlewareFunc, error) {
	switch cfg.AuthMode {
	case "cognito":
		return newCognitoAuth(cfg)
	case "dev":
		// newDevAuth の実体はビルドタグで切り替わる。
		//   dev_auth あり : auth_dev.go        （動作する）
		//   dev_auth なし : auth_dev_disabled.go（起動時にエラーを返す）
		return newDevAuth(cfg)
	default:
		return nil, errors.New("未知の AUTH_MODE: " + cfg.AuthMode)
	}
}
