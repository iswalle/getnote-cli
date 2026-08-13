---
name: getnote-note
description: 使用得到大脑保存文字、链接、图片和长笔记，查看笔记详情、原文、录音转写、附件、时间线、快捷笔记和会议待办，并安全更新、删除或分享笔记。
---

# 得到大脑笔记

通过官方 `getnote` CLI 完成真实操作。不要自己拼 OpenAPI 请求、ID 或笔记链接；机器调用优先使用 `-o json`，以退出码和下述结果契约判断结果。

## 统一结果判定

所有 API 命令的 JSON 结果先看下面这层结构，再读取每条命令规定的 `data` 字段：

```json
{
  "success": true,
  "data": {},
  "request_id": "可选"
}
```

失败结果为 `success=false` 或命令退出码非 0，读取 `error.code`、`error.message`、`error.reason`、`error.retryable` 和可选 `request_id`。HTTP 成功、上传完成、出现任务 ID 或拿到空笔记链接，都不能替代最终成功结果。

## 意图路由

| 意图 | 命令入口 |
|---|---|
| 保存文字、链接或本地图片 | `getnote save` |
| 查询异步保存任务 | `getnote task` |
| 查看最近笔记 | `getnote notes` |
| 查看详情或字段 | `getnote note` |
| 链接/文字原文 | `getnote note original` |
| 录音、会议或课堂转写 | `getnote note transcript` |
| 图片、音频和文件附件 | `getnote note attachments` |
| 录音或会议时间线 | `getnote note timeline` |
| 录音快捷笔记 | `gnote note quick`，旧版回退 `getnote note quick-note` |
| 会议总结中的派生待办 | `getnote note todos` |
| 修改笔记 | `getnote note update` |
| 删除笔记 | `getnote note delete` |
| 公开分享 | `getnote note share` |

不确定参数时先运行目标命令 `--help`。

## 保存流程

### 文字与长文

1. 保留用户原意，不擅自扩写；未指定时不添加知识库、父笔记、标签或公开分享。
2. 短文本可作为参数传入。长文本、Markdown、含复杂引号或换行的内容必须使用 `--content-file` 或 `--stdin`，避免截断和转义损坏。
3. 重试同一次创建时复用同一个 `--idempotency-key`。
4. 只有命令退出码为 0，且最终结构中存在非空字符串 `data.note.note_id`、`data.note.title`、`data.note.note_url`，才回复保存成功。

### 链接

1. 以 `http://` 或 `https://` 开头且用户表达保存意图时按链接保存，不当作普通文字。
2. CLI 会轮询异步任务。处理中可以告诉用户“正在抓取并生成笔记”，但不能提前给出成功结论。
3. 最终成功必须满足文字保存的三项字段，并且 `data.note` 已能读取；不要自行拼接链接。

### 图片

1. 使用本轮用户明确给出的本地图片路径，不把文件名保存成文字，也不带上历史图片。
2. CLI 会校验真实文件格式、上传图片并轮询识别任务。
3. 只有最终笔记详情返回有效 `note_id/title/note_url` 才算成功；“图片已上传”不是“笔记已生成”。

### 异步超时与安全重试

- `getnote save ... -o json` 正常会等待最终结果；若退出码非 0 且输出含 `task_id`、`status=pending|processing`，操作结果仍不确定。
- 结果不确定时使用 `getnote task <task_id> -o json` 查询原任务。`done|success` 且有有效 `note_id` 后再读取笔记；`failed` 时展示 `error_msg` 或 `msg`。
- 超时、断流或网络错误后禁止直接再次保存；先查询原任务或最近笔记。只有 CLI/API 明确 `retryable=true` 且已确认原操作没有成功时才重试。

## 查询和深层读取

