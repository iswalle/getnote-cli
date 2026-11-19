# 得到大脑（Get笔记）CLI

在终端里存链接、记文字、搜笔记、整理知识库。也可以交给 AI（Claude Code、Codex、Cursor、WorkBuddy 等）直接调用。

## 安装

**推荐：一条命令装好 CLI，并接入本机 AI**

```bash
npm install -g @getnote/cli@latest && getnote setup
```

它会自动找到本机装了哪些 AI，给它们装上得到大脑 Skill，再打开浏览器让你登录一次。装完运行 `getnote doctor` 确认一切正常。WorkBuddy 装完要重启；OpenClaw（小龙虾）、QClaw 会提示你去 ClawHub 点一下确认。

**只要命令行**

```bash
npm install -g @getnote/cli
getnote auth login
```

也可以在 [Releases](https://github.com/iswalle/getnote-cli/releases) 下载对应系统的程序，放进 `PATH`。

- 安装包已自带各平台程序，不需要访问 GitHub；国内用户配置 npm 镜像（如 npmmirror）即可
- pnpm 10 以上：在 `pnpm-workspace.yaml` 里加 `allowBuilds: { '@getnote/cli': true }` 后重新 `pnpm install`
- 装好后 `getnote` 和 `gnote` 两个命令都能用

## 登录和额度

```bash
getnote auth login     # 打开浏览器授权
getnote auth status    # 看登录状态
getnote quota          # 看当前额度
```

所有得到大脑用户授权后都可以调用，会员享有更高额度（[了解会员](https://www.biji.com/checkout?product_alias=9Ab36BB3ZD&spm=openapi_cli)）。额度用完后按提示的时间再试。

## 常用操作

```bash
# 存
getnote save https://example.com/article --tag 阅读   # 存链接，自动抓取正文
getnote save "周五前回复王总的邮件" --tag 待办        # 记一条文字
getnote save ./photo.jpg                              # 存图片
getnote save --content-file ./long.md --title 周报    # 长文本从文件读

# 找
getnote notes                                  # 最近 20 条
getnote search "LLM 推荐阅读"                   # 语义搜索
getnote search "产品设计" --kb <知识库ID>        # 只在某个知识库里搜
getnote notes --source vendor_moan --all       # 录音台的笔记（录音卡用 getseed）

# 读
getnote note <笔记ID>                           # 看详情
getnote note transcript <笔记ID>                # 录音、会议的转写原文
getnote note outline <笔记ID>                   # 长笔记先看目录
getnote note section <笔记ID> --keyword 定价     # 再按关键词或位置读一段

# 整理
getnote kbs                                    # 我的知识库
getnote kb add <知识库ID> <笔记ID>              # 加进知识库
getnote tag add <笔记ID> 重要                   # 打标签
getnote note share <笔记ID>                     # 生成公开分享链接
getnote note delete <笔记ID>                    # 删除（进回收站，可在 App 恢复）

# 导出
getnote notes --all -o json > all-notes.json
```

笔记正文里写上 `https://biji.com/note/<笔记ID>`，就能链接到另一条笔记。

## 给 AI 用

- 所有命令加 `-o json` 输出结构化结果，AI 可以直接读
- `skills/` 里是 5 个 Skill（笔记、搜索、知识库、标签、登录诊断），告诉 AI 什么时候用哪个命令。`getnote setup` 会自动装好；单独补装：`npx skills add iswalle/getnote-cli -y -g`
- `getnote capabilities -o json` 列出当前版本支持的全部命令和返回字段，AI 以它为准
- 笔记 ID 一律按字符串处理，不要转成数字

## 检查和升级

```bash
getnote doctor            # 检查安装、登录、网络和 AI 接入
getnote update --check    # 看有没有新版本
getnote update            # 升级 CLI，同步 Skill，再检查一遍
getnote update --cli-only # 只升级 CLI
```

## 全部命令

用 `getnote <命令> --help` 查看每个命令的参数。

| 分类 | 命令 |
| --- | --- |
| 登录与环境 | `auth login/status/logout`、`setup`、`doctor`、`quota`、`update`、`version`、`capabilities` |
| 保存 | `save`（链接、文字、图片）、`task`（查看链接处理进度） |
| 查找 | `notes`（可按来源、类型、时间筛选）、`search`、`references`（按课程或老师找） |
| 读取 | `note`、`note original/transcript/outline/section`、`note attachments/timeline/chapters/quick-note/todos`、`notes batch`、`marks`、`sprouts`、`sprout` |
| 修改 | `note update`、`note delete`、`note share`、`tag add/remove/list` |
| 知识库 | `kbs`、`kbs-sub`、`kb`、`kb create/add/remove`、`kb directories` 及目录增删改、`kb blogger-follow`、`kb live-follow` 等 |
| 文件上传 | `file-capabilities`、`file-token`、`upload`、`file-add`、`image-config`、`image-token` 等 |

## 配置

登录信息保存在 `~/.getnote/config.json`。也可以用环境变量，优先级更高：

| 变量 | 说明 |
| --- | --- |
| `GETNOTE_API_KEY` | API Key |
| `GETNOTE_CLIENT_ID` | Client ID |
| `GETNOTE_API_URL` | 换成别的 API 地址，一般不用设 |

## 从源码构建

需要 Go 1.24.2 以上：

```bash
git clone https://github.com/iswalle/getnote-cli.git
cd getnote-cli
make install
```

## 更新日志

| 版本 | 日期 | 主要变化 |
| --- | --- | --- |
| v1.10.1 | 2026-11-18 | 文档：说明所有用户都能调用，会员额度更高 |
| v1.10.0 | 2026-11-17 | 长笔记先看目录再按段读；表格输出显示完整笔记链接；修复部分系统找不到程序的问题 |
| v1.9.0 | 2026-11-04 | 遇到限流自动排队、有限重试；上传中断后能接着传，不重复上传 |
| v1.7.0 | 2026-10-13 | 按书名、作者找书；按引用来源和时间筛选笔记 |
| v1.6.0 | 2026-09-24 | 上传文件到知识库；读取标记、发芽报告和章节时间线 |
| v1.5.x | 2026-08 | 一键接入本机 AI；`doctor` 诊断；知识库目录管理；订阅抖音博主 |

更早的版本见 [Releases](https://github.com/iswalle/getnote-cli/releases)。

## 相关链接

- [得到大脑官网](https://biji.com)
- [开放平台](https://www.biji.com/openapi)
- [问题反馈](https://github.com/iswalle/getnote-cli/issues)

## License

[MIT](https://opensource.org/licenses/MIT)
