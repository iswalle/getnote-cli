---
name: getnote-tag
description: 查看得到大脑笔记已有标签，为笔记添加标签，或按真实标签 ID 安全删除标签，并避免误删系统标签或覆盖其他标签。
---

# 得到大脑标签

通过官方 `getnote` CLI 管理真实标签。不要自己构造标签 ID，也不要把标签名误当成删除参数。机器调用统一使用 `-o json`；退出码为 0 但 `success=false` 仍是失败。

## 路由与安全边界

| 用户意图 | 命令 | 不能做什么 |
|---|---|---|
| “这条笔记有哪些标签” | `getnote tag list <note_id>` | 不自动添加或删除。 |
| “给这条笔记加上产品优化” | `getnote tag add <note_id> <tag>` | 只新增指定标签，不覆盖原标签。 |
| “删掉这个标签” | 先 `tag list`，再 `getnote tag remove <note_id> <tag_id>` | 不能把标签名称直接当作 `tag_id`。 |
| “替换为这些标签” | 先读取当前标签，再交给 `getnote note update --help` | 属于覆盖性操作，必须先确认。 |

笔记 ID、标签 ID 及任何雪花 ID 始终作为字符串原样传递。系统标签不可删除；权限不足、标签不存在或笔记不存在时不能说“已经删掉”。

## 命令结果与用户呈现

### 查看标签

```bash
getnote tag list <note_id> -o json
```

成功 JSON：

| 字段 | 含义 | 回复规则 |
|---|---|---|
| `success=true` | 标签读取成功 | 才能展示列表。 |
| `data.note_id` | 实际读取的笔记 | 必须与目标笔记一致。 |
| `data.tags[].id` | 标签真实 ID | 仅用于后续删除，不必在普通回复中暴露。 |
| `data.tags[].name` | 标签名称 | 展示给用户。 |
| `data.tags[].type` | 普通/系统等标签类型 | 系统类型明确不能删除。 |

若 `tags=[]`，回复“这条笔记目前没有标签”，这是成功结果。

### 添加标签

```bash
getnote tag add <note_id> "产品优化" -o json
```

成功 JSON 必须有 `success=true`、`data.note_id` 与更新后的 `data.tags[]`。回复“已添加「产品优化」”，并在用户要求时列出当前全部标签；不要说“已替换标签”。如果结果没有新的标签集合，先再执行一次 `tag list` 核验，再向用户确认。

### 删除标签

```bash
getnote tag remove <note_id> <tag_id> -o json
```

删除前必须从当前 `tag list` 结果中拿到真实 `tag_id`，并确认该条不是系统标签。成功返回 `success=true` 才能回复“已删除「标签名」”；如果用户需要看剩余标签，再执行 `tag list`，不能猜测剩余集合。

## 失败与恢复

- 失败时展示实际 `error.message` / `error.reason`、是否可重试和 `request_id`；不要把“标签不存在”“系统标签”“无权限”归为通用网络错误。
- 不能确定用户要删除哪一个同名标签时，先列出名称和类型让用户确认；不要按第一个结果擅自删除。
- 在群聊中仅展示用户当前要求的标签信息；不通过标签列表推断或泄露笔记其他正文。
