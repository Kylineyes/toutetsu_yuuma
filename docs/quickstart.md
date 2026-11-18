# 快速开始与本地开发

## 快速开始（Docker）

```bash
export PASSWORD=changeme      # Web UI 登录口令
export API_TOKEN=             # 可选：第三方调用令牌
docker compose up -d --build
# 前端 http://localhost:8088   后端 API http://localhost:8080/api/v1
```

## 本地开发

```bash
cd backend && GOPROXY=https://goproxy.cn,direct go run ./cmd/server
cd frontend && npm install && npm run dev      # http://localhost:5173（代理 /api → :8080）
```

> 返回上级：[README](../README.md)
