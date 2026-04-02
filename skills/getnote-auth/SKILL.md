---
name: getnote-auth
version: 0.1.0
description: Manage authentication for Get笔记 CLI
---

# getnote-auth Skill

Log in, log out, and check authentication status for the `getnote` CLI.

## Commands

### Log in

```
getnote auth login [--api-key <key>] [--client-id <id>]
```

Two modes:

| Mode | Command | Description |
|------|---------|-------------|
| OAuth (recommended) | `getnote auth login` | Opens browser to authorize; saves credentials automatically |
| API Key | `getnote auth login --api-key <key>` | Saves API key directly, no browser needed |

**Examples:**
```bash
# OAuth flow (opens browser)
getnote auth login

# Direct API key
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

Shows whether the CLI is authenticated and which API key is in use.

**Example:**
```bash
getnote auth status
```

---

### Log out

```
getnote auth logout
```

Removes the saved API key from local config.

**Example:**
```bash
getnote auth logout
```

---

## Agent Usage Notes

- Always run `getnote auth status` before other commands to verify authentication.
- If not authenticated, prompt the user to run `getnote auth login`.
- The `--api-key` flag on any command overrides the saved config temporarily (does not save).
- Credentials are stored locally; `auth logout` removes them.
