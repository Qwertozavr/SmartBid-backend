# syntax=docker/dockerfile:1

FROM golang:1.26-alpine3.23 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/smartbid \
    ./cmd/smartbid
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOBIN=/out \
    go install \
    -tags="no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb" \
    github.com/pressly/goose/v3/cmd/goose@v3.24.3

FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S smartbid \
    && adduser -S -D -H -G smartbid smartbid

WORKDIR /app

COPY --from=builder --chown=smartbid:smartbid --chmod=0555 /out/smartbid /bin/smartbid
COPY --from=builder --chown=smartbid:smartbid --chmod=0555 /out/goose /bin/goose
COPY --chown=smartbid:smartbid migrations ./migrations
COPY --chown=smartbid:smartbid --chmod=0555 docker/entrypoint.sh /entrypoint.sh

USER smartbid:smartbid

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
