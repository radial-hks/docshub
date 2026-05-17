# DocsHub

局域网文档共享中心 — 通过 CLI 发布 Markdown/HTML 文章到 Go 服务器，在浏览器中通过 Docsify 阅读和搜索。

[English](./README.md) | 中文

---

## 特性

- 单二进制，零依赖部署
- Markdown + HTML 双格式支持
- AI 自动分类（调用本地 LLM 推断标题、分类、标签）
- Frontmatter 元数据解析
- 文章版本管理
- Docsify 前端 + 全文搜索
- 纯局域网运行，无需认证

## 架构

```
┌──────────┐   HTTP/JSON   ┌────────────┐   静态文件   ┌─────────┐
│   CLI    │ ────────────> │   Server   │ ──────────> │ Docsify │
│ (push等) │               │ (Go HTTP)  │              │(浏览器)  │
└──────────┘               └────────────┘              └─────────┘
                                  │
                                  ▼
                           web/articles/*.md
                           web/articles/*.html
                           web/_sidebar.md
                           index.json
```

- **CLI** (`docshub`) — 读取本地文件，解析 frontmatter，POST 到服务器
- **Server** — 存储文章到磁盘，维护 `index.json`，自动生成 `_sidebar.md`，托管 Docsify 前端
- **Docsify** — 渲染 `web/` 为文档站点，带侧边栏导航和全文搜索

## 快速开始

### 安装

从 [Releases](https://github.com/radial-hks/docshub/releases) 下载对应平台的二进制，或自行编译：

```bash
# 需要 Go 1.22+
go build -o docshub ./cmd/docshub
```

### 启动服务器

```bash
./docshub serve
```

默认监听 `:8080`，数据目录 `./web`。可通过环境变量覆盖：

```bash
DOCSHUB_PORT=9090 DOCSHUB_DATA=/data/docs ./docshub serve
```

浏览器打开 `http://localhost:8080` 即可看到 Docsify 前端。

### 配置 CLI

```bash
./docshub init
```

交互式输入服务器地址、作者名、AI 分类配置，保存到 `~/.docshub.json`。

### 发布文章

```bash
# 发布 Markdown 文章
./docshub push article.md

# 发布 HTML 文章（自动检测后缀）
./docshub push page.html

# 指定分类和标签
./docshub push article.md --category AI --tags llm,rag

# 使用 AI 自动分类
./docshub push article.md --classify

# 跳过确认直接发布
./docshub push article.md --yes
```

## CLI 命令

### `docshub init`

交互式配置。写入 `~/.docshub.json`。

### `docshub push <file> [flags]`

发布文章到服务器。

| Flag | 说明 |
|------|------|
| `--category <分类>` | 设置文章分类 |
| `--tags <标签>` | 逗号分隔的标签 |
| `--format <格式>` | 文章格式：`html` 或 `md`（默认自动检测文件后缀） |
| `--classify` | 调用本地 LLM 自动推断标题/分类/标签 |
| `--classify-json <JSON>` | 直接传入分类 JSON，如 `{"category":"AI","tags":["llm"]}` |
| `--yes` | 跳过确认提示 |

元数据优先级（从高到低）：

`--classify-json` > `--classify`（AI 结果）> CLI flags > frontmatter > 默认值

### `docshub list [flags]`

列出已发布文章。

| Flag | 说明 |
|------|------|
| `--category <分类>` | 按分类过滤 |
| `--tag <标签>` | 按标签过滤 |
| `--author <作者>` | 按作者过滤 |

### `docshub search <query>`

全文搜索文章标题和摘要。

### `docshub delete <id> [--yes]`

删除文章。默认需要确认，`--yes` 跳过。

### `docshub serve`

启动 DocsHub 服务器。

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/articles` | 创建/发布文章，请求体为 `PublishRequest` JSON |
| GET | `/api/articles` | 列出文章，支持 `category`、`tag`、`author`、`q`（搜索）查询参数 |
| GET | `/api/articles/{id}` | 获取单篇文章元数据 |
| DELETE | `/api/articles/{id}` | 删除文章 |
| GET | `/html/{category}/{slug}` | 浏览器原生渲染 HTML 文章 |

`web/` 下的静态文件（`index.html`、`_sidebar.md`、`articles/`）从 `/` 提供。

## 配置

### CLI 配置 (`~/.docshub.json`)

```json
{
  "server_url": "http://localhost:8080",
  "author": "radial",
  "classify_url": "http://localhost:11434/v1/chat/completions",
  "classify_model": "qwen2.5:7b"
}
```

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `server_url` | 服务器地址 | `http://localhost:8080` |
| `author` | 默认作者名 | 空 |
| `classify_url` | AI 分类 API 地址（OpenAI 兼容） | 空（禁用） |
| `classify_model` | AI 分类使用的模型 | `qwen2.5:7b` |

### 服务器环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DOCSHUB_PORT` | 监听端口 | `8080` |
| `DOCSHUB_DATA` | 数据目录 | `./web` |

## Frontmatter

Markdown 文件可在开头使用 YAML frontmatter：

```markdown
---
title: 文章标题
category: AI
tags: [llm, rag, workflow]
author: radial
---

# 正文从这里开始
```

任何字段均可被 CLI flags 覆盖。

## HTML 文章

DocsHub 同时支持 HTML 文章：

- `.html`/`.htm` 后缀的文件会被自动识别为 HTML 格式
- HTML 文章通过 `/html/{category}/{slug}` 路由提供浏览器原生渲染
- 也可使用 `--format html` 手动指定格式
- Markdown 文章继续由 Docsify 渲染，HTML 文章由浏览器直接渲染

## AI 自动分类

配置 `classify_url` 后，在 push 时使用 `--classify` 标志：

```bash
# 先配置 AI 分类（在 init 中或直接编辑 ~/.docshub.json）
# classify_url 指向 Ollama 或其他 OpenAI 兼容 API

./docshub push article.md --classify
```

流程：
1. 提取文章内容（前 3000 字符）发送给 LLM
2. LLM 返回建议的标题、分类、标签
3. 显示 AI 建议并等待用户确认
4. 用户确认后发布

LLM 不可用时自动回退到手动指定元数据，不阻断发布流程。

## 版本管理

重新发布同一篇文章（通过 API 传入 `version_of` 字段）时：

- 旧版本归档到 `web/articles/<category>/.versions/<slug>/v<N>-<日期>.md`
- 新内容写入原位置
- 版本历史记录在 `meta.json` 中
- 文章 `version` 字段自增

## 开发

### 运行测试

```bash
make test
# 或
go test ./...
```

### 构建

```bash
make build          # 构建单二进制
make dist           # 交叉编译所有平台 + 生成 SHA256 校验和
make clean          # 清理构建产物
```

### 项目结构

```
cmd/
  docshub/              # 单一入口点
internal/
  model/                # 共享类型（Article, PublishRequest, ...）
  server/               # 存储、侧边栏、HTTP 处理器
  cli/                  # 配置、push、list、delete、search、classify、serve
test/                   # 集成测试
web/                    # Docsify 前端，作为静态文件提供
  index.html
  articles/             # 运行时生成
  _sidebar.md           # 运行时生成
```

## License

MIT
