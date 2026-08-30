// codetrain-api — 配信 API（Go + Echo）。
//
// ローカルでは docker compose の api サービスとして air 経由で起動する
// （LOCAL_DEV.md §4.2「ビルド・実行はすべてコンテナ内」）。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
	"github.com/gabaison-2026-09/codetrain-api/internal/handler"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("起動に失敗しました", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	auth, err := middleware.NewAuth(cfg)
	if err != nil {
		return err
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(echomw.Logger())

	// CORS は admin（http://localhost:3000）からのアクセスのために開発時だけ緩める。
	// 許可オリジンは環境変数で渡す（LOCAL_DEV.md §4.2）。
	if len(cfg.CORSAllowedOrigins) > 0 {
		allowHeaders := []string{
			echo.HeaderOrigin, echo.HeaderContentType,
			echo.HeaderAccept, echo.HeaderAuthorization,
		}
		// dev モード固有のヘッダ（X-Dev-User）はビルドタグ側から受け取る。
		// 本番ビルドでは空になり、ヘッダ名すらバイナリに残らない。
		allowHeaders = append(allowHeaders, middleware.ExtraCORSHeaders()...)

		e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
			AllowOrigins: cfg.CORSAllowedOrigins,
			AllowHeaders: allowHeaders,
		}))
	}

	h := handler.New(st)

	e.GET("/healthz", h.Health)

	v1 := e.Group("/v1")
	v1.GET("/skills", h.ListSkills) // 認証不要
	v1.GET("/me", h.Me, auth)       // 認証必須

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("codetrain-api を起動しました", "port", cfg.Port, "auth_mode", cfg.AuthMode)
		if err := e.StartServer(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("サーバが異常終了しました", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("シャットダウンします")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return e.Shutdown(shutdownCtx)
}
