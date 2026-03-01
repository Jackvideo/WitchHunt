# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOMAXPROCS=2 CGO_ENABLED=0 GOOS=linux \
    go build -p 2 -ldflags="-s -w" -o witchhunt ./cmd/main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /build/witchhunt .
EXPOSE 8080
CMD ["./witchhunt"]
