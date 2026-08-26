# AXmiPusher 单机部署镜像(产物拷贝式)。
# 构建产物在宿主机生成(Go 交叉编译 + 前端 npm build), 镜像只负责封装运行环境。
# 理由: 服务器仅 1G 内存, 在容器内跑 go/node 构建会 OOM。
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
WORKDIR /app
COPY api /usr/local/bin/api
COPY web/user /app/web/user
COPY web/admin /app/web/admin
RUN chmod +x /usr/local/bin/api
EXPOSE 8080
CMD ["api"]
