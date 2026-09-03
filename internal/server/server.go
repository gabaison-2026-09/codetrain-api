// Package server は Echo インスタンスの生成・ミドルウェア設定・ルーティングを持つ。
//
// 「どの URL がどのハンドラに繋がるか」を1箇所で読めるようにするための層で、
// ビジネスロジックは持たない。
package server

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
	"github.com/gabaison-2026-09/codetrain-api/internal/config"
	"github.com/gabaison-2026-09/codetrain-api/internal/handler"
	"github.com/gabaison-2026-09/codetrain-api/internal/middleware"
)

// New はミドルウェアとルートを設定した Echo を返す。
// auth は認証が必要なルートに適用するミドルウェア（AUTH_MODE で実体が変わる）。
func New(cfg config.Config, h *handler.Handler, auth echo.MiddlewareFunc) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	// 失敗レスポンスを共通エンベロープ {"error":{"code","message"}} に統一する。
	e.HTTPErrorHandler = apperr.HTTPErrorHandler
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

	e.GET("/healthz", h.Health)

	v1 := e.Group("/v1")
	v1.GET("/skills", h.ListSkills)                           // 認証不要
	v1.GET("/me", h.Me, auth)                                 // 認証必須
	v1.POST("/me", h.CreateMe, auth)                          // 認証必須（JIT プロビジョニング）
	v1.GET("/task-slots/options", h.TaskOptions, auth)        // :slot_no より先に登録する
	v1.PUT("/task-slots/:slot_no", h.PutTaskSlot, auth)       // 認証必須
	v1.DELETE("/task-slots/:slot_no", h.DeleteTaskSlot, auth) // 認証必須
	v1.GET("/questions", h.ListQuestions, auth)               // 認証必須
	v1.GET("/questions/:id", h.GetQuestion, auth)             // 認証必須
	v1.POST("/questions/:id/attempts", h.SubmitAttempt, auth) // 認証必須
	v1.GET("/srs/due", h.SRSDue, auth)                        // 認証必須
	v1.PATCH("/me", h.UpdateMe, auth)                         // 認証必須
	v1.GET("/task-slots", h.ListTaskSlots, auth)              // 認証必須
	v1.GET("/calendar", h.Calendar, auth)                     // 認証必須
	v1.GET("/home", h.Home, auth)                             // 認証必須

	// /v1/admin/* は認証に加えてレビュアー権限を要求する。
	admin := v1.Group("/admin", auth, middleware.RequireReviewer(cfg))
	admin.GET("/questions", h.AdminListQuestions)
	admin.GET("/questions/:id", h.AdminGetQuestion)
	admin.POST("/questions/:id/review", h.AdminReview)
	admin.GET("/review-queue", h.AdminReviewQueue)

	return e
}
