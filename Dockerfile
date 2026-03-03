# ============ 构建阶段 ============
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.Version=1.0.0 -X main.BuildTime=$(date +%Y-%m-%dT%H:%M:%S)" \
    -o tinLink ./cmd/server

# ============ 运行阶段 ============
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

COPY --from=builder /app/tinLink .
COPY --from=builder /app/configs ./configs

ENV TZ=Asia/Shanghai

RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./tinLink"]