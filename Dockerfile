# ========== 第一阶段：构建 ==========
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build


# ========== 第二阶段：编译 Go ==========
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS go-builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags="-s -w" -trimpath -o golive ./cmd/server


# ========== 第三阶段：运行 ==========
FROM alpine:3.21

RUN apk add --no-cache ca-certificates dumb-init

# 创建非 root 用户
RUN addgroup -g 1000 appgroup && \
    adduser  -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 复制 Go 二进制
COPY --from=go-builder    /src/golive  ./

# 复制前端构建产物
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

# 复制空数据库占位（程序会在首次启动时自动创建）
RUN touch /app/golive.db && chown appuser:appgroup /app/golive.db

USER appuser

EXPOSE 8080

ENV GOLIVE_DB=/app/golive.db

ENTRYPOINT ["dumb-init", "--"]
CMD ["./golive"]
