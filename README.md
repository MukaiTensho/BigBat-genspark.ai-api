# bigbat

`bigbat` 是一个行为复刻、代码重写的 Genspark 反向代理服务，对外提供 OpenAI 兼容 API，便于接入 OpenClaw、NextChat、one-api 等客户端。

## 1. 核心能力

- OpenAI 兼容接口：
  - `POST /v1/chat/completions`（流式/非流式）
  - `POST /v1/images/generations`
  - `POST /v1/videos/generations`
  - `GET /v1/models`
- Anthropic 兼容接口：
  - `POST /v1/messages`（支持 Claude Code 场景）
  - `GET /v1/messages/health`（Anthropic 兼容可用性探针）
- OpenAI 探针接口：
  - `GET /v1/chat/health`（OpenAI chat 可用性探针）
- Cookie 池管理：轮换、失效剔除、限流封禁
- 模型会话绑定：支持自动映射与固定映射
- 可选 ReCaptcha 代理注入（`RECAPTCHA_PROXY_URL`）
- 管理后台：可视化管理 cookies、运行参数、模型调用方式、健康检查
- 运行时状态持久化：重启后保留后台改动（`ADMIN_STATE_FILE`）

## 2. 目录结构

- `cmd/bigbat/main.go`：程序入口
- `internal/server/`：HTTP 路由、OpenAI 兼容接口、管理后台
- `internal/genspark/`：上游请求、SSE/任务状态处理
- `internal/state/`：Cookie 池、会话管理、限流
- `internal/config/`：环境变量与模型配置
- `scripts/`：安装、卸载、服务注册、打包脚本
- `Dockerfile` / `docker-compose.yml`：容器化部署

## 3. 环境要求

- Docker + Docker Compose（推荐部署方式）
- 若使用 Linux 服务注册：需要 `systemd`
- 若使用 macOS 服务注册：使用 `launchd`（当前用户登录后自启）

## 4. 快速启动（开发模式）

```bash
cp .env.example .env
go run ./cmd/bigbat
```

默认监听 `:7055`。

## 5. Docker 部署（推荐）

```bash
cp .env.example .env
docker compose up -d --build
docker compose logs -f bigbat
```

停止服务：

```bash
docker compose down
```

## 6. 一键部署 / 一键卸载（含服务注册）

### 一键部署并注册服务

```bash
./scripts/deploy-service.sh
```

- macOS：注册 `launchd` 服务
- Linux：注册 `systemd` 服务
- 容器本身同时具备 `restart: unless-stopped` 重启策略
- 服务启动命令使用 `scripts/start.sh`（不依赖 Docker Hub 即时拉取）

### 一键卸载（含服务注销）

```bash
./scripts/remove-service.sh
```

连 `.env` 一并删除：

```bash
./scripts/remove-service.sh --purge-env
```

### 仅安装/卸载容器（不处理系统服务）

```bash
./scripts/install.sh
./scripts/uninstall.sh
```

### 常用运行脚本

- 启动（优先用本地已有镜像，不走重建）：

```bash
./scripts/start.sh
```

- 强制重建并启动（需要可访问 Docker Hub）：

```bash
./scripts/rebuild.sh
```

- 离线重建（不访问 Docker Hub，适合你当前 EOF 场景）：

```bash
./scripts/rebuild-offline.sh
```

- 启动 Claude Code（自动注入 Big Bat 兼容环境变量）：

```bash
BIGBAT_BASE_URL="http://100.92.199.24:7055" BIGBAT_API_KEY="123456" ./scripts/start-claudecode.sh
```

## 7. 服务管理脚本（手动）

### macOS (launchd)

```bash
./scripts/service/install-launchd.sh
./scripts/service/uninstall-launchd.sh
```

### Linux (systemd)

```bash
./scripts/service/install-systemd.sh
./scripts/service/uninstall-systemd.sh
```

## 8. 管理后台

- 地址：`http://127.0.0.1:7055/admin/ui`
- 使用方式：先输入 API Key（即 `.env` 里的 `API_SECRET`），再执行后台操作

功能包括：

- Cookies 管理（加载、保存）
- Cookie 健康检查（healthy / expired / limited / blocked / error）
- 运行参数修改（限速、API keys 等）
- 模型列表与调用方式（可复制 cURL 和 body）
- 接入参数面板（Base URL / Token / 模型名一键复制）

后台 API：

- `GET /admin/state`
- `GET|POST|DELETE /admin/cookies`
- `GET /admin/cookies/health`
- `GET /admin/cookies/health?debug=1`（返回判定调试信息）
- `PATCH /admin/config`
- `GET /admin/models`

## 9. 客户端接入参数（最常用）

- Base URL：`http://127.0.0.1:7055/v1`
- API Key：`API_SECRET` 的值（例如 `123456`）
- 模型：`opus4.6` 或 `claude-opus-4-6`

Anthropic/Claude Code 兼容参数：

- Base URL：`http://127.0.0.1:7055`
- Endpoint：`/v1/messages`
- API Key：支持 `x-api-key: <API_SECRET>` 或 `Authorization: Bearer <API_SECRET>`
- 模型：`opus4.6` 或 `claude-opus-4-6`

示例：

```bash
curl -s "http://127.0.0.1:7055/v1/chat/completions" \
  -H "Authorization: Bearer 123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "opus4.6",
    "stream": false,
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

Anthropic 协议示例：

```bash
curl -s "http://127.0.0.1:7055/v1/messages" \
  -H "x-api-key: 123456" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "opus4.6",
    "max_tokens": 1024,
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

Anthropic 可用性探针：

```bash
curl -s "http://127.0.0.1:7055/v1/messages/health" \
  -H "x-api-key: 123456" | jq
```

OpenAI 可用性探针：

```bash
curl -s "http://127.0.0.1:7055/v1/chat/health" \
  -H "Authorization: Bearer 123456" | jq
```

## 10. 环境变量说明

关键变量：

- `HOST`：监听地址，默认 `0.0.0.0`（局域网可访问）
- `GS_COOKIE`：必填
  - 支持单个 cookie
  - 多个 cookie 推荐用换行或 `||` 分隔（不要用逗号）
- `API_SECRET`：可选，开启后所有接口需 `Authorization: Bearer <key>`
- `PORT`：默认 `7055`
- `ROUTE_PREFIX`：可选，如 `hf`，路由变为 `/hf/v1/...`
- `PROXY_URL`：可选，上游代理
- `RECAPTCHA_PROXY_URL`：可选，需提供 `GET /genspark` 返回 `{"code":200,"token":"..."}`
- `REQUEST_RATE_LIMIT`：每 IP 每分钟限流，默认 `60`
- `AUTO_DEL_CHAT`：默认 `0`，设 `1` 请求后自动清理会话
- `AUTO_MODEL_CHAT_MAP_TYPE`：默认 `1`，自动模型会话映射
- `MODEL_CHAT_MAP`：可选，手动 `model=id` 绑定
- `SESSION_IMAGE_CHAT_MAP`：可选，图片/视频会话绑定
- `ADMIN_STATE_FILE`：运行时状态文件（默认 `./data/runtime-state.json`）

模型别名：

- `opus4.6` -> `claude-opus-4-6`
- `opus-4.6` -> `claude-opus-4-6`
- `claude-opus-4.6` -> `claude-opus-4-6`

## 11. 一键打包（源代码 + README + 脚本）

生成可迁移压缩包：

```bash
./scripts/package.sh
```

输出目录默认是 `dist/`，包含：

- 完整源代码
- 完整 README
- 部署/卸载/服务注册脚本
- Docker 相关文件

## 12. 迁移到新机器

在旧机器：

```bash
./scripts/package.sh
```

把 `dist/*.tar.gz` 复制到新机器后：

```bash
tar -xzf bigbat-*.tar.gz
cd bigbat
cp .env.example .env
# 编辑 .env 填入你的 GS_COOKIE / API_SECRET
./scripts/deploy-service.sh
```

## 13. 故障排查

