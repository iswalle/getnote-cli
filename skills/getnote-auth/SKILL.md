---
name: getnote-auth
description: 安装和连接得到大脑，完成浏览器授权、环境诊断、配额检查、CLI 升级与领域 Skill 同步。用户说“安装/连接/登录/更新得到大脑”“检查为什么不能用”“查看额度”时使用。
---

# 得到大脑连接、诊断与升级

负责把“用户想用得到大脑”推进到真正可执行的状态。不要只说“已安装”：CLI 可执行、账号已授权、API 可读三项都通过才算连接成功。

## 首次安装闭环

按顺序执行，已经满足的步骤直接跳过：

1. 用 `command -v getnote` 检查 CLI；不要因为 Skill 已安装就假设 CLI 存在。独立的 ClawHub/OpenClaw Skill 由平台管理 CLI 依赖，不查找或运行 `scripts/install.sh`。
2. CLI 缺失时检查 `node --version` 和 `npm --version`，再自动执行 `npm install -g @getnote/cli@latest`。这是 Agent 的工作，不要求用户手工复制命令；只有系统弹出安装授权时才请用户确认。
3. 执行 `getnote version`，必须能够正常启动。
4. 执行 `getnote auth status`。未登录时运行 `getnote auth login`，让用户只在浏览器中确认，不索要 API Key、Cookie 或 Authorization。
5. 执行 `getnote doctor -o json`。只有 `diagnostics_completed=true`、`ready=true` 且 `status=ready`，才能宣布完整连接；`ready=true,status=degraded` 表示核心能力可用但仍应处理警告。`success` 和旧 `checks` 字段仅用于兼容。若未就绪，先处理 `issues[].blocking=true`，再按 `next_actions[]` 修复。需要确认的动作不得静默执行。
6. 在 Codex、Claude Code 或 Cursor 等本地 Agent 中，执行 `getnote setup` 同步 5 个领域 Skill；如果当前平台已经由独立 Skill 包携带这些领域 Skill，或 CLI 明确提示未检测到受支持平台，不把这一步失败误报成账号连接失败。
7. 先用 `getnote notes --limit 1 -o json` 做无写入验收。只有用户同意创建测试内容时，才保存测试笔记。

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

## 每条命令的结果与下一步

| 命令 | 成功后读取/确认 | 成功后怎么做 |
|---|---|---|
| `getnote auth login` | 浏览器已确认，凭证已写入本机 | 再运行 `doctor -o json`；不在聊天中展示凭证。 |
| `getnote auth status` | `Authenticated` / `Not authenticated` 或环境变量登录状态 | 未登录才启动 `auth login`；状态里只能出现掩码。 |
| `getnote auth logout` | `Logged out successfully.` | 只说明本机已退出；不声称已撤销服务端授权。 |
| `getnote doctor -o json` | `diagnostics_completed`、`local_ready`、`ready/status/summary`、`checks[]`、`issues[]`、`next_actions[]`、`update`、`integrations[]` | `ready=true,status=ready` 才表示完整可用；`degraded` 需继续处理警告。`platforms[]` 仅表示检测到应用或命令，不能证明 Skill 已安装；用 `integrations[].skill_status/ready` 判断 AI 接入状态。 |
| `getnote capabilities -o json` | `contract_version`、`commands`、`command_aliases`、`command_results`、`guarantees` | 只在安装、升级或兼容排查时读取；这是命令和结果字段的唯一事实源。 |
| `getnote setup -o json` | `success`、`targets[]`、`installed_cli`、`installed_skills`、`authenticated`、`platforms[]`、`next_actions[]`、`next` | `platforms[].status=installed` 表示本地安装完成；`verify_in_platform` 表示 OpenClaw（小龙虾）或 QClaw 由平台管理，只引导用户完成对应的唯一 `next_action`。没有识别到 AI 平台不影响 CLI 和账号连接。 |
| `getnote quota -o json` | `data.read/write/write_note` 下的 `daily/monthly.limit/used/remaining/reset_at` | 按真实桶说明剩余额度，不自行换算或合并桶。 |
| `getnote version` | 版本文本 | 只用于展示版本；机器契约仍以 `capabilities -o json` 为准。 |
| `getnote update --check` | 当前/可用新版本文本 | 有新版本再运行 `getnote update`。 |
| `getnote update` | CLI 更新、Skills 同步和 doctor 验证结果 | 默认由新版 CLI 自动完成完整更新闭环；只有全部步骤成功才能说升级完成。只升级 CLI 时使用 `--cli-only`。 |

所有命令以退出码为第一判断：退出码非 0 即失败。使用 `-o json` 的 API 与本地错误均返回 `success=false`、`data=null`、`error.code/message/reason/retryable` 和可选 `request_id`；不能把 HTTP 200 或“命令运行过”当成成功。

### Doctor 机器决策规则

1. `diagnostics_completed=false` 或命令退出非 0：诊断本身失败，不能根据残缺输出操作。
2. `ready=true`：CLI、账号授权和 OpenAPI 连通性均可用；若 `status=degraded`，核心能力可用但仍有非阻断警告。
3. `status=partial`：通常是 `--offline` 跳过远端检查；读取 `local_ready` 判断本地环境，`ready=null` 表示远端未知，不得声称已连接。
4. `issues[].blocking=true`：按顺序优先处理；保留 `code`、`details.request_id` 和错误字段。
5. `next_actions[]`：`requires_confirmation=true` 时先向用户确认，再执行精确 `command`；执行后重新运行 doctor 验证。
6. `integrations[].detected=true` 只表示发现宿主；只有 `skill_status=installed` 且 `ready=true` 才能确认 CLI 可验证的 Skill 已齐全。`unverified` 表示安装状态由平台管理，应让用户在平台内确认，不能猜测成功。
7. `update.update_available=true` 是非阻断警告；用户明确要求升级后才执行升级动作。

## 更新闭环

用户说“帮我更新得到大脑”已经构成完整更新授权；不要再让用户选择 CLI、领域 Skill 等内部组件：

1. 执行 `getnote update --check`；有新版本或需要刷新 Skills 时执行 `getnote update`。命令会升级 CLI，再由新版 CLI 自动运行 `setup` 和 `doctor`，并同步 CLI 随附的五个领域 Skill。
2. ClawHub/OpenClaw 托管的独立聚合 Skill 由平台更新；`getnote update` 不下载或覆盖该 Skill。若宿主支持检查 Skill 更新则继续使用宿主流程，否则只告知用户唯一必要的确认入口。
3. 检查更新输出中的 CLI 版本、五个领域 Skill 同步结果和 doctor 结果。
4. 用最近笔记读取做验收，再告诉用户版本、诊断结果和仍需动作。

## 安全与恢复

- 不展示或记录完整凭证；`auth status` 只能出现掩码。
- 用户未明确要求时不退出登录。
- 授权超时、拒绝或验证码过期时重新启动一次登录流程，不复用旧 code。
- 失败时保留执行步骤、错误原因和 `request_id`；不要只回复“连接失败”。
