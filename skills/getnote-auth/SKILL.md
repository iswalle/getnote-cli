---
name: getnote-auth
version: 0.3.0
description: Manage authentication for Get笔记 CLI
---

# getnote-auth Skill

Log in, log out, and check authentication status.

## Commands

### Log in

```
getnote auth login [--api-key <key>] [--client-id <id>]
```

| Mode | Command | Description |
|------|---------|-------------|
| OAuth (recommended) | `getnote auth login` | Opens browser to authorize; saves credentials automatically |
| API Key | `getnote auth login --api-key <key>` | Saves key directly, no browser needed |

```bash
# OAuth flow (opens browser)
getnote auth login

# API key directly
getnote auth login --api-key gk_live_xxx

# API key + Client ID
getnote auth login --api-key gk_live_xxx --client-id cli_xxx
```

Get your API key at: https://www.biji.com/settings/developer (keys start with `gk_live_`)

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

- Always run `getnote auth status` first to verify authentication before other commands.
- If not authenticated, prompt the user to run `getnote auth login`.
- `--api-key` on any command is a temporary per-invocation override and does not save credentials.
- Credentials saved at `~/.getnote/config.json`; env vars `GETNOTE_API_KEY` / `GETNOTE_CLIENT_ID` take higher priority.
