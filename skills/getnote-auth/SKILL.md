---
name: getnote-auth
version: 0.2.0
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
| OAuth (recommended) | `getnote auth login` | Opens browser to authorize |
| API Key | `getnote auth login --api-key <key>` | Saves key directly, no browser |

```bash
# OAuth flow
getnote auth login

# API key directly
getnote auth login --api-key gk_live_xxx

# API key + Client ID
getnote auth login --api-key gk_live_xxx --client-id cli_xxx
```

API keys start with `gk_live_`. Get yours at: https://www.biji.com/settings/developer

---

### Check status

```
getnote auth status
```

```bash
getnote auth status
```

---

### Log out

```
getnote auth logout
```

Removes saved credentials from local config.

```bash
getnote auth logout
```

---

## Agent Usage Notes

- Always run `getnote auth status` first to verify authentication before other commands.
- If not authenticated, prompt the user to run `getnote auth login`.
- `--api-key` on any command is a temporary override and does not save credentials.
- Credentials are stored in `~/.getnote/config.json`; `auth logout` removes them.
