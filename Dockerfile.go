# 编译阶段
FROM golang:1.26-alpine AS builder
WORKDIR /app

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct && \
    go mod download

# 复制源代码（包括 pb 目录和 gateway.go）
COPY . .

# 编译成二进制文件
RUN go build -o gateway gateway.go

# 运行阶段：使用极小的镜像
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway .

EXPOSE 8081

CMD ["./gateway"]