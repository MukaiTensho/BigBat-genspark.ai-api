# Big Bat 部署与对接说明

本文档面向“直接部署可用”的场景，包含：

- 一键部署（注册服务，重启自动运行）
- 一键卸载
- OpenAI / Anthropic（Claude Code）对接
- 常见问题排查

## 1) 准备

- 平台：macOS 或 Linux
- 依赖：Docker + Docker Compose

## 2) 快速部署（注册服务）

```bash
cp .env.example .env
# 编辑 .env，至少填 GS_COOKIE、API_SECRET
./scripts/deploy-service.sh
```

部署完成后默认端口：`7055`

管理后台：

`http://127.0.0.1:7055/admin/ui`

## 3) 卸载

```bash
./scripts/remove-service.sh
```

连 `.env` 一并删除：

```bash
./scripts/remove-service.sh --purge-env
```

## 4) 验证

```bash
curl -s http://127.0.0.1:7055/ | jq
curl -s http://127.0.0.1:7055/v1/models -H "Authorization: Bearer 123456" | jq
curl -s http://127.0.0.1:7055/v1/messages/health -H "x-api-key: 123456" | jq
```

## 5) OpenAI 客户端对接

- Base URL: `http://<服务器IP>:7055/v1`
- API Key: `.env` 中的 `API_SECRET`
- Model: `opus4.6` 或 `claude-opus-4-6`

## 6) Claude Code 对接

推荐用脚本：

```bash
BIGBAT_BASE_URL="http://<服务器IP>:7055" \
BIGBAT_API_KEY="123456" \
BIGBAT_MODEL="claude-opus-4-6" \
./scripts/start-claudecode.sh
```

脚本会自动处理环境变量冲突（清理 `ANTHROPIC_AUTH_TOKEN`）。

## 7) 常见问题

- `no healthy cookies for anthropic endpoint`
  - 说明 Anthropic 探针判断当前 cookie 不可用于真实聊天请求
  - 用后台 `Cookies 健康检查 + 调试视图`确认原因

- `invalid_authorization`
  - API Key 与 `API_SECRET` 不一致

- `docker hub EOF/context deadline exceeded`
  - 网络导致重建失败，先用：
    - `./scripts/start.sh`
  - 更新代码后离线热重建：
    - `./scripts/rebuild-offline.sh`
