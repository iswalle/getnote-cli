# getnote-cli

CLI tool for [Get笔记](https://biji.com) — manage notes and knowledge bases from the terminal.

Designed for both humans and AI agents.

## Installation

### npm (recommended)

```bash
npm install -g @getnote/cli
```

### Manual

Download the latest binary from [Releases](https://github.com/iswalle/getnote-cli/releases) and place it on your `$PATH`.

### Build from source

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make install
```

## Quick Start

```bash
# Authenticate (OAuth browser flow)
getnote auth login

# Save a URL
getnote save https://example.com/article --title "Great article"

# Save plain text
getnote save "Remember to review the docs" --tag work

# List recent notes
getnote notes

# List all notes (auto-paginate)
getnote notes --all
```

## Commands

### Authentication

```
getnote auth login                   Authenticate via OAuth (browser)
getnote auth login --api-key <key>   Authenticate with API key directly
getnote auth status                  Show authentication status
getnote auth logout                  Remove saved credentials
```

### Save & Tasks

```
getnote save <url|text>              Save a URL or text note
  --title <title>                    Optional title
  --tag <tag>                        Tag (repeatable)

getnote task <task_id>               Check the progress of an async save task
```

### Notes

```
getnote notes                        List recent notes (default 20)
  --limit <n>                        Number of notes (default 20)
  --since-id <id>                    Pagination cursor
  --all                              Fetch all notes (auto-paginate)

getnote note <id>                    Show note details
  --field <name>                     Output a single field as plain text
                                     (id, title, content, type,
                                      created_at, updated_at, url, excerpt)

getnote note update <id>             Update a note
  --title <title>
  --content <content>                plain_text notes only
  --tag <tags>                       Comma-separated, replaces existing tags

getnote note delete <id>             Delete a note (moves to trash)
  -y, --yes                          Skip confirmation
```

### Knowledge Bases

```
getnote kbs                          List all knowledge bases

getnote kb <topic_id>                List notes in a knowledge base
  --limit <n>                        Number of notes (default 20)
  --all                              Fetch all notes (auto-paginate)

getnote kb create <name>             Create a knowledge base
  --desc <description>

getnote kb add <topic_id> <note_id> [note_id...]     Add notes
getnote kb remove <topic_id> <note_id> [note_id...]  Remove notes
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--api-key <key>` | Override API key for this command |
| `-o, --output json\|table` | Output format (default: `table`) |
| `--env prod\|dev` | Target environment (default: `prod`) |

## Field Output (Pipe-friendly)

Use `--field` to extract a single value for use in scripts:

```bash
# Get note content
getnote note 1234567890 --field content

# Get source URL of a link note
getnote note 1234567890 --field url

# Pipe into another command
getnote note 1234567890 --field content | pbcopy
```

## Configuration

Credentials are stored in `~/.getnote/config.json`:

```json
{
  "api_key": "gk_live_xxx",
  "client_id": "cli_xxx"
}
```

Environment variables (higher priority than config file):

| Variable | Description |
|----------|-------------|
| `GETNOTE_API_KEY` | API key |
| `GETNOTE_CLIENT_ID` | Client ID |
| `GETNOTE_API_URL` | Override the API base URL |

## AI Agent Usage

All commands support `-o json` for machine-readable output. See the [skills/](./skills/) directory for agent skill documentation.

## License

MIT
