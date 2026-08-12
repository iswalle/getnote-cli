---
name: getnote-note
description: 使用得到大脑保存文字、链接、图片和长笔记，查看笔记详情、原文、录音转写、附件、时间线、快捷笔记和会议待办，并安全更新、删除或分享笔记。
---

# 得到大脑笔记

通过官方 `getnote` CLI 完成真实操作。不要自己拼 OpenAPI 请求、ID 或笔记链接；机器调用优先使用 `-o json`，以退出码和下述结果契约判断结果。

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
2. 用户给出 ID 或私有笔记链接时读取详情；雪花 ID 全程按字符串原样传递。
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
4. 用户未要求公开时只返回私有 `note_url`，不自动生成分享链接。

## 结果契约与回复格式

- `save -o json`：最终成功是 `success=true` 且 `data.note` 有 `note_id/title/note_url`。回复“已保存《标题》”并附真实链接。
- `task -o json`：读取 `data.task_id/status/note_id/msg/error_msg`；处理中继续查询，失败说明原因，完成后再读详情。
- `notes -o json`：读取 `data.notes[]`、`data.has_more`、`data.cursor`；每项使用真实标题、字符串 ID、`note_url`。
- `note -o json`：读取 `data.note`。`original/transcript/attachments/timeline/quick-note/todos -o json` 统一返回 `success=true`、`data.note_id`、`data.title`，并在 `data` 中分别提供 `original`、`transcript`、`attachments`、`timeline`、`quick_note`、`meeting_todos`；不猜缺失字段。
- 更新/删除/分享：退出码 0 且业务 `success=true` 才算完成；分享只使用返回的 `share_url`。
- API 失败时回复失败步骤、`error.message/reason`、是否可重试和 `request_id`；不能把 HTTP 200 当业务成功。

群聊或共享会话中只先展示必要标题和链接，不主动展开私密全文。
