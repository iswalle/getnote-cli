---
name: getnote-search
description: 在得到大脑查找笔记：按书名、课程名、文稿标题、作者、评论或模糊记忆选择来源定位与语义搜索，返回真实笔记和链接。
---

# 得到大脑搜索

通过官方 `getnote` CLI 搜索真实笔记。不要自己拼 OpenAPI 请求、笔记 ID 或访问链接。机器调用使用 `-o json`；只有退出码为 0 且 JSON 的 `success=true` 才是业务成功。

## 何时使用

- “关于某主题、模糊记忆”：使用 `getnote search`；明确书籍/来源或要全量时按下方路径选择。
- “最近有哪些笔记、按保存时间列出来”：交给笔记 Skill 的 `getnote notes`，不要把列表误当语义搜索。
- 用户选中某一条后，再交给笔记 Skill 用 `getnote note <note_id>` 读取详情；搜索阶段不自动修改、移动、分享或创建笔记。

## 先选查找路径

- 明确书籍（书名或书籍作者），尤其要“全部划线”：先读知识库参考。书名用 `kbs --scope BOOKSPACE --query`；只有作者名时用 `kbs --scope BOOKSPACE --author` 并省略 `--query`，不能把作者塞进只匹配名称的 query。书名和作者都已知时才同时传，两者为 AND 条件。必要时用真实 `--book-id` 消歧，再按用户要“全部笔记”还是“划线”选择是否加 `--note-type ref`，需要全文时按 ID 调用 `note`。
- 明确文章/课程（包括“某老师的课”）：按下方“课程、文稿与评论定位”执行。已有真实来源 ID/文章链接时直接精确筛选；没有候选也能按来源列定位信息。`topics=[]` 不阻断这条路线。
- 只有主题或模糊记忆：语义搜索 → 按字符串笔记 ID 去重 → 对选中候选读详情/原文。Top-K（最大 10）不是完整清单；零命中也不能证明全库没有相关笔记。

## 容易混淆的组合问法

| 用户说法 | 路径与不能丢失的条件 |
|---|---|
| “找博尔赫斯的书籍笔记” / “找提到博尔赫斯的笔记” | 前者按书籍 author 定位；后者是内容主题搜索，不把所有提到一个人的笔记都限制为其著作。 |
| “某课程共读《某本书》那篇文稿” | 明确课程/文稿时优先课程引用路径，书名在这里是文稿线索，不跳去 BOOKSPACE。 |
| “课程笔记” | 先区分得到课程引用与自己录制的课堂笔记；只有用户明确来源时才设 dedao/course_article，不能把课堂录音排除掉。 |
| “某库/某书里关于某主题” | 先定位真实 topic_id，再 search --kb；未命中按知识库参考在该库列表补查，不把空搜索当作库内没有，不得遗漏库范围。 |
| “上周保存/昨天修改的某课程笔记” | 列表用对应时间筛选；search 没有时间/来源筛选参数，语义候选需核对 timestamps/reference。不能把时间词拼进 query 就宣称完成了时间筛选。 |
| “某课程链接下的笔记” / “这篇文稿链接的划线” | 已知课程名/老师时查课程容器；只有主页 URL 时核对真实 reference.url，不猜课程 ID。文稿 URL 才传 --article-url。 |
| “全部笔记” / “全部划线” | 前者不默认加 note-type ref；书籍库可能有心得等非引用笔记。得到课程引用路径只能证明该来源下的划线集合，不能涵盖用户另写的课程心得或录音。 |

多条件查询要保留用户全部约束；从候选转入精确文稿列表时也保留已指定的 topic/time/source。条件冲突或作者、版本不符时说明实际候选，不静默删除条件。有正文主题限制时，来源匹配只证明“来自这里”，还需核对正文；截断 preview 未出现某词不能当作全文不包含。多个库/多本书合并结果时按字符串 note_id 去重，不能合并不同 ID 的不同划线。

增强结果的 `resource_type` 使用小写 note/file 等资源类型，旧顶层 `note_type` 保留大写 NOTE/FILE 兼容值；真正笔记类型在 `note.note_type`。用 `note.display_title` 展示，用 `snippet`（旧接口为 `content`）解释命中，不拿详情 preview 替换搜索片段。`snippet_source=unknown` 如实保留；`score=null` 是未知，不是 0；`returned_count` 是片段数，`distinct_note_count` 是去重笔记数。FILE 等非笔记结果没有笔记 ID 时只呈现真实资源信息，不能拼笔记链接。

