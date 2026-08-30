# codetrain-api

配信 API（Go + Echo）。成果物はコンテナイメージで、ECR → EKS にデプロイする。

## 動かし方

**ホストで `go run` しない。** ローカルでも必ずコンテナで動かす（[Document/LOCAL_DEV.md](../Document/LOCAL_DEV.md) §1）。
起動は `codetrain-devenv` から:

```bash
cd ../codetrain-devenv
make up-core        # postgres / ministack
make seed           # migrate + シード投入
make up-product     # api（air でホットリロード）
make logs SVC=api
```

ホストの Go（goenv / `.go-version`）はエディタ補完と `go mod tidy` のための補助。
**実際にコンパイルする Go のバージョンは `Dockerfile.dev` / `Dockerfile` のベースイメージが決める**（§9.2）。

## エンドポイント

| メソッド | パス | 認証 | 内容 |
| --- | --- | --- | --- |
| GET | `/healthz` | 不要 | `{"status":"ok","db":"ok"}`。DB 不通なら 503 |
| GET | `/v1/skills` | 不要 | スキルツリー（シード確認用） |
| GET | `/v1/me` | **必須** | ユーザーと進捗 |

```bash
curl -sS http://localhost:8080/healthz
curl -sS http://localhost:8080/v1/skills
curl -sS -H "X-Dev-User: seed-user-01" http://localhost:8080/v1/me
```

## 認証（`AUTH_MODE`）

| モード | 挙動 |
| --- | --- |
| `cognito` | JWKS を取得して署名・iss・aud・期限を検証（本番と同じコードパス） |
| `dev`（既定） | 署名検証をせず `X-Dev-User` ヘッダを sub として扱う |

**`dev` の実装はビルドタグ `dev_auth` で分離している。**

- `internal/middleware/auth_dev.go` — `//go:build dev_auth`
- `internal/middleware/auth_dev_disabled.go` — `//go:build !dev_auth`（起動時にエラー）

開発イメージは `.air.toml` が `-tags dev_auth` を付けてビルドし、**本番用 `Dockerfile` は付けない**。
環境変数の設定ミスで本番の認証が無効化される経路を、コンパイル時に消すのが目的。

確認:

```bash
# 本番と同じ条件でビルドすると X-Dev-User の文字列が入らない
go build -o /tmp/api-prod ./cmd/api && strings /tmp/api-prod | grep -c X-Dev-User   # => 0
```

## `codetrain-core` への依存

`go.mod` に `replace github.com/gabaison-2026-09/codetrain-core => ../codetrain-core` が入っている。
core が GitHub に push されていないための暫定措置で、compose は core も同じ相対位置にマウントしている。

**core を公開したら `replace` を外し、バージョン固定の参照に切り替えること**（LOCAL_DEV.md §10.1）。
`replace` を残したままだと本番用 `Dockerfile` のビルドも CI も通らない。
