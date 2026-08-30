package config

import "testing"

func TestLoad(t *testing.T) {
	const dsn = "postgres://codetrain:codetrain@postgres:5432/codetrain?sslmode=disable"

	t.Run("既定値", func(t *testing.T) {
		t.Setenv("DATABASE_URL", dsn)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want 8080", cfg.Port)
		}
		// 既定は dev。認証を触らない開発者が Cognito の疎通で詰まらないようにするため
		// （LOCAL_DEV.md §5.4）。
		if cfg.AuthMode != "dev" {
			t.Errorf("AuthMode = %q, want dev", cfg.AuthMode)
		}
	})

	t.Run("DATABASE_URL なしは起動させない", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if _, err := Load(); err == nil {
			t.Error("DATABASE_URL が無いのにエラーになりませんでした")
		}
	})

	t.Run("未知の AUTH_MODE を弾く", func(t *testing.T) {
		t.Setenv("DATABASE_URL", dsn)
		t.Setenv("AUTH_MODE", "none")
		if _, err := Load(); err == nil {
			t.Error("AUTH_MODE=none がエラーになりませんでした")
		}
	})

	t.Run("CORS_ALLOWED_ORIGINS はカンマ区切り", func(t *testing.T) {
		t.Setenv("DATABASE_URL", dsn)
		t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://localhost:3001 ,")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		want := []string{"http://localhost:3000", "http://localhost:3001"}
		if len(cfg.CORSAllowedOrigins) != len(want) {
			t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, want)
		}
		for i := range want {
			if cfg.CORSAllowedOrigins[i] != want[i] {
				t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], want[i])
			}
		}
	})
}
