# 得到大脑（Get笔记） CLI

得到大脑（Get笔记）的命令行工具，让你在终端和 AI Agent 里直接管理笔记和知识库。

存链接、记文字、搜笔记、管知识库——一条命令搞定，支持脚本和 AI Agent 调用。

---

## 安装

### 让本地 AI 自动完成安装（推荐）

适用于 Codex、Claude Code 和 Cursor：

```bash
npx -y @getnote/cli@latest setup
```

这条命令会安装 CLI、识别本机支持的平台、安装五个原子 Skill，并引导完成一次得到大脑授权。安装后运行：

```bash
getnote doctor -o json
```

### 只安装命令行

```bash
npm install -g @getnote/cli
getnote auth login
```

或者从 [Releases](https://github.com/iswalle/getnote-cli/releases) 下载对应平台的二进制文件，放到 `$PATH` 里。

**Windows 用户**：推荐直接从 [Releases](https://github.com/iswalle/getnote-cli/releases) 下载 `.exe` 文件，放到 PATH 里。用 `npm install -g` 安装时，如果遇到 `Expand-Archive` 相关报错，可尝试：

```bash
npm install -g @getnote/cli --ignore-scripts
```

然后手动从 Releases 下载对应平台的二进制文件。

---

## 使用要求

> **需要得到大脑（Get笔记）会员**
>
> OpenAPI（包括本 CLI 和 Skill）仅对**得到大脑会员**开放。未开通会员时执行命令会提示 `OpenAPI 仅对会员开放`。
>
> 开通会员：[前往得到大脑会员购买页](https://www.biji.com/checkout?product_alias=9Ab36BB3ZD&spm=openapi_cli)。

---

## 开始使用

如果已经单独安装 CLI，还可以为 Codex、Claude Code、Cursor 补装内置的五个原子 Skill：
```bash
npx skills add iswalle/getnote-cli -y -g
```

使用 `getnote setup` 时不需要再执行这条命令。

登录：
```bash
getnote auth login
```
会自动打开浏览器完成授权。也可以直接用 API Key（需同时传入 Client ID）：
```bash
getnote auth login --api-key gk_live_xxx --client-id cli_xxx
```

开始用：
```bash
# 存一篇文章
getnote save https://example.com/article --tag 阅读

# 记一条文字
getnote save "周五前要回复王总的邮件" --tag 待办

# 看最近的笔记
getnote notes
```

---

## 典型使用场景

**场景 1：边看边存**
```bash
# 存链接，自动抓取页面内容
getnote save https://simonwillison.net/2024/llms-reading-list/ --tag ai

# 存完自动展示笔记内容，不用再手动查
```

**场景 2：搜索笔记**
```bash
# 全局搜索
getnote search "LLM 推荐阅读"

# 在某个知识库内搜索
getnote search "产品设计" --kb vnrOAaGY
```

**场景 3：管理知识库**
```bash
# 列出所有知识库
getnote kbs

# 查看某个知识库的笔记
getnote kb vnrOAaGY --all

# 把笔记加入知识库
getnote kb add vnrOAaGY 1234567890
```

**场景 4：脚本批处理**
```bash
# 导出所有笔记为 JSON
getnote notes --all -o json > all-notes.json

# 取出某条笔记的正文
getnote note 1234567890 --field content | pbcopy
```

---

## AI Agent 使用

所有命令支持 `-o json` 输出结构化数据，AI Agent 可以直接解析：

```bash
getnote notes -o json
getnote note 1234567890 -o json
getnote search "关键词" -o json
getnote save https://example.com -o json   # 自动轮询，返回最终笔记
```

保存、列表、详情和搜索的 JSON 结果都会返回当前环境可打开的 `note_url`。测试环境会返回测试站地址，生产环境返回 `https://www.biji.com/note/{id}`。

`skills/` 目录是 CLI 内置的五个原子 Skill，也是本地 Agent 的唯一维护源。安装后 AI Agent 可以直接用自然语言操作笔记：

```bash
npx skills add iswalle/getnote-cli -y -g
```

安装后在 Claude Code / Cursor 里说「帮我搜一下关于 RAG 的笔记」即可直接调用。

独立的聚合 Skill 面向 WorkBuddy、QClaw 和 OpenClaw 生态，只负责意图理解，底层仍调用同一个 CLI，不再维护第二套 OpenAPI 请求实现。

---

## 完整命令参考

### 认证

```
getnote auth login                   OAuth 登录（浏览器授权）
getnote auth login --api-key <key> --client-id <id>  直接用 API Key 登录
getnote auth status                  查看当前登录状态
getnote auth logout                  退出登录
getnote doctor                       检查安装、登录和 API 连通性
getnote capabilities                 查看当前版本的稳定能力
getnote setup                        为本机 AI 安装原子 Skill 并引导授权
```

### 保存笔记

```
getnote save <url|文字|图片路径>      保存链接/文字/图片笔记
  --title <标题>                      可选标题
  --tag <标签>                        标签（可重复）
  --topic-id <topic_id>               直接存入普通/书籍/客户档案知识库
  --parent-id <note_id>               创建子笔记（ID 推荐传字符串）
  --idempotency-key <key>             同一创建请求重试时复用的幂等键

getnote task <task_id>               查看异步任务进度
```

链接笔记是异步处理的，CLI 会自动轮询等待完成，完成后直接展示内容。`-o json` 模式下静默轮询，返回最终笔记 JSON。

### 查看和管理笔记

```
getnote notes                        最近 20 条笔记
  --limit <n>                        自定义数量
  --all                              获取全部（自动翻页）

getnote note <id>                    笔记详情
  --field <字段名>                   只输出某个字段的值
                                     （id/note_url/title/content/type/
                                       created_at/url/excerpt/
                                       web_content/audio_original）

getnote note update <id>             更新笔记
  --title <标题>
  --content <内容>                   仅文字笔记可用
  --tag <标签>                       逗号分隔，会替换现有标签

getnote note delete <id>             删除笔记（移入回收站）
  -y                                 跳过确认

getnote note share <id>              生成公开分享链接
  --exclude-audio                    分享时排除音频
```

### 字段语义说明（适合 AI Agent 参考）

不同笔记类型的"原文"字段不同，`content` 通常是 AI 总结而非原文：

| 笔记类型 | 原文字段 | AI 总结字段 |
|---------|---------|------------|
| 普通文字笔记 | `content` | `content` |
| 链接/网页笔记 | `web_content` | `content` |
| 录音笔记 | `audio_original` | `content` |
| 知识库博主内容 | `post_media_text`（via `kb blogger-content`）| `content` |
| 知识库直播 | `post_media_text`（via `kb live`）| `post_summary` |

> **AI Agent 提示**：用户要求"读原文"时，先用 `getnote note <id> -o json` 查看 `note_type`，再按上表选择对应字段。

### 搜索

```
getnote search <关键词>              全局语义搜索
  --limit <n>                        返回数量（默认 10）
  --kb <topic_id>                    限定在某个知识库内搜索
```

### 标签

```
getnote tag add <note_id> <标签>     给笔记加标签
getnote tag remove <note_id> <标签>  删除笔记标签
getnote tag list <note_id>           查看笔记的所有标签
```

### 知识库

```
getnote kbs                          列出所有自有知识库（DEFAULT / BOOKSPACE / CUSTOMER）

getnote kb <topic_id>                知识库内的笔记
  --limit <n>
  --all                              获取全部
  --no-content                       JSON 输出时省略 content 字段（节省 AI token）

getnote kb create <名称>             新建知识库
  --desc <描述>

getnote kb add <topic_id> <note_id> [note_id...]     加入知识库
getnote kb remove <topic_id> <note_id> [note_id...]  移出知识库
getnote kb live-follow <topic_id> <url>              订阅得到直播课，直播结束后 AI 摘要自动入库

getnote kbs-sub                                      获取我订阅的知识库列表
```

---

## 全局参数

| 参数 | 说明 |
|------|------|
| `--api-key <key>` | 临时覆盖 API Key |
| `-o, --output json\|table` | 输出格式（默认 table） |
| `--env prod\|dev` | 切换 API 环境 |

---

## 配置

凭证保存在 `~/.getnote/config.json`：

```json
{
  "api_key": "gk_live_xxx",
  "client_id": "cli_xxx"
}
```

也支持环境变量（优先级更高）：

| 变量 | 说明 |
|------|------|
| `GETNOTE_API_KEY` | API Key |
| `GETNOTE_CLIENT_ID` | Client ID |
| `GETNOTE_API_URL` | 覆盖 API 地址 |
| `GETNOTE_WEB_URL` | 覆盖笔记网页地址；通常无需设置，CLI 会随 API 环境自动选择 |

`GETNOTE_API_URL` 可传站点根地址、`/open` 或完整 `/open/api/v1`，业务请求和 OAuth 会同时使用该地址；未设置时仍使用生产环境。

### 新版 API 兼容约定

- `note_id`、`parent_note_id`、`children_ids`、`cursor` 等雪花 ID 按字符串无损输出；历史 `id` / `next_cursor` 数字字段继续保留。
- HTTP 200 且 `success:false` 仍按失败处理并返回非零退出码；错误保留 `code/reason/retryable/field/constraint/expected_type/request_id`。
- `getnote save --idempotency-key` 用于防止网络重试产生重复笔记；同一请求必须复用同一个键。

---

## 从源码构建

需要 Go 1.21+。

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make install
```

---

## 🚀 进阶用法：用笔记内链实践柳比歇夫时间日志法

柳比歇夫时间日志法的核心是**每天记录自己把时间花在了哪里**，事后统计、复盘、改进。

结合 得到大脑（Get笔记）内链，可以用脚本自动串联：

```bash
# 每天早上新建当天日志，内链到关联项目笔记
$ getnote save "今天工作日志 - 以下是进展：\n参考：biji.com/note/{note_id}" --tag 日志

# 每周查看本周日志，复盘时间分配
$ getnote search "工作日志" --limit 7 -o json
```

**内链格式**：在笔记正文里用 `https://biji.com/note/{note_id}` 引用其他笔记。示例：

```
参考上次的讨论：https://biji.com/note/1234567890000000001
```

---

## 🆕 更新日志

| 日期 | 版本 | 新能力 | 适合怎么用 |
|------|------|--------|------------|
| 2026-04-23 | **v1.1.1** | 1. 笔记内链<br>2. 保存分享链接自动变笔记 | 1. 用内链串联每天的工作日志和项目笔记，实践时间日志法<br>2. 收到别人发来的分享链接直接存入笔记 |
| 2026-04-16 | **v1.1.0** | 1. `getnote note share`：生成分享链接<br>2. `getnote kb live-follow`：订阅得到直播 | 1. 把笔记一键分享给朋友<br>2. 在知识库里订阅得到直播课，直播结束后 AI 摘要自动入库 |
| 2026-04-03 | **v1.0.x** | 1. `getnote kb bloggers/lives`：查看博主和直播列表<br>2. `getnote update`：自动升级 | 1. 查看订阅博主的内容更新和直播摘要<br>2. 直接运行 `getnote update` 升级到最新版 |

---

## 相关链接

- [得到大脑（Get笔记）官网](https://biji.com)
- [开放平台文档](https://www.biji.com/openapi)
- [问题反馈](https://github.com/iswalle/getnote-cli/issues)

## License

[MIT](https://opensource.org/licenses/MIT)
