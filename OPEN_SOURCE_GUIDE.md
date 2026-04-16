# 第一次开源：GitHub 操作指南（Big Bat）

这份指南是给你这种“第一次开源”的场景准备的，按步骤做就可以把项目安全、规范地发布到 GitHub。

## 1) 开源前先检查（非常重要）

在发布前，先确认没有敏感信息进入仓库：

- `.env` 不要提交
- 任何真实 `GS_COOKIE` 不要提交
- 任何真实 `API_SECRET` 不要提交
- `data/runtime-state.json`（如果含运行态数据）不要提交

本项目已默认忽略：`.env`、`data/`、`dist/`。

## 2) 选择开源许可证（License）

建议二选一：

- `MIT`：最宽松，别人可商用、修改、闭源再发布
- `Apache-2.0`：同样宽松，但带专利授权条款，企业更常用

如果你只是想快速开源，选 `MIT` 最简单。

## 3) 本地初始化 Git（如果还没 init）

```bash
cd /Users/tensho/Code/bigbat
git init
git add .
git commit -m "chore: initial open-source release of Big Bat"
```

## 4) 在 GitHub 创建仓库

去 GitHub 网页：

1. 点右上角 `+` -> `New repository`
2. Repository name 建议：`big-bat`
3. Description 可填：
   - `OpenAI + Anthropic compatible proxy for Genspark web models`
4. 选择 `Public`
5. **不要勾选** `Add a README / .gitignore / license`（我们本地已有）
6. 点 `Create repository`

## 5) 绑定远程并首次推送

把下面 URL 替换成你的用户名：

```bash
cd /Users/tensho/Code/bigbat
git branch -M main
git remote add origin https://github.com/<你的用户名>/big-bat.git
git push -u origin main
```

## 6) 推荐仓库设置（开源友好）

在仓库网页里设置：

- `Settings -> General`
  - 开启 Issues
  - 开启 Discussions（可选）
- `Settings -> Features`
  - 勾选 `Wikis`（可选）
- `Settings -> Branches`
  - 给 `main` 加保护规则（至少要求 PR 合并）

## 7) 建议补充的开源文件

可选但推荐：

- `LICENSE`（MIT/Apache-2.0）
- `CONTRIBUTING.md`（贡献流程）
- `CODE_OF_CONDUCT.md`（社区行为准则）
- `SECURITY.md`（漏洞提交流程）

## 8) 发布第一个 Release（可选）

```bash
git tag -a v0.1.0 -m "Big Bat first public release"
git push origin v0.1.0
```

然后到 GitHub -> `Releases` -> `Draft a new release`，选择 `v0.1.0`。

## 9) 给用户的最短部署命令（你可以贴在 README 顶部）

```bash
cp .env.example .env
./scripts/deploy-service.sh
```

## 10) Claude Code 对接命令（你可放 README）

```bash
BIGBAT_BASE_URL="http://<你的服务器IP>:7055" \
BIGBAT_API_KEY="<你的API_SECRET>" \
BIGBAT_MODEL="claude-opus-4-6" \
./scripts/start-claudecode.sh
```

---

如果你愿意，我还可以继续帮你补三件最实用的开源文件：

1. `LICENSE`（MIT）
2. `CONTRIBUTING.md`
3. `SECURITY.md`

这样你的仓库会看起来更专业，也更容易让别人参与。