## 课程、文稿与评论定位

先区分课程容器和单篇文稿；`search` 是 Top-K 语义召回，来源元数据出现在结果中不能证明课程范围参与召回。课程名、老师、主题、评论、时间和真实知识库范围均需保留。

### 已知课程名或老师：先查课程容器

使用 `getnote references list`，无需用户提供内部 ID，也不需要先建知识库：

```bash
getnote references list --source dedao --resource-type course --query '<课程名>' --author '<准确老师姓名>' --match-mode exact --all -o json
getnote notes --source dedao --course-type '<course_type>' --course-id '<course_id>' --note-type ref --all -o json
```

只传已知的 query/author，至少一个；二者为 AND 条件，query 只匹配课程名，author 精确匹配老师。课程名不完整时显式用 `--match-mode prefix`；不把“讲概率”这样的正文主题塞进课程名。结果同名或版本不明时列候选消歧；用返回的 `course_type`+`course_id`，当前支持66，不解析标题、URL或课节 ID 生成课程 ID。空结果页若 `pagination.has_more=true` 仍按真实 `next_cursor` 继续，保持所有条件；循环或失败表示遍历未完成。`--all` 成功的 `traversal.completed=true` 只证明当前同步、授权范围内的遍历，不是得到 App 的全部划线。

课程筛选同时包含课程直接引用和所属课节；显式增加 `--reference-type course_article` 会再限制为课节引用，`--reference-id` 仍是原课节 ID，组合按交集。需要时间条件时保留对应创建/编辑/记录更新时间参数。首版只支持个人笔记，course+topic_id 明确不支持；用户同时指定真实知识库时说明当前交集尚不可用，不能改查个人笔记后声称满足库范围。课程内语义搜索尚未开放，不能在全局 Top-K 后过滤冒充；可先精确列出课程笔记，再按需读正文核实主题。

新请求需当前服务端声明 `reference.resources.v1` / `note.course_filter.v1`，并核对 applied_filters。服务未支持时报告能力缺失，不自动删除课程条件继续，不把 CLI 安装成功当作服务已发布。旧参数和默认调用继续可用。

### 已知单篇文稿或文章 URL

直接使用已验证的 `reference.resource_id`+`reference.resource_type` 或 `reference.article.url` 精确筛选：

```bash
getnote notes --source dedao --note-type ref --reference-type course_article --reference-id '<真实课节ID>' --all -o json
getnote notes --source dedao --note-type ref --article-url '<真实文章URL>' --all -o json
```

仅有文稿标题时可语义查候选并核对真实 reference.article.title；找到了课程线索就进入课程容器流程缩小范围。课程主页 URL 与文稿 URL 不互换，不从编码链接猜 ID。没有名称线索的课程主页可沿用既有来源列表核对 reference.url，但不得假称服务端按课程名筛选；定位前5页的预算到达时说明未完成，用户要全部时继续合法分页。metadata_state.reference 为 partial/unavailable 时保留为待核验候选，不能凭缺失字段排除。

### 读取原文与评论

对不超过20个已定位的个人笔记，使用 `getnote notes batch '<note_id_1>' '<note_id_2>' --fields original -o json`；引用原文来自 ref_content，查个人评论改用 fields=content。保留每条 status、content_state、pending_ids、partial；deferred_budget 表示整条待读，不能说完整，也不自动并发20次单条读取。用户确需未读全文时使用旧 `getnote note original '<note_id>' -o json`。知识库共享详情继续原单条权限路径，不用个人批量绕过权限。

只记得评论时先语义搜索实质文本并读 content 核对；已指定课程就先按课程缩小，不因 preview 没出现评论而排除。定位候选多于20条时保留用户线索继续缩小或分页按需读取，不下载整个账号正文。同一篇文稿的不同笔记 ID 是不同划线，不按文章 ID 合并。

## 语义搜索执行步骤

仅当上面的路径选择需要语义搜索时执行；没有指定知识库不等于必须先语义搜索。

