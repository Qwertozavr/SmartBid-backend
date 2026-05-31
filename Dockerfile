FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /bin/smartbid ./cmd/smartbid
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /bin/smartbid /bin/smartbid
COPY --from=builder /go/bin/goose /bin/goose
COPY migrations ./migrations
COPY docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
