---
name: getnote-note
description: 使用得到大脑保存文字、链接、图片和长笔记，按录音台或录音卡来源查找笔记，查看笔记详情、原文、录音转写、附件、时间线、标记、发芽报告、快捷笔记和会议待办，并安全更新、删除或分享笔记。标记、发芽与时间线分别读取。
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
| 按预算读取少量个人笔记正文/原文 | `getnote notes batch` |
| 查看详情或字段 | `getnote note` |
| 引用/链接/文字原文 | `getnote note original` |
| 录音、会议或课堂转写 | `getnote note transcript` |
| 图片、音频和文件附件 | `getnote note attachments` |
| 录音或会议时间线 | `getnote note timeline` |
| 总结中的章节时间线 | `getnote note chapters` |
| 录音笔记标记（独立数据） | `getnote marks <note_id>` |
| 按月列出发芽报告 | `getnote sprouts <YYYY-MM>` |
| 读取发芽报告原文 | `getnote sprout <id>` |
| 录音快捷笔记 | `gnote note quick`，旧版回退 `getnote note quick-note` |
| 会议总结中的派生待办 | `getnote note todos` |
| 修改笔记 | `getnote note update` |
| 删除笔记 | `getnote note delete` |
| 公开分享 | `getnote note share` |

不确定参数时先运行目标命令 `--help`。

## 录音台与录音卡来源定位

设备来源与笔记类型是两个独立条件。用户明确说“录音台”“AI录音台”“墨案/MOAN录音台”时使用 `--source vendor_moan`；“录音卡”“GetSeed录音卡”使用 `--source getseed`。查设备全部笔记时只限定来源，不额外加 `--note-type audio` 或 `meeting`，避免漏掉其他实际类型；录音卡长录音和闪念也由同一个 getseed 来源覆盖。用户明确只要长录音/闪念时，才分别叠加 `recorder_audio` / `recorder_flash_audio`。

```bash
getnote notes --source vendor_moan --page-size 20 --all -o json
getnote notes --source getseed --page-size 20 --all -o json
```

“昨天/上周的录音台笔记”同时保留对应 `--created-from/to`，按用户时区换算为 RFC3339，from 包含、to 不包含；明确修改时间才改用 edited 或 record-updated。翻页始终保留 source、时间、知识库等原条件，核对 `applied_filters.source` 和每条笔记的 `source`，按真实游标继续并按字符串 note_id 去重。仅在遍历结束、没有未解决的 partial/warnings 或错误时，才能说当前授权范围已遍历；旧 total 不是设备范围的总条数。

结果只涵盖已经同步成笔记且当前账号有权读取的主笔记，不代表设备端未同步、尚未形成笔记的全部会议记录。空结果不能改查其他来源后声称完成，不能用“AI录音台”标签或标题猜来源。只说“录音笔记”而未限定设备时不加 vendor_moan/getseed；按已知录音模式核对真实类型，模式不明时保留不同类型候选并说明范围。

来源内有正文主题时，先在该来源和时间范围列出候选，再按需读取目标正文核实；`search` 不提供设备来源筛选，不能把全局 Top-K 后过滤当作该设备完整搜索。录音台/录音卡转写使用 `getnote note transcript '<note_id>' -o json`，不把 AI 总结、preview 或缺失转写当作原文。

## 来源筛选、轻量定位与原文

列表接口只返回定位信息，再用详情接口读取完整笔记。`display_title` 是展示标题，不覆盖原始 `title`；引用笔记空标题很正常。`preview` 是带来源和截断标记的定位片段，不是全文。

```bash
getnote notes --source dedao --note-type ref --page-size 20 -o json
getnote note '<note_id>' -o json
getnote note original '<note_id>' -o json
```

| note_type | 原文字段 | content 的语义 |
|---|---|---|
| ref | ref_content | 用户评论，不是引用原文 |
| plain_text | content | 用户正文 |
| link | web_page.content | 通常是摘要 |
| audio / local_audio | audio.original | 通常是摘要 |

录音按真实类型选择：即时录音/普通录音用 `--note-type audio`，上传的音频文件用 `local_audio`，自己录制的课堂笔记用 `class_audio`，会议录音用 `meeting`，系统内录用 `internal_record`，录音卡长录音/闪念分别用 `recorder_audio` / `recorder_flash_audio`。这些类型与得到课程引用不同；用户未限定平台时不加 `--source dedao` 或课程筛选。只说“录音”且模式不明时保留候选并说明类型，不能把上传音频当即时录音。

```bash
getnote notes --note-type audio --all -o json
getnote notes --note-type local_audio --all -o json
```

`original` 只按真实类型读取上述字段，并返回 `original_source`；原文缺失或类型不支持会报错，不能把评论或定位片段补成原文。列表中的 `content_state` 会标明正文被省略；详情接口返回完整正文和元数据。用 `metadata_state.reference/topics` 区分完整、部分、不可用的元数据；缺失的作者、书 ID、来源 URL 不猜测补写。

得到课程无需知识库：已知课程名/老师时先按搜索 Skill 执行 `getnote references list --source dedao --resource-type course --query '<课程名>' --all -o json`，再用真实课程容器身份筛选；author 精确匹配老师，query 可显式 prefix。同名版本先消歧，topics=[] 不阻断；不要求用户提供内部 ID。

```bash
getnote notes --source dedao --course-type '<course_type>' --course-id '<course_id>' --note-type ref --all -o json
```

课程容器身份是 course_type+course_id；当前66，不能用课节65的 resource_id 代替。课程筛选包含课程直接引用和所属课节，增量 reference.course 不改旧 resource_id。首版 course+topic_id 不支持，不用个人笔记冒充知识库交集。新能力缺失时报告不支持，不删除筛选继续查询。

单篇文稿已有真实 reference ID 或文章 URL 时继续使用旧精确筛选：

```bash
getnote notes --source dedao --note-type ref --reference-type course_article --reference-id '<reference.resource_id>' --all -o json
# 缺少资源 ID，但有 reference.article.url 时，可改用精确文章 URL
getnote notes --source dedao --note-type ref --article-url '<reference.article.url>' --all -o json
```

平台值取 `reference.source`；资源类型取 `reference.resource_type`，不要用标签或内部 ptype 猜枚举。`--reference-id` 必须同时传 `--source` 和 `--reference-type`。课程文稿的 resource_id 指向单篇文稿，课程筛选再加 reference-id 时取交集。resource_id、course_id、original_note_id、书 ID 与笔记 ID 是不同身份；全程用返回的字符串。课程主页不传到 --article-url，也不解析编码 ID；文章 URL 保留 query/fragment。同篇文稿的不同笔记 ID 仍是不同划线。

创建、正文编辑、记录更新是三个独立时间轴。按需使用 `--created-from/to`、`--edited-from/to`、`--record-updated-from/to`，值必须为带时区 RFC3339；区间 from 包含、to 不包含。不得把 `updated_at` 当正文编辑时间。“今天/上周”按用户当前日期和时区换算边界；保存时间不是文章发布日期，不把“最近”擅自解释成固定一个月。用户同时限定课程/知识库时保留这些条件。

新参数需要服务端对应 `meta.capabilities` 且 `applied_filters` 与请求一致；不支持时停止该查询并说明升级服务/客户端的需要，不自动移除筛选。全量按知识库参考中的分页规则验收；只拿一页或一次语义搜索，不能说“全部找到”。

分页结束与来源完整性分别判断：`partial=true`、`warnings`、候选 `metadata_state=partial` 仍须说明。特别是 `course_directory_no_lessons` 表示课程目录缺少课节，即使 `has_more=false` 或 `traversal.completed=true` 也不能保证找全老师的所有课程；保留已找到的真实结果和缺口，不通过删除筛选或重复同一请求消除警告。

## 有预算的批量正文读取

不超过20个已定位个人笔记用 `getnote notes batch '<note_id_1>' '<note_id_2>' --fields original -o json`。默认 fields=content,ref_content,reference；original 显式选择，引用原文来自 ref_content，个人评论来自 content。默认32000个Unicode码点，--max-total-chars 可设1..64000，服务端另有1MiB上限；不是给旧单条详情加截断。

退出码0和 success=true 只代表批量调用成功：逐条检查 status=ok/unavailable/failed/deferred_budget、content_state、partial、pending_ids。成功项正文完整，deferred_budget 整条待读，不能把 omitted 当空正文；未知未来状态原样保留并说明未确认。用户确需全文时再用旧 `getnote note original '<真实note_id>' -o json` 补读，不自动并发20次单条请求。不存在与无权统一 unavailable，不判断对象是否存在；首版不批量读知识库共享笔记，不绕过原单条权限。

参数错误读取 field/supported_values；限流保留 retry_after_seconds。CLI 已对已知只读瞬时429做最多额外2次、累计等待5秒重试；Agent 不叠加重试，日/月配额、写操作与上传凭证不自动重试。

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

1. “最近、列表、有哪些”使用 `getnote notes`；有明确书名或引用来源时按上面的来源/知识库路线查找，模糊主题才交给搜索 Skill。
2. 用户给出 ID 时直接读取详情；雪花 ID 全程按字符串原样传递。“这条笔记”只复用当前会话中已经由 CLI 返回并验证过的字符串 ID。当前 CLI 若不能直接接收某种私有链接，就先请用户提供 ID，不能自行截取、猜测或转成数字。
3. 用户只要列表时展示标题、字符串 ID 和真实 `note_url`。用户明确要某条或某来源的划线、原文、评论时，定位匹配候选后直接读详情完成请求，不额外等待用户选择；仅在候选仍有歧义、无法确认目标时请用户选择。
4. 不确定笔记类型时先读 `getnote note <id> -o json`：
   - 引用/链接/文字原文：`original`；
   - 录音、会议、课堂逐字稿：`transcript`；
   - 图片、音频、文件：`attachments`；
   - 章节时间点与会议过程：`chapters`，读取 `chapter_timeline.items`，保留规则解析来源 `source`；录音 moments 使用 `timeline`，两者不是同一份数据；
   - 用户现场快捷记录：`quick-note`；
   - 会议待办：`todos`，必须保留 `source`，不得把规则解析结果说成上游原生待办。
5. 不拿 `content` 中的 AI 摘要冒充原文。
6. 标记从 `getnote marks` 的 `data.marks[]` 读取，不使用 Timeline 的条数或内容替代。发芽从 `getnote sprouts` 列出报告，再用 `getnote sprout` 读取正文；`has_sprout=false` 如实表示没有可读报告，不把标记或笔记总结当作发芽。

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
| `getnote notes -o json` | `data.notes[].note_id/title/note_url`；新筛选响应另有 `pagination` | 返回标题、字符串 ID 和真实链接。新筛选按 `pagination.has_more/next_cursor` 翻页；仅无 `pagination` 的旧响应按 `has_more/cursor`。旧 `total` 不能当作筛选后数量。 |
| `getnote note <id> -o json` | `data.note.note_id/title/note_url/note_type`，按需读 `content` | 先给摘要和链接；私密全文只在用户明确要求时展开。 |
| `getnote note original <id> -o json` | `data.note_id/title/original/original_source` | 只把 `original` 当原文；原文不可用会失败，不能拿评论或摘要补全。 |
| `getnote note transcript <id> -o json` | `data.note_id/title/transcript` | 仅录音类笔记可用；不可用时如实说明。 |
| `getnote note attachments <id> -o json` | `data.note_id/title/attachments[]` | 列出真实附件，不把笔记封面冒充附件。 |
| `getnote note timeline <id> -o json` | `data.note_id/title/timeline` | 仅在存在录音/会议时间线时展示。 |
| `getnote note chapters <id> -o json` | `data.note_id/title/chapter_timeline` | `items[].start_ms` 为毫秒；空数组表示无可解析章节，不拿标记补齐。 |
| `gnote note quick <id> -o json` | `data.note_id/title/quick_note` | 返回现场快捷笔记；旧环境回退 `getnote note quick-note`。 |
| `getnote note todos <id> -o json` | `data.note_id/title/meeting_todos[]` | 保留每项 `source`；这是从明确会议总结章节按规则解析，不说成上游原生待办。 |
| `getnote note update <id> … -o json` | `success=true`、`data?` | 修改正文或全量标签前必须 `--yes`；需要展示最终内容时再读一次 `note`。 |
| `getnote note delete <id> --yes -o json` | `success=true`、`data?` | 只能说“已移入回收站”，不是永久删除。 |
| `getnote note share <id> --yes -o json` | `success=true`、`data.note_id/share_id/share_url` | 只返回真实 `share_url`，并明确其为公开链接。 |

API 失败时回复失败步骤、`error.message/reason`、是否可重试和 `request_id`；不能把 HTTP 200 当业务成功。

群聊或共享会话中只先展示必要标题和链接，不主动展开私密全文。
