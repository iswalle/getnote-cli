# Get笔记 CLI

Get笔记的命令行工具，让你在终端和 AI Agent 里直接管理笔记和知识库。

存链接、记文字、搜笔记、管知识库——一条命令搞定，支持脚本和 AI Agent 调用。

---

## 安装

```bash
npm install -g @getnote/cli
```

或者从 [Releases](https://github.com/iswalle/getnote-cli/releases) 下载对应平台的二进制文件，放到 `$PATH` 里。

---

## 三步开始用

**第一步：安装**
```bash
npm install -g @getnote/cli
```

**（可选）安装 AI Agent Skill**

在 Claude Code、Cursor 等 AI 编程工具里用自然语言操作笔记，需要额外安装 Skill：
```bash
npx skills add iswalle/getnote-cli -y -g
```

> ⚠️ 需先完成第一步安装 CLI，再安装 Skill。

**第二步：登录**
```bash
getnote auth login
```
会自动打开浏览器完成授权。也可以直接用 API Key：
```bash
getnote auth login --api-key gk_live_xxx
```

**第三步：开始用**
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

`skills/` 目录下有 Claude Code 专用的 Skill 文件，安装后 AI Agent 可以直接用自然语言操作笔记：

```bash
npx skills add iswalle/getnote-cli -y -g
```

安装后在 Claude Code / Cursor 里说「帮我搜一下关于 RAG 的笔记」即可直接调用。

---

## 完整命令参考

### 认证

```
getnote auth login                   OAuth 登录（浏览器授权）
getnote auth login --api-key <key>   直接用 API Key 登录
getnote auth status                  查看当前登录状态
getnote auth logout                  退出登录
```

### 保存笔记

```
getnote save <url|文字|图片路径>      保存链接/文字/图片笔记
  --title <标题>                      可选标题
  --tag <标签>                        标签（可重复）

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
                                     （id/title/content/type/
                                       created_at/url/excerpt）

getnote note update <id>             更新笔记
  --title <标题>
  --content <内容>                   仅文字笔记可用
  --tag <标签>                       逗号分隔，会替换现有标签

getnote note delete <id>             删除笔记（移入回收站）
  -y                                 跳过确认
```

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
getnote kbs                          列出所有知识库

getnote kb <topic_id>                知识库内的笔记
  --limit <n>
  --all                              获取全部

getnote kb create <名称>             新建知识库
  --desc <描述>

getnote kb add <topic_id> <note_id> [note_id...]     加入知识库
getnote kb remove <topic_id> <note_id> [note_id...]  移出知识库
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

---

## 从源码构建

需要 Go 1.21+。

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make install
```

---

## 相关链接

- [Get笔记官网](https://biji.com)
- [开放平台文档](https://openapi.biji.com/docs)
- [问题反馈](https://github.com/iswalle/getnote-cli/issues)

## License

[MIT](https://opensource.org/licenses/MIT)
