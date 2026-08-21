# 阶段1：编译go代码（构建器）
FROM golang:1.26.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO禁用，静态编译，适配alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

# 阶段2：运行镜像（只放二进制，无go编译环境）
FROM alpine:3.20
WORKDIR /app
# 时区可选，解决日志时间问题
RUN apk add --no-cache tzdata
COPY --from=builder /app/server .
# 配置文件如果有，一并复制
# COPY config.yaml .
EXPOSE 8090
CMD ["./server"]
