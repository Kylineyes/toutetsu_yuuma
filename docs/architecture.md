# 架构与数据布局

## 目录结构

```
toutetsu_yuuma/
├── docker-compose.yml
├── backend/                 Go 服务（内置 ffmpeg）
│   ├── Dockerfile
│   ├── cmd/server/main.go
│   └── internal/{config,db,api,worker,deepseek,media,subtitle,pipeline}/
└── frontend/                Vue3 前端（nginx 托管）
    ├── Dockerfile, nginx.conf
    └── src/{views,api.js,main.js}
```

## 数据布局

```
/data/db/app.db
/data/knowledge-bases/blobs/{sha前2}/{sha}      # 内容寻址去重
/data/knowledge-bases/{kbId}/v{ver}/manifest.json
/tmp/intermediates/{taskId}/                     # 音频/分片/截图帧（TMP_DIR 下）
/data/task/{taskId}/upload|subs|output/
```

> 返回上级：[README](../README.md)