定位时可显式使用 `getnote search '<主题>' --response-mode snippet --max-snippet-chars 400 -o json`，需当前接口声明 search.snippet.v1。旧调用不加参数仍返回原契约。摘要 full 只表示本次命中的完整片段，不是全文；保留 snippet_source、truncated、content_state 和资源身份，未知来源/类型原样保留。需要全文时读真实笔记详情。

1. 用户没有指定知识库时，执行 `getnote search <query> --limit <1-10> -o json`。默认上限是 10，不能自行放大。
2. 用户指定知识库名称时，先按其类型执行 `getnote kbs --scope <scope> -o json`；默认查 `DEFAULT`，书籍、客户档案和团队知识库分别使用 `BOOKSPACE`、`CUSTOMER`、`TEAMSPACE`。用返回的真实 `topic_id` 和 `scope` 匹配；同名或意图不明确时必须让用户选择。
3. 已确认知识库后执行 `getnote search <query> --kb <topic_id> --limit <1-10> -o json`。
4. 只从返回值读取标题、摘要、字符串 `note_id` 和真实 `note_url`。没有 `note_id/note_url` 的非笔记结果可以展示内容，但不能伪造“打开笔记”链接。
5. 同一篇笔记可能因命中多个片段出现多次。面向用户列“几篇笔记”时必须按字符串 `note_id` 去重；若去重后不足用户要求的数量，可在上限 10 内增大 `--limit` 重搜，仍不足时如实返回实际数量。

## 命令结果与用户呈现

### 搜索

```bash
getnote search "支付流程" --limit 10 -o json
getnote search "客户反馈" --kb <topic_id> --limit 5 -o json
```

成功 JSON 的稳定字段：

| 字段 | 含义 | Agent 如何使用 |
|---|---|---|
| `success` | 业务是否成功 | 只有 `true` 才继续呈现结果。 |
| `data.results[]` | 搜索结果 | 空数组是一次成功的“未找到”，不是失败。 |
| `data.results[].title` | 笔记标题 | 优先用 note.display_title；旧响应再用 title，均缺失才显示未命名。 |
| `data.results[].note_id` | 笔记雪花 ID | 始终作为字符串原样传入后续 `note` 命令。 |
| `data.results[].note_url` | 真实笔记链接 | 只有非空时才给用户“打开笔记”。 |
| `data.results[].content` | 命中摘要或正文片段 | 只摘取与问题相关的短片段，不能冒充全文。 |
| `data.results[].score` | 相关性分数，可为 null | null 表示未知；仅用于内部排序，不向用户虚构“准确率”。 |

成功时按相关性给出编号列表，例如：

```text
这次搜索找到 3 条不同的相关笔记：
1. 《支付流程优化想法》——“用户等待时增加进度提示…”
   打开：<真实 note_url>
2. 《客户支付反馈》——“…”
   打开：<真实 note_url>
你想看哪一条的详情或原文？
```

若 `data.partial=true` 或存在 `data.warnings`，先说明检索有部分结果未能完成权限验证或读取，并按 warnings 解释原因；空结果也不能表述为完整检索后无命中。若 `data.results=[]` 或结果均不符合用户指定来源，先按已选路径完成同一请求范围内的来源补查；补查不是跨知识库或跨账号扩大范围。只有主题搜索且无可用来源线索时，说明“这次语义搜索未找到”，不要说接口失败或全库没有；不创建笔记、不编造结果。

## 失败、隐私与后续动作

- `success=false` 或退出码非 0：回复失败步骤、`error.message` / `error.reason`、是否 `retryable` 和可选 `request_id`；不要将 HTTP 成功或空输出说成搜索成功。
- 参数错误先读取 field/supported_values；限流保留 retry_after_seconds。CLI 已对已知只读瞬时429做最多额外2次、累计等待5秒的重试，Agent 不再叠加无界重试；日/月配额和写操作不自动重试。
- 若 CLI 提示“搜索服务响应超时”，这是检索未完成，不是“没有结果”。建议稍后重试，或缩小关键词、指定知识库后再试；不要伪造空列表。
- 检索结果可能包含私密正文。群聊或共享会话默认只展示标题、必要摘要和真实链接；用户明确要求后再展开全文。
- 用户选中结果后，复用返回的字符串 `note_id`，再读取 `getnote note <note_id> -o json`；不要从 URL 截取或转换为数字。