1. “最近、列表、有哪些”使用 `getnote notes`；“找、搜、关于某主题”交给搜索 Skill。
2. 用户给出 ID 时直接读取详情；雪花 ID 全程按字符串原样传递。“这条笔记”只复用当前会话中已经由 CLI 返回并验证过的字符串 ID。当前 CLI 若不能直接接收某种私有链接，就先请用户提供 ID，不能自行截取、猜测或转成数字。
3. 列表先展示标题、字符串 ID 和真实 `note_url`，用户选择后再读取全文。
4. 不确定笔记类型时先读 `getnote note <id> -o json`：
   - 链接/文字原文：`original`；
   - 录音、会议、课堂逐字稿：`transcript`；
   - 图片、音频、文件：`attachments`；
   - 时间点与会议过程：`timeline`；
   - 用户现场快捷记录：`quick-note`；
   - 会议待办：`todos`，必须保留 `source`，不得把规则解析结果说成上游原生待办。
5. 不拿 `content` 中的 AI 摘要冒充原文。

## 修改、删除和分享

1. 先读取目标笔记和当前版本，确认用户指向的对象。
2. 追加或前置内容必须使用 CLI 当前帮助中对应的增量语义，不用覆盖模拟追加。
3. 覆盖正文、替换全部标签、删除和公开分享必须先确认；确认后才使用 `--yes`。
4. 分享录音类笔记时，确认话术必须说明公开链接是否包含音频；用户不希望公开音频时使用 CLI 帮助中的 `--exclude-audio`。不能替用户默认决定音频公开范围。
5. 用户未要求公开时只返回私有 `note_url`，不自动生成分享链接。

## 每条命令的结果与回复格式

| 命令 | 成功结果必须包含 | 回复规则 |
|---|---|---|
| `getnote save … -o json` | `success=true`、`data.note.note_id/title/note_url` | 回复“已保存《标题》”和真实链接。链接/图片先只得到 `task_id` 时还未成功，继续查任务。 |
| `getnote task <task_id> -o json` | `data.task_id/status/note_id?/msg?/error_msg?` | 仅 `done`/`success` 且有 `note_id` 为完成；`pending`/`processing` 是处理中；`failed` 说明真实原因。 |
| `getnote notes -o json` | `data.notes[].note_id/title/note_url`、`has_more/cursor/total` | 返回标题、字符串 ID 和真实链接；下一页只用返回的 `cursor`。 |
| `getnote note <id> -o json` | `data.note.note_id/title/note_url/note_type`，按需读 `content` | 先给摘要和链接；私密全文只在用户明确要求时展开。 |
| `getnote note original <id> -o json` | `data.note_id/title/original` | 只把 `original` 当原文；空字段就是该笔记没有可用原文。 |
| `getnote note transcript <id> -o json` | `data.note_id/title/transcript` | 仅录音类笔记可用；不可用时如实说明。 |
| `getnote note attachments <id> -o json` | `data.note_id/title/attachments[]` | 列出真实附件，不把笔记封面冒充附件。 |
| `getnote note timeline <id> -o json` | `data.note_id/title/timeline` | 仅在存在录音/会议时间线时展示。 |
| `gnote note quick <id> -o json` | `data.note_id/title/quick_note` | 返回现场快捷笔记；旧环境回退 `getnote note quick-note`。 |
| `getnote note todos <id> -o json` | `data.note_id/title/meeting_todos[]` | 保留每项 `source`；这是从明确会议总结章节按规则解析，不说成上游原生待办。 |
| `getnote note update <id> … -o json` | `success=true`、`data?` | 修改正文或全量标签前必须 `--yes`；需要展示最终内容时再读一次 `note`。 |
| `getnote note delete <id> --yes -o json` | `success=true`、`data?` | 只能说“已移入回收站”，不是永久删除。 |
| `getnote note share <id> --yes -o json` | `success=true`、`data.note_id/share_id/share_url` | 只返回真实 `share_url`，并明确其为公开链接。 |

API 失败时回复失败步骤、`error.message/reason`、是否可重试和 `request_id`；不能把 HTTP 200 当业务成功。

群聊或共享会话中只先展示必要标题和链接，不主动展开私密全文。
