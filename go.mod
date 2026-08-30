module github.com/gabaison-2026-09/codetrain-api

go 1.27

require (
	github.com/gabaison-2026-09/codetrain-core v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/labstack/echo/v4 v4.13.4
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
)

// codetrain-core はまだ GitHub に push されていないため、隣のチェックアウトを指す。
// LOCAL_DEV.md §10.1 のとおり、core にタグを打って公開した時点でこの replace を外し、
// GOPRIVATE 経由のバージョン固定参照に切り替える。
replace github.com/gabaison-2026-09/codetrain-core => ../codetrain-core
