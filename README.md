# getnote-cli

A CLI for [Get笔记 (biji.com)](https://biji.com) — save URLs and notes, browse your knowledge base, pipe results to scripts or AI agents.

## Install

```bash
npm install -g @getnote/cli
```

Or grab a binary from [Releases](https://github.com/iswalle/getnote-cli/releases) and put it on `$PATH`.

```bash
# Authenticate (opens browser)
getnote auth login

# Or use an API key directly
getnote auth login --api-key gk_live_xxx
```

## Quick Start

```bash
# Save a link — getnote fetches the page and extracts content async
getnote save https://simonwillison.net/2024/llms-reading-list/ --tag ai

# Save a plain-text note
getnote save "Read 'Thinking Fast and Slow' before the Q3 review" --tag reading

# List your last 20 notes
getnote notes

# View a note
getnote note 1234567890

# Pipe the content somewhere
getnote note 1234567890 --field content | pbcopy

# Search / browse a knowledge base
getnote kbs                         # list all KBs
getnote kb vnrOAaGY                 # notes in a specific KB
getnote kb vnrOAaGY --all           # all pages, auto-paginated
```

## AI Agent Usage

Every command outputs JSON with `-o json`:

```bash
getnote notes -o json
getnote note 1234567890 -o json
getnote kbs -o json
getnote save https://example.com -o json   # polls silently, returns final note
```

The output is the raw API response — pipe it into `jq`, pass it to an LLM, or use it in scripts.

Claude Code skill definitions are in [`skills/`](./skills/) — add the directory to your Claude project for one-command note management inside any chat session.

## Commands

### Auth

```
getnote auth login                   OAuth browser flow
getnote auth login --api-key <key>   API key (non-interactive)
getnote auth status                  Show current credentials
getnote auth logout                  Remove saved credentials
```

### Save & Tasks

```
getnote save <url|text>              Save a URL or text note
  --title <title>
  --tag <tag>                        Repeatable

getnote task <task_id>               Check status of an async save
```

URL saves are async — the CLI polls automatically and shows the result when done. With `-o json` it polls silently and returns the final note object.

### Notes

```
getnote notes                        List recent notes (default 20)
  --limit <n>
  --since-id <id>                    Pagination cursor
  --all                              Fetch all (auto-paginate)

getnote note <id>                    Show note detail
  --field <name>                     Extract one field as plain text:
                                     id, title, content, type,
                                     created_at, updated_at, url, excerpt

getnote note update <id>
  --title <title>
  --content <content>                plain_text notes only
  --tag <tags>                       Comma-separated, replaces existing

getnote note delete <id>
  -y, --yes                          Skip confirmation
```

### Knowledge Bases

```
getnote kbs                          List all knowledge bases

getnote kb <topic_id>                Notes in a KB
  --limit <n>
  --all

getnote kb create <name>
  --desc <description>

getnote kb add <topic_id> <note_id> [note_id...]
getnote kb remove <topic_id> <note_id> [note_id...]
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--api-key <key>` | Override API key for this invocation |
| `-o, --output json\|table` | Output format (default: `table`) |
| `--env prod\|dev` | API target environment |

## Configuration

Credentials are stored at `~/.getnote/config.json`:

```json
{
  "api_key": "gk_live_xxx",
  "client_id": "cli_xxx"
}
```

Environment variables (override config file):

| Variable | Description |
|----------|-------------|
| `GETNOTE_API_KEY` | API key |
| `GETNOTE_CLIENT_ID` | Client ID |
| `GETNOTE_API_URL` | Override API base URL |

## Build from Source

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make install
```

## License

MIT
