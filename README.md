# getnote-cli

CLI tool for [Get笔记](https://getnote.ai) — manage notes and knowledge bases from the terminal.

Designed for both humans and AI agents.

## Installation

### npm (recommended)

```bash
npm install -g @getnote/cli
```

### Homebrew (macOS)

```bash
brew install iswalle/tap/getnote
```

### Manual

Download the latest binary from [Releases](https://github.com/iswalle/getnote-cli/releases) and place it on your `$PATH`.

### Build from source

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make build
```

## Quick Start

```bash
# Authenticate
getnote auth login --api-key <your-api-key>

# Save a URL
getnote note save https://example.com/article --title "Great article"

# Save plain text
getnote note save "Remember to review the docs"

# List recent notes
getnote note list

# List knowledge bases
getnote kb list
```

## Commands

### Authentication

```
getnote auth login --api-key <key>   Save API key to ~/.getnote/config.json
getnote auth status                  Show authentication status
getnote auth logout                  Remove saved API key
```

### Notes

```
getnote note save <url|text>         Save a URL or text note
  --title <title>                    Optional title
  --tag <tag>                        Tag (repeatable)

getnote note list                    List recent notes
  --limit <n>                        Number of results (default 20)
  --since-id <id>                    Pagination cursor

getnote note get <note_id>           Get note details
getnote note update <note_id>        Update title, content, or tags
getnote note delete <note_id>        Delete a note (prompts for confirmation)
getnote note task <task_id>          Check save-task progress
```

### Knowledge Bases

```
getnote kb list                      List all knowledge bases
getnote kb create <name>             Create a knowledge base
  --desc <description>

getnote kb notes <topic_id>          List notes in a knowledge base
  --limit <n>

getnote kb add <topic_id> <note_id> [note_id...]    Add notes
getnote kb remove <topic_id> <note_id> [note_id...] Remove notes
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--api-key <key>` | Override API key for this command |
| `--output json\|table` | Output format (default: `table`) |
| `--env prod\|dev` | Target environment (default: `prod`) |

## Configuration

The CLI stores credentials in `~/.getnote/config.json`:

```json
{
  "api_key": "your-api-key",
  "client_id": "getnote-cli"
}
```

Environment variables (higher priority than config file):

| Variable | Description |
|----------|-------------|
| `GETNOTE_API_KEY` | API key |
| `GETNOTE_API_URL` | Override the API base URL |

## AI Agent Usage

All commands support `--output json` for machine-readable output. See the [skills/](./skills/) directory for agent skill documentation.

## License

MIT
