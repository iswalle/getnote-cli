---
name: getnote-kb
description: 查看和管理得到大脑的默认知识库、书籍、客户档案、团队知识库、文件夹、抖音博主订阅与直播，并把笔记准确归档到指定知识库和目录。
---

# 得到大脑知识库

通过官方 CLI 读取真实知识库、权限和目录后再操作；不能用名称猜 ID。机器调用统一使用 `-o json`；只有退出码为 0 且 `success=true` 才是业务成功。返回中的所有知识库、目录、博主和笔记 ID 都按字符串原样传递。

所有 API 命令的 JSON 都先判断 `success`。`success=false` 或退出码非 0 时读取 `error.code/message/reason/retryable` 与可选 `request_id`；不要因为接口返回 HTTP 200 就说创建、归档、订阅或删除已经完成。

## 意图路由

| 意图 | 命令入口 |
|---|---|
| 自有/可管理知识库 | `getnote kbs` |
| 我订阅的知识库 | `getnote kbs-sub` |
| 知识库笔记 | `getnote kb <topic_id>` |
| 新建个人知识库 | `getnote kb create` |
| 加入笔记 | `getnote kb add` |
| 移出笔记 | `getnote kb remove` |
| 浏览文件夹 | `gnote kb dir` |
| 创建文件夹 | `gnote kb mkdir` |
| 重命名/移动文件夹 | `gnote kb mvdir` |
| 删除空文件夹 | `gnote kb rmdir` |
| 博主列表 | `getnote kb bloggers` |
| 博主内容列表 | `getnote kb blogger-contents` |
| 博主内容详情 | `getnote kb blogger-content` |
| 订阅抖音博主 | `getnote kb blogger-follow` |
| 直播列表 | `getnote kb lives` |
| 直播详情 | `getnote kb live` |
| 订阅直播 | `getnote kb live-follow` |

`gnote` 和短命令是稳定别名；旧环境没有别名时回退到 `getnote kb directories/directory-create/directory-update/directory-delete`。参数一律以目标命令 `--help` 为准。

## 选择知识库和权限

1. 用户问“我管理的/我的知识库”时执行 `getnote kbs -o json`；问“所有能访问的知识库”时还要执行 `getnote kbs-sub -o json`，分别标明自有/可管理与已订阅，按各自 `has_more` 翻完页，不能混成同一种权限。
2. `getnote kbs` 必须保留接口实际返回的全部 Scope，例如 `DEFAULT`、`BOOKSPACE`、`CUSTOMER`、`TEAMSPACE`，不能只返回默认知识库，也不能虚构未返回的 Scope。
3. 按名称和 `scope` 匹配；同名或用户意图不明确时让用户选择，不猜 `topic_id`。目标名称既可能是知识库也可能是文件夹时，先问清楚。
4. 订阅知识库通常只读。若列表结果明确返回角色或可写标记，按真实字段判断；当前 CLI 契约未保证权限字段时，不得声称已经预检为 owner/admin，可在用户明确授权后尝试写入，并忠实处理权限失败。
5. 普通成员写入失败时明确说明权限不足，不尝试绕过。当前不代替用户新建团队知识库。

## 文件夹和归档流程

1. 用户要求放入文件夹时，用 `gnote kb dir <topic_id> -o json` 读取根目录或指定目录。
2. 已有文件夹使用 `data.directories[].id` 返回的真实目录 ID；浏览指定目录时把该值传给 `--directory-id`。缺失时先询问是否创建，再用 `mkdir`。
3. `getnote kb add` 同时传真实 `topic_id`、字符串 `note_id` 和 CLI 帮助中规定的目录参数。
4. 每批最多 20 条。移出笔记和删除目录必须先确认；删除目录还必须由 CLI/服务校验为空。
5. 移动或重命名时只改变用户指定项，未指定的名称或父目录保持不变。

## 博主和直播

1. 用户给出抖音主页并要求持续关注时，先确认目标知识库和写权限，再使用 `blogger-follow`；只是找某条内容时先查询，不创建订阅。
2. 列表先返回博主/直播名称、真实字符串 ID 和必要状态，选中后再读取完整内容。
3. 博主内容详情中的 `post_media_text` 才是完整原文，不用摘要冒充。

## 每条命令的结果与回复格式

