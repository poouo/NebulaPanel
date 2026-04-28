## NebulaPanel - Multi-stage Docker Build
## Single container deployment

# ── Stage 1: Build Go binary ──
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /nebula-panel ./cmd/

# ── Stage 2: Runtime ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /nebula-panel /app/nebula-panel
COPY web/ /app/web/

RUN mkdir -p /data

ENV DB_PATH=/data/nebula.db
ENV LISTEN=:3001
ENV ADMIN_USER=admin
ENV ADMIN_PASS=admin123

EXPOSE 3001

VOLUME ["/data"]

CMD ["/app/nebula-panel"]
