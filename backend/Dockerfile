# syntax=docker/dockerfile:1
FROM golang:1.23-bookworm AS build
# Go 模块代理，便于在受限网络（如无法直连 proxy.golang.org）环境下构建。
# 可通过 docker build --build-arg GOPROXY=https://proxy.golang.org,direct 覆盖。
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/server /app/server
ENV PORT=8080 \
    DATA_DIR=/data \
    TZ=Asia/Shanghai
EXPOSE 8080
CMD ["/app/server"]
