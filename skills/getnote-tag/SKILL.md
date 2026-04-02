---
name: getnote-tag
version: 0.2.0
description: Manage note tags in Get笔记 via the getnote CLI
---

# getnote-tag Skill

Add, list, and remove tags on notes.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")

## Commands

### List tags on a note

```
getnote tag list <note_id>
```

Returns all tags including their IDs (needed for `tag remove`).

```bash
getnote tag list 1896830231705320746
getnote tag list 1896830231705320746 -o json
```

---

### Add a tag

```
getnote tag add <note_id> <tag>
```

```bash
getnote tag add 1896830231705320746 工作
```

---

### Remove a tag

```
getnote tag remove <note_id> <tag_id>
```

> ⚠️ Requires **tag ID** (integer), not tag name. Run `getnote tag list <note_id>` first to get the ID.
> System tags cannot be removed.

```bash
# Get tag IDs first
getnote tag list 1896830231705320746 -o json

# Remove by tag ID
getnote tag remove 1896830231705320746 123
```

---

## Agent Usage Notes

- `tag remove` takes a **tag ID**, not a name — always call `tag list` first if you only have the name.
- For bulk tag replacement, use `getnote note update --tag` instead (replaces all tags at once).
- Exit code `0` = success; non-zero = error. Error details go to stderr.
