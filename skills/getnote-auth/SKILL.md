---
name: getnote-auth
description: 使用 getnote CLI 登录、退出、诊断连接、查看版本和升级得到大脑执行组件。
---

# getnote-auth Skill

管理得到大脑 CLI 的连接、授权、诊断和升级。所有真实状态以 CLI 结构化输出为准。

## Preflight

```bash
getnote doctor -o json
getnote capabilities -o json
```

只有 `auth` 和 `api` 均通过，才能说连接成功。`contract_version` 应为 `2.0`。

## Commands

### Log in

```
getnote auth login [--api-key <key> --client-id <client_id>]
```

| Mode | Command | Description |
|------|---------|-------------|
| OAuth (recommended) | `getnote auth login` | Opens browser to authorize; saves credentials automatically |
| API Key | `getnote auth login --api-key <key> --client-id <id>` | 保存已有 Key 及所属 Client ID |

```bash
# OAuth flow (opens browser)
getnote auth login

# API key directly
getnote auth login --api-key gk_live_xxx --client-id cli_xxx
```

不要向用户索要或展示 API Key。只有用户明确选择手工 Key 登录且已安全提供时才使用该方式。

---

### Check status

```
getnote auth status
```

Shows whether authenticated and which API key is active.

```bash
getnote auth status
```

---

### Log out

```
getnote auth logout
```

Removes saved credentials from `~/.getnote/config.json`.

```bash
getnote auth logout
```

---

## Agent Usage Notes

- 首次使用或出错时优先运行 `getnote doctor -o json`，而不是只看本地状态。
- 未授权时运行 `getnote auth login`，把浏览器授权步骤交给用户。
- `--api-key` on any command is a temporary per-invocation override and does not save credentials.
- Credentials saved at `~/.getnote/config.json`; env vars `GETNOTE_API_KEY` / `GETNOTE_CLIENT_ID` take higher priority.
- `GETNOTE_API_URL` may override the business API host for an explicitly selected test environment; otherwise production remains the default.

## Upgrade

```bash
getnote update --check
getnote update
```

也可使用 `npm install -g @getnote/cli@latest` 完整升级 npm 包。升级后重新运行 `doctor` 与 `capabilities`。CLI 升级不会自动更新 Skill 文档。