| 命令 | 成功结果必须包含 | 回复规则 |
|---|---|---|
| `getnote kbs -o json` | `success=true`、`data.topics[].topic_id/name/scope/stats`、`has_more/total` | 展示全部真实 Scope，不只展示默认知识库。 |
| `getnote kbs-sub -o json` | `success=true`、`data.topics[].topic_id/name`、`has_more/total` | 订阅知识库通常只读，不能把它当作可写知识库。 |
| `getnote kb <topic_id> -o json` | `success=true`、`data.notes[].note_id/title/note_type`、`has_more/total` | 返回知识库内真实笔记；需要链接时再用 `note` 读取详情。 |
| `getnote kb create <name> -o json` | `success=true`、`data?` | 仅创建个人知识库；不得在没有返回 ID 时虚构 `topic_id`。 |
| `getnote kb add <topic_id> <note_id…> -o json` | `success=true`、`data?` | 最多 20 条；需要确认最终目录归属时重新读取目录。 |
| `getnote kb remove <topic_id> <note_id…> --yes -o json` | `success=true`、`data?` | 最多 20 条；先确认，再说明已从该知识库移出。 |
| `gnote kb dir <topic_id> -o json` | `success=true`、`data.current_directory?`、`directories[].id/name`、`resources[].id/directory_id/note_id?/name/type/status`、`total` | 目录主键读取 `directories[].id`；资源归属读取 `resources[].directory_id`，笔记资源用 `resources[].note_id` 验证；旧环境回退 `getnote kb directories`。 |
| `gnote kb mkdir <topic_id> --name <name> -o json` | `success=true`、`data?` | 也兼容位置参数 `<name>`；二者不可同时使用。若需给出新目录 ID 或层级，随后重新读取目录。 |
| `gnote kb mvdir <topic_id> <directory_id> … -o json` | `success=true`、`data?` | 只确认用户指定的改名/移动；需要最终名称或父级时重新读取目录。 |
| `gnote kb rmdir <topic_id> <directory_id> --yes -o json` | `success=true`、`data?` | 只删除空目录；说明已删除前必须拿到业务成功。 |
| `getnote kb bloggers <topic_id> -o json` | `success=true`、`data.bloggers[].follow_id_str/account_name/platform`、`has_more/total` | 列表中使用 `follow_id_str` 作为后续查询 ID。 |
| `getnote kb blogger-follow <topic_id> <link> -o json` | `success=true`、`data.follow_id_str/url/platform/type/created_at` | 说明实际订阅的平台和对象；先确认目标知识库有写权限。 |
| `getnote kb blogger-contents <topic_id> <follow_id> -o json` | `success=true`、`data.contents[].post_id_alias/post_title/post_publish_time`、`has_more/total` | 返回标题与摘要；阅读全文前让用户选择具体内容。 |
| `getnote kb blogger-content <topic_id> <post_id> -o json` | `success=true`、`data.post_title/post_summary?/post_media_text?/post_url?/post_publish_time` | `post_media_text` 才是完整原文，摘要不能替代它。 |
| `getnote kb lives <topic_id> -o json` | `success=true`、`data.lives[].live_id/name/status`、`has_more/total` | 先列出真实直播，再按用户选择读取详情。 |
| `getnote kb live <topic_id> <live_id> -o json` | `success=true`、`data.post_title/post_summary?/post_media_text?/post_publish_time` | `post_media_text` 是直播原文/转写；没有时不凭摘要补写。 |
| `getnote kb live-follow <topic_id> <link> -o json` | `success=true`、`data.follow_id_str/url/platform/type/created_at` | 说明真实订阅对象和平台。 |

权限不足、目录非空、批量超限等失败必须原样解释，不伪造降级成功；保留 `request_id` 和 `retryable`。

## 完成、处理中与失败后的动作

- `kbs`、`kbs-sub`、`kb`、`dir`、博主和直播的空列表都是成功结果：如实说“目前没有”，不要自动创建知识库、目录或订阅。
- `kb create`、`kb add`、`mkdir`、`mvdir` 等写操作的 `data` 可能因服务版本不同而不提供完整对象。只在 `success=true` 时确认动作已提交；若要向用户返回新目录 ID、最终层级或笔记归属，必须紧接着重新读取 `gnote kb dir` 或 `getnote kb <topic_id>`，不能猜字段。
- `kb add` 即使请求成功，也不能把“已发起加入”说成“已进入某个目录”；只有重新读取后在 `resources[]` 的同一项出现对应字符串 `note_id` 和 `directory_id`，才能展示最终归属。
- `kb remove`、`rmdir` 前必须有用户明确确认并带 `--yes`；若 `reason=knowledge_directory_not_empty`，明确提示“目录非空，请先移出内容或删除子目录”，不要原请求重试。其它失败也应说明真实原因和下一步，绝不宣称已删除。
- 用户给的团队知识库不在 `getnote kbs` 返回中时，先说明当前账号没有访问权限；不能把个人知识库同名项替代成团队知识库。
- “这条笔记”只能复用当前会话中已经由 CLI 返回并验证过的字符串 `note_id`。没有可靠上下文时先请用户提供笔记 ID；不能从不受支持的私有链接、标题或雪花 ID 数值转换中猜测。
