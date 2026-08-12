---
name: getnote-auth
description: 安装和连接得到大脑，完成浏览器授权、环境诊断、配额检查、CLI 升级与领域 Skill 同步。用户说“安装/连接/登录/更新得到大脑”“检查为什么不能用”“查看额度”时使用。
---

# 得到大脑连接、诊断与升级

负责把“用户想用得到大脑”推进到真正可执行的状态。不要只说“已安装”：CLI 可执行、账号已授权、API 可读三项都通过才算连接成功。

## 首次安装闭环

按顺序执行，已经满足的步骤直接跳过：

1. 用 `command -v getnote` 检查官方 CLI。不要因为 Skill 已安装就假设 CLI 存在。
2. CLI 缺失时检查 `node --version` 和 `npm --version`；需要 Node.js 20 或以上。
3. 自动执行 `npm install -g @getnote/cli@latest`。这是 Agent 的工作，不要求用户手工复制命令；只有系统弹出安装授权时才请用户确认。
4. 执行 `getnote version`，必须能够正常启动。
5. 执行 `getnote auth status`。未登录时运行 `getnote auth login`，让用户只在浏览器中确认，不索要 API Key、Cookie 或 Authorization。
6. 执行 `getnote doctor -o json`。只有 `success=true`，且 `checks` 中 `cli`、`auth`、`api` 均为 `ok=true`，才能宣布连接完成。
7. 在 Codex、Claude Code 或 Cursor 等本地 Agent 中，执行 `getnote setup` 同步 5 个领域 Skill；如果当前平台已经由独立 Skill 包携带这些领域 Skill，或 CLI 明确提示未检测到受支持平台，不把这一步失败误报成账号连接失败。
8. 先用 `getnote notes --limit 1 -o json` 做无写入验收。只有用户同意创建测试内容时，才保存测试笔记。

## 日常路由

| 意图 | 命令 |
|---|---|
| 登录 | `getnote auth login` |
| 查看登录状态 | `getnote auth status` |
| 退出登录 | `getnote auth logout` |
| 诊断连接 | `getnote doctor -o json` |
| 查看 CLI 能力契约 | `getnote capabilities -o json` |
| 为本机 AI 同步领域 Skill | `getnote setup` |
| 查看 AI 对话额度 | `getnote quota -o json` |
| 查看版本 | `getnote version` |
| 检查升级 | `getnote update --check` |
| 执行升级 | `getnote update` |

参数不确定时读取对应命令的 `--help`，不要凭旧文档猜参数。

## 结果契约

- `doctor -o json`：读取 `success`、`cli_version`、`checks[]`、`platforms[]`。`success=false` 或关键检查失败都表示尚未可用。
- `capabilities -o json`：读取 `contract_version`、`commands`、`command_aliases`、`guarantees`；只在首次安装、升级后或兼容性排查时使用，不要每次任务都调用。
- `quota -o json`：按 CLI 返回的真实额度窗口说明已用和剩余次数，不自行换算。
- 所有命令以退出码为第一判断：退出码非 0 即失败。使用 `-o json` 时，API 与本地错误都返回 `success=false`、`data=null`、`error.code/message/reason/retryable` 和可选 `request_id`；不能伪装成功。

## 更新闭环

用户说“帮我更新得到大脑”已经构成更新授权：

1. 执行 `getnote update --check`。
2. 有新版本时执行 `getnote update`；若 CLI 明确提示 npm 安装方式，则用 `npm install -g @getnote/cli@latest`。
3. 执行 `getnote version` 和 `getnote doctor -o json`。
4. 执行 `getnote setup` 同步内置领域 Skill；技能市场托管的独立 Skill 需要平台更新时，只让用户完成唯一必要的点击。
5. 用最近笔记读取做验收，再告诉用户版本、诊断结果和仍需动作。

## 安全与恢复

- 不展示或记录完整凭证；`auth status` 只能出现掩码。
- 用户未明确要求时不退出登录。
- 授权超时、拒绝或验证码过期时重新启动一次登录流程，不复用旧 code。
- 失败时保留执行步骤、错误原因和 `request_id`；不要只回复“连接失败”。
