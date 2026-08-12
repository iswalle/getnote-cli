---
name: getnote-auth
description: 使用 getnote CLI 登录、退出、诊断连接、查看版本和升级得到大脑执行组件。
---

# 得到大脑连接与升级

所有真实状态以 CLI 为准，不向用户索要或展示 API Key。

## 路由

| 意图 | 命令 |
|---|---|
| 登录 | `getnote auth login` |
| 查看登录状态 | `getnote auth status` |
| 退出登录 | `getnote auth logout` |
| 诊断连接 | `getnote doctor -o json` |
| 查看能力兼容性 | `getnote capabilities -o json` |
| 为本机 AI 安装领域 Skill | `getnote setup` |
| 查看 AI 对话额度 | `getnote quota` |
| 查看版本 | `getnote version` |
| 检查或执行升级 | `getnote update --check` / `getnote update` |

不确定参数时运行对应命令的 `--help`。

## 规则

- 正常笔记任务不需要重复运行 `capabilities`；只在首次使用、升级后或故障排查时检查。
- 只有 `doctor` 的 `auth` 与 `api` 都通过，才能说连接成功。
- 未授权时运行 `getnote auth login`，把浏览器授权步骤交给用户。
- 用户未明确要求时不要执行 `auth logout`。
- CLI 升级后重新运行 `doctor`；CLI 升级不会自动更新 Skill。
- 用户要求更新整套得到大脑能力时，升级 CLI 后运行 `getnote setup` 同步领域 Skill，再执行诊断。
