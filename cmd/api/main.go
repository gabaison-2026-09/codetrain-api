// codetrain-api — 配信 API（Go + Echo）。
//
// ローカルでは docker compose の api サービスとして air 経由で起動する
// （LOCAL_DEV.md §4.2「ビルド・実行はすべてコンテナ内」）。
//
// このファイルの役割は各層の配線（DI）とプロセスのライフサイクル管理だけ。
// 依存の組み立てが1箇所で読めるように、ここに集約している。
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

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
	"github.com/gabaison-2026-09/codetrain-api/internal/handler"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
	"github.com/gabaison-2026-09/codetrain-api/internal/server"
	"github.com/gabaison-2026-09/codetrain-api/internal/service"
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

	repo, err := repository.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	auth, err := middleware.NewAuth(cfg)
	if err != nil {
		return err
	}

	adminQuestions := service.NewAdminQuestionWithUpdater(repo, repo, repo)
	adminReview := service.NewAdminReview(repo, repo)
	h := handler.New(handler.Deps{
		Health:       service.NewHealth(repo),
		Skills:       service.NewSkill(repo),
		Users:        service.NewUser(repo),
		TaskSlots:    service.NewTaskSlot(repo, repo),
		TaskOptions:  service.NewTaskOptions(repo),
		Questions:    service.NewQuestion(repo, repo),
		Admin:        adminQuestions,
		AdminGetter:  adminQuestions,
		AdminUpdater: adminQuestions,
		ReviewQueue:  service.NewReviewQueue(repo),
		Reviewer:     adminReview,
		SRS:          service.NewSRS(repo, repo),
		Calendar:     service.NewCalendar(repo, repo),
		Attempts:     service.NewAttempt(repo, repo, repo),
		Home:         service.NewHome(repo, repo),
	})

	e := server.New(cfg, h, auth)

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
