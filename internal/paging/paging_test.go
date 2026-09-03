package paging

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/gabaison-2026-09/codetrain-api/internal/apperr"
)

// cursorKey は Encode/Decode 往復の検証用。実際の一覧でも同形を使う想定。
type cursorKey struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantLimit int
		wantCur   string
		wantErr   bool
	}{
		{name: "未指定なら default 20", query: "", wantLimit: 20},
		{name: "limit=50", query: "limit=50", wantLimit: 50},
		{name: "limit=1000 は 100 に clamp", query: "limit=1000", wantLimit: 100},
		{name: "limit=0 は 1 に clamp", query: "limit=0", wantLimit: 1},
		{name: "cursor を保持", query: "cursor=abc&limit=10", wantLimit: 10, wantCur: "abc"},
		{name: "limit=-5 は VALIDATION_ERROR", query: "limit=-5", wantErr: true},
		{name: "limit=abc は VALIDATION_ERROR", query: "limit=abc", wantErr: true},
		{name: "limit=1.5 は VALIDATION_ERROR", query: "limit=1.5", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			got, err := ParseParams(c)
			if tt.wantErr {
				if err == nil {
					t.Fatal("err = nil, want VALIDATION_ERROR")
				}
				var apiErr *apperr.APIError
				if !errors.As(err, &apiErr) || apiErr.Code != apperr.CodeValidationError {
					t.Fatalf("err = %v, want VALIDATION_ERROR", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseParams: %v", err)
			}
			if got.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Cursor != tt.wantCur {
				t.Errorf("Cursor = %q, want %q", got.Cursor, tt.wantCur)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	want := cursorKey{
		CreatedAt: time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
		ID:        "q-uuid-01",
	}

	cur, err := Encode(want)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if cur == "" {
		t.Fatal("Encode が空文字を返した")
	}

	var got cursorKey
	if err := Decode(cur, &got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Errorf("Decode = %+v, want %+v", got, want)
	}
}

func TestDecodeInvalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{name: "空文字", cursor: ""},
		{name: "壊れた base64", cursor: "!!!not-base64!!!"},
		{name: "JSON ではない", cursor: base64.RawURLEncoding.EncodeToString([]byte("not-json"))},
		{name: "改竄（型不一致）", cursor: base64.RawURLEncoding.EncodeToString([]byte(`"string-not-object"`))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst cursorKey
			err := Decode(tt.cursor, &dst)
			if err == nil {
				t.Fatal("err = nil, want VALIDATION_ERROR")
			}
			var apiErr *apperr.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != apperr.CodeValidationError {
				t.Fatalf("err = %v, want VALIDATION_ERROR", err)
			}
		})
	}
}

func TestPage(t *testing.T) {
	type item struct {
		ID string
	}
	keyOf := func(it item) any { return cursorKey{ID: it.ID} }

	t.Run("次頁なし（件数 = limit）", func(t *testing.T) {
		items := []item{{"a"}, {"b"}}
		page, next, err := Page(items, 2, keyOf)
		if err != nil {
			t.Fatalf("Page: %v", err)
		}
		if len(page) != 2 {
			t.Errorf("page = %d 件, want 2", len(page))
		}
		if next != nil {
			t.Errorf("next_cursor = %v, want nil", *next)
		}
	})

	t.Run("次頁なし（件数 < limit）", func(t *testing.T) {
		items := []item{{"a"}}
		page, next, err := Page(items, 2, keyOf)
		if err != nil {
			t.Fatalf("Page: %v", err)
		}
		if len(page) != 1 {
			t.Errorf("page = %d 件, want 1", len(page))
		}
		if next != nil {
			t.Errorf("next_cursor = %v, want nil", *next)
		}
	})

	t.Run("次頁あり（limit+1 件）", func(t *testing.T) {
		items := []item{{"a"}, {"b"}, {"c"}}
		page, next, err := Page(items, 2, keyOf)
		if err != nil {
			t.Fatalf("Page: %v", err)
		}
		if len(page) != 2 || page[0].ID != "a" || page[1].ID != "b" {
			t.Errorf("page = %+v, want [{a} {b}]", page)
		}
		if next == nil {
			t.Fatal("next_cursor = nil, want 非 nil")
		}
		var key cursorKey
		if err := Decode(*next, &key); err != nil {
			t.Fatalf("Decode(next): %v", err)
		}
		if key.ID != "b" {
			t.Errorf("next key.ID = %q, want %q", key.ID, "b")
		}
	})

	t.Run("nil 入力は空スライスを返す", func(t *testing.T) {
		page, next, err := Page[item](nil, 20, keyOf)
		if err != nil {
			t.Fatalf("Page: %v", err)
		}
		if page == nil {
			t.Error("page = nil, want 空スライス")
		}
		if len(page) != 0 {
			t.Errorf("page = %d 件, want 0", len(page))
		}
		if next != nil {
			t.Errorf("next_cursor = %v, want nil", *next)
		}
	})
}
