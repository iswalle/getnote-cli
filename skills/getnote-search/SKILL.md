---
name: getnote-search
description: 在得到大脑的全部笔记或指定知识库中按自然语言进行语义搜索，返回真实标题、摘要、字符串 ID 和可打开的笔记链接。
---

# 得到大脑搜索

通过官方 `getnote` CLI 搜索真实笔记。不要自己拼 OpenAPI 请求、笔记 ID 或访问链接。机器调用使用 `-o json`；只有退出码为 0 且 JSON 的 `success=true` 才是业务成功。

## 何时使用

- “找找、搜一下、关于某主题、我之前记过什么”：使用 `getnote search`。
- “最近有哪些笔记、按保存时间列出来”：交给笔记 Skill 的 `getnote notes`，不要把列表误当语义搜索。
- 用户选中某一条后，再交给笔记 Skill 用 `getnote note <note_id>` 读取详情；搜索阶段不自动修改、移动、分享或创建笔记。

## 执行步骤

1. 用户没有指定知识库时，直接执行 `getnote search <query> --limit <1-10> -o json`。默认上限是 10，不能自行放大。
2. 用户指定知识库名称时，先执行 `getnote kbs -o json`，用返回的真实 `topic_id` 和 `scope` 匹配；同名、多个团队知识库或意图不明确时必须让用户选择。
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
| `data.results[].title` | 笔记标题 | 列表主标题；为空时如实显示“未命名笔记”。 |
| `data.results[].note_id` | 笔记雪花 ID | 始终作为字符串原样传入后续 `note` 命令。 |
| `data.results[].note_url` | 真实笔记链接 | 只有非空时才给用户“打开笔记”。 |
| `data.results[].content` | 命中摘要或正文片段 | 只摘取与问题相关的短片段，不能冒充全文。 |
| `data.results[].score` | 相关性分数 | 仅用于内部排序，不向用户虚构“准确率”。 |

成功时按相关性给出编号列表，例如：

```text
找到 3 条相关笔记：
1. 《支付流程优化想法》——“用户等待时增加进度提示…”
   打开：<真实 note_url>
2. 《客户支付反馈》——“…”
   打开：<真实 note_url>
你想看哪一条的详情或原文？
```

若 `data.results=[]`，回复“没有找到相关笔记；可以换关键词、时间范围或指定知识库再试”，不要说“接口失败”，也不要自动扩大检索范围、创建笔记或调用模型编造结果。

## 失败、隐私与后续动作

- `success=false` 或退出码非 0：回复失败步骤、`error.message` / `error.reason`、是否 `retryable` 和可选 `request_id`；不要将 HTTP 成功或空输出说成搜索成功。
- 若 CLI 提示“搜索服务响应超时”，这是检索未完成，不是“没有结果”。建议稍后重试，或缩小关键词、指定知识库后再试；不要伪造空列表。
- 检索结果可能包含私密正文。群聊或共享会话默认只展示标题、必要摘要和真实链接；用户明确要求后再展开全文。
- 用户选中结果后，复用返回的字符串 `note_id`，再读取 `getnote note <note_id> -o json`；不要从 URL 截取或转换为数字。
