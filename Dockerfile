# 本番用イメージ（ECR → EKS）。
#
# **dev_auth タグを付けない。**
# これにより AUTH_MODE=dev の実装（internal/middleware/auth_dev.go）は
# バイナリにコンパイルされず、設定ミスで認証が無効化される経路が存在しなくなる
# （LOCAL_DEV.md §5.4 / §13-10、OPEN_ISSUES D-22）。

FROM golang:1.27 AS build

WORKDIR /src

# core を private module として取得するには git 認証が必要になる。
# 認証情報をレイヤに焼かないよう、BuildKit の secret mount で渡すこと（OPEN_ISSUES D-24）。
#   docker build --secret id=netrc,src=$HOME/.netrc .
COPY go.mod go.sum ./
RUN --mount=type=secret,id=netrc,target=/root/.netrc,required=false \
    go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
