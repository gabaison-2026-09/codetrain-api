// Package config は環境変数から構成値を読む。
//
// LOCAL_DEV.md §5.3 のとおり、構成値はすべて環境変数で与える。
// ファイルや実行環境ごとの分岐をコードに書かない（Ministack と実 AWS の差も
// エンドポイントの差し替えだけで吸収する）。
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port     string
	AuthMode string // "dev" | "cognito"

	DatabaseURL string

	CognitoJWKSURL     string
	CognitoUserPoolID  string
	CognitoClientID    string
	CORSAllowedOrigins []string
	ReviewerSubs       []string

	AWSEndpointURL string
	AWSRegion      string
	S3Bucket       string
}

func Load() (Config, error) {
	cfg := Config{
		Port:               env("API_PORT", "8080"),
		AuthMode:           env("AUTH_MODE", "dev"),
		DatabaseURL:        env("DATABASE_URL", ""),
		CognitoJWKSURL:     env("COGNITO_JWKS_URL", ""),
		CognitoUserPoolID:  env("COGNITO_USER_POOL_ID", ""),
		CognitoClientID:    env("COGNITO_CLIENT_ID", ""),
		CORSAllowedOrigins: splitAndTrim(env("CORS_ALLOWED_ORIGINS", "")),
		ReviewerSubs:       splitAndTrim(env("REVIEWER_SUBS", "")),
		AWSEndpointURL:     env("AWS_ENDPOINT_URL", ""),
		AWSRegion:          env("AWS_REGION", "ap-northeast-1"),
		S3Bucket:           env("S3_BUCKET", ""),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL が設定されていません")
	}
	switch cfg.AuthMode {
	case "dev", "cognito":
	default:
		return Config{}, fmt.Errorf("AUTH_MODE は dev か cognito のいずれかです: %q", cfg.AuthMode)
	}

	return cfg, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
