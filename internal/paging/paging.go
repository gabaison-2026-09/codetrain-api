// Package paging はカーソルページングの共通処理を提供する。
//
// Document/API_DESIGN.md §1 の規約に従う:
//
//	クエリ: cursor（任意）/ limit（default 20, max 100）
//	レスポンス: next_cursor（次頁なしは null）
//
// domain に依存しない。呼び出し側（service）が並び替えキーの型を決め、
// Encode / Decode / Page を組み合わせる。
package paging

import (
	"encoding/base64"
	"encoding/json"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Params は一覧系エンドポイントのページングクエリ。
type Params struct {
	Cursor string
	Limit  int
}

// ParseParams は echo.Context から cursor / limit を読む。
//
//	limit 未指定 → 20
//	limit は 1..100 に clamp（0 は 1、100 超は 100）
//	負や非数 → VALIDATION_ERROR
//	cursor はそのまま返す（中身の検証は Decode 側）
func ParseParams(c echo.Context) (Params, error) {
	p := Params{
		Cursor: c.QueryParam("cursor"),
		Limit:  defaultLimit,
	}

	if raw := c.QueryParam("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return Params{}, apperr.Validation("limit は整数で指定してください")
		}
		if n < 0 {
			return Params{}, apperr.Validation("limit は 0 以上で指定してください")
		}
		p.Limit = clampLimit(n)
	}
	return p, nil
}

func clampLimit(n int) int {
	if n < 1 {
		return 1
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

// Encode は並び替えキーを base64(JSON) のカーソル文字列にする。
func Encode(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Decode はカーソル文字列を dst に書き戻す。
// 空文字・壊れた base64・壊れた JSON は VALIDATION_ERROR。
func Decode(cursor string, dst any) error {
	if cursor == "" {
		return apperr.Validation("cursor が不正です")
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return apperr.Validation("cursor が不正です")
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return apperr.Validation("cursor が不正です")
	}
	return nil
}

// Page は limit+1 件取得済みの items から、返却ページと next_cursor を作る。
//
//	len(items) > limit → 先頭 limit 件を返し、limit 番目の keyOf で next_cursor を生成
//	それ以外 → 全件を返し next_cursor は nil
//
// keyOf は並び替えキー（Encode に渡す値）を返す。
func Page[T any](items []T, limit int, keyOf func(T) any) (page []T, nextCursor *string, err error) {
	if limit < 1 {
		limit = defaultLimit
	}
	if len(items) <= limit {
		if items == nil {
			items = []T{}
		}
		return items, nil, nil
	}

	page = items[:limit]
	encoded, err := Encode(keyOf(page[limit-1]))
	if err != nil {
		return nil, nil, err
	}
	return page, &encoded, nil
}
