// Cognito の JWT を検証する認証ミドルウェア。**本番と同じコードパス**。
//
// AUTH_MODE=cognito のときに使う（LOCAL_DEV.md §5.4）。
// JWKS を取得して署名・iss・aud（access token では client_id）・期限を検証する。
//
// 外部の JWKS キャッシュライブラリは使わず、必要な範囲を自前で持っている。
// 依存を増やさずに済むこと、鍵の再取得タイミングを明示的に制御したいことが理由。
package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/config"
)

// jwksMinRefreshInterval は、未知の kid を見たときに JWKS を再取得する最短間隔。
// 不正な kid を大量に投げられても取得元に負荷をかけないようにする。
const jwksMinRefreshInterval = time.Minute

type jwksCache struct {
	url    string
	client *http.Client

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastFetched time.Time
}

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
		keys:   map[string]*rsa.PublicKey{},
	}
}

func (c *jwksCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	k, ok := c.keys[kid]
	last := c.lastFetched
	c.mu.RUnlock()
	if ok {
		return k, nil
	}

	// 鍵はローテーションされうるため、未知の kid では一度だけ取り直す。
	if time.Since(last) < jwksMinRefreshInterval {
		return nil, fmt.Errorf("未知の kid: %s", kid)
	}
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("未知の kid: %s", kid)
}

func (c *jwksCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("JWKS の取得に失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS の取得に失敗: status=%d url=%s", resp.StatusCode, c.url)
	}

	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("JWKS の解析に失敗: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			return fmt.Errorf("JWK(kid=%s) の変換に失敗: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}

	c.mu.Lock()
	c.keys = keys
	c.lastFetched = time.Now()
	c.mu.Unlock()
	return nil
}

func rsaPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func newCognitoAuth(cfg config.Config) (echo.MiddlewareFunc, error) {
	if cfg.CognitoJWKSURL == "" {
		return nil, errors.New("AUTH_MODE=cognito には COGNITO_JWKS_URL が必要です")
	}
	cache := newJWKSCache(cfg.CognitoJWKSURL)

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if raw == "" {
				return ErrUnauthorized
			}

			claims := jwt.MapClaims{}
			_, err := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, errors.New("kid ヘッダがありません")
				}
				return cache.key(c.Request().Context(), kid)
			})
			if err != nil {
				c.Logger().Debugf("JWT の検証に失敗: %v", err)
				return ErrUnauthorized
			}

			if err := verifyIssuerAndAudience(claims, cfg); err != nil {
				c.Logger().Debugf("JWT のクレーム検証に失敗: %v", err)
				return ErrUnauthorized
			}

			sub, _ := claims["sub"].(string)
			if sub == "" {
				return ErrUnauthorized
			}
			setSubject(c, sub)
			return next(c)
		}
	}, nil
}

func verifyIssuerAndAudience(claims jwt.MapClaims, cfg config.Config) error {
	if cfg.CognitoUserPoolID != "" {
		iss, _ := claims["iss"].(string)
		if !strings.HasSuffix(iss, "/"+cfg.CognitoUserPoolID) {
			return fmt.Errorf("iss が User Pool と一致しません: %q", iss)
		}
	}

	if cfg.CognitoClientID == "" {
		return nil
	}
	// ID トークンは aud、アクセストークンは client_id にクライアント ID が入る。
	if aud, _ := claims["aud"].(string); aud == cfg.CognitoClientID {
		return nil
	}
	if cid, _ := claims["client_id"].(string); cid == cfg.CognitoClientID {
		return nil
	}
	return errors.New("aud / client_id がアプリクライアントと一致しません")
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}
