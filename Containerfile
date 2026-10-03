# 多阶段构建：构建阶段一次性编译 cmd/ 下全部服务；运行阶段单镜像承载，
# 由 compose 用不同 command 启动 gateway / identity / catalog / order
# 若服务器拉不动官方镜像，把 FROM 换成 docker.1ms.run/library/ 前缀的同名镜像
FROM golang:1.23-alpine AS builder

# 国内访问不了 proxy.golang.org，走 goproxy.cn；在海外构建可删掉这行
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src
COPY go.mod go.sum ./
# 依赖单独成层：go.mod/go.sum 不变时命中构建缓存，不用重新下载
RUN go mod download
COPY . .
RUN for svc in gateway identity catalog order; do \
      CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/$svc ./cmd/$svc; \
    done

FROM alpine:3.20
# tzdata：DSN 里 loc=Local 依赖本机时区，不装的话时间字段会差 8 小时
RUN apk add --no-cache tzdata ca-certificates
ENV TZ=Asia/Shanghai
# 关闭 gin 的 debug 日志
ENV GIN_MODE=release

WORKDIR /app
COPY --from=builder /out/ .
# 故意不写 ENTRYPOINT/CMD：启动哪个服务由 compose 的 command 指定