- 局域网无法访问：检查 `HOST=0.0.0.0`、防火墙、以及 Docker 端口映射 `7055:7055`
- 执行 `docker compose up -d --build` 失败且提示 `EOF` / `context deadline exceeded`：
  - 这是 Docker Hub 网络问题，不是程序逻辑问题
  - 用 `./scripts/start.sh` 从本地已有镜像启动
  - 等网络恢复再执行 `./scripts/rebuild.sh`
- 后台提示 `invalid_authorization`：API Key 与 `API_SECRET` 不一致
- Docker 出现 `$xxx variable is not set`：`.env` 中 cookie 含 `$`，重新执行 `./scripts/install.sh`
- 聊天返回上游 retired 提示：确认版本已使用 `/api/agent/ask_proxy` 新链路
- 模型不可用：先看 `/admin/cookies/health`，失效/限流会直接影响可用性

## 14. 安全建议

- 不要把 `.env`、完整 cookies、API key 上传到公开仓库
- 建议定期轮换 cookies 与 API_SECRET
- 对外网部署时建议加反向代理 + TLS + IP 白名单

## 15. 兼容与免责声明

- 本项目为行为兼容实现，不直接复制原项目代码
- 上游网页接口可能变更，必要时需调整适配逻辑
- 仅供学习与技术研究，请遵守目标平台条款及当地法律法规

## 16. Claude Code 对接（完整步骤）

本项目已支持 Anthropic 兼容接口：

- `POST /v1/messages`
- `GET /v1/messages/health`

### 16.1 对接前检查

先确认服务可用：

```bash
curl -s "http://127.0.0.1:7055/v1/messages/health" \
  -H "x-api-key: 123456" | jq
```

当返回 `ready: true` 时再接入 Claude Code。

### 16.2 推荐启动方式（脚本）

```bash
BIGBAT_BASE_URL="http://100.92.199.24:7055" \
BIGBAT_API_KEY="123456" \
BIGBAT_MODEL="claude-opus-4-6" \
/Users/tensho/Code/bigbat/scripts/start-claudecode.sh
```

该脚本会：

- 自动清理 `ANTHROPIC_AUTH_TOKEN`（避免和 API Key 冲突）
- 设置 `ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`
- 启动 `claude` 或 `claude-code`

### 16.3 手动方式（不使用脚本）

```bash
unset ANTHROPIC_AUTH_TOKEN
export ANTHROPIC_BASE_URL="http://100.92.199.24:7055"
export ANTHROPIC_API_KEY="123456"
export ANTHROPIC_MODEL="claude-opus-4-6"
```

### 16.4 常见错误

- 报错 `Auth conflict: Both a token and an API key are set`
  - 说明同时设置了 `ANTHROPIC_AUTH_TOKEN` 和 `ANTHROPIC_API_KEY`
  - 解决：`unset ANTHROPIC_AUTH_TOKEN`，或在 Claude Code 内 `/logout` 后仅使用 API key

- 报错 `invalid_authorization`
  - 说明 `x-api-key` 与 Big Bat 的 `API_SECRET` 不一致

- 报错 `no healthy cookies for anthropic endpoint`
  - 说明当前 cookie 对聊天接口不可用（可能失效/限流/风控）
  - 请先在后台看：
    - `GET /admin/cookies/health?debug=1`
    - `GET /v1/messages/health`

## 17. 最终部署流程（建议照抄）

以下流程可用于新机器上线：

```bash
# 1) 解压发布包
tar -xzf bigbat-*.tar.gz
cd bigbat-*

# 2) 复制环境变量模板并填写
cp .env.example .env

# 3) 一键部署并注册服务（开机自动）
./scripts/deploy-service.sh

# 4) 验证健康状态
curl -s http://127.0.0.1:7055/ | jq
curl -s http://127.0.0.1:7055/v1/messages/health -H "x-api-key: 123456" | jq
```

卸载：

```bash
./scripts/remove-service.sh
```

彻底卸载（含 `.env`）：

```bash
./scripts/remove-service.sh --purge-env
```
