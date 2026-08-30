//go:build !dev_auth

package middleware

import (
	"strings"
	"testing"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// 本番と同じ条件（dev_auth タグなし）のビルドでは、
// AUTH_MODE=dev を指定しても認証ミドルウェアを組み立てられないこと。
//
// LOCAL_DEV.md §13-10 / OPEN_ISSUES D-22。
// go test（タグなし）で回すことで、この保証が壊れたら CI で気づける。
func TestDevAuthIsNotAvailableInProductionBuild(t *testing.T) {
	mw, err := newDevAuth(config.Config{AuthMode: "dev"})
	if err == nil {
		t.Fatal("dev_auth タグなしのビルドで dev 認証が組み立てられてしまいました")
	}
	if mw != nil {
		t.Error("エラー時にミドルウェアが返っています")
	}
	if !strings.Contains(err.Error(), "dev_auth") {
		t.Errorf("エラーメッセージに理由が含まれていません: %v", err)
	}
}

// 本番ビルドでは X-Dev-User を CORS で許可しない。
func TestExtraCORSHeadersIsEmptyInProductionBuild(t *testing.T) {
	if got := ExtraCORSHeaders(); len(got) != 0 {
		t.Errorf("ExtraCORSHeaders() = %v, want empty", got)
	}
}
