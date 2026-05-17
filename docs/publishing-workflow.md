# DocsHub 文章发布工作流

本文档描述使用 DocsHub 发布文章的完整流程，包括环境准备、日常发布、AI 分类、版本管理等场景。

---

## 目录

1. [环境准备](#1-环境准备)
2. [发布文章](#2-发布文章)
3. [元数据优先级](#3-元数据优先级)
4. [AI 自动分类](#4-ai-自动分类)
5. [HTML 文章](#5-html-文章)
6. [版本管理](#6-版本管理)
7. [文章管理](#7-文章管理)
8. [完整示例](#8-完整示例)
9. [常见问题](#9-常见问题)

---

## 1. 环境准备

### 启动服务器

```bash
# 默认端口 8080，数据目录 ./web
./docshub serve

# 自定义端口和数据目录
DOCSHUB_PORT=9090 DOCSHUB_DATA=/data/docs ./docshub serve
```

服务器启动后，浏览器访问 `http://localhost:8080` 查看 Docsify 前端。

### 初始化 CLI 配置

```bash
./docshub init
```

交互式输入：

| 字段 | 说明 | 默认值 |
|------|------|--------|
| Server URL | 服务器地址 | `http://localhost:8080` |
| Author | 默认作者名 | 空 |
| Classify URL | AI 分类 API 地址（OpenAI 兼容） | 空（禁用） |
| Classify Model | 分类模型名 | `qwen2.5:7b` |

配置保存在 `~/.docshub.json`，也可直接编辑：

```json
{
  "server_url": "http://localhost:8080",
  "author": "radial",
  "classify_url": "http://localhost:11434/v1/chat/completions",
  "classify_model": "qwen2.5:7b"
}
```

---

## 2. 发布文章

### 基本用法

```bash
./docshub push article.md
```

CLI 会：
1. 读取文件内容
2. 解析 YAML frontmatter（如有）
3. 展示元数据摘要，请求确认
4. POST 到服务器
5. 输出文章 ID、版本号和访问路径

### 带参数发布

```bash
# 指定分类和标签
./docshub push article.md --category AI --tags llm,rag

# 跳过确认直接发布
./docshub push article.md --yes

# 指定格式（覆盖自动检测）
./docshub push article.txt --format md
```

### 使用 Frontmatter

在 Markdown 文件开头添加元数据：

```markdown
---
title: Voxel API 探索
category: UE
tags: [Python, 3D, Voxel]
author: radial
---

# Voxel API 探索

正文内容...
```

Frontmatter 支持的字段：`title`、`category`、`tags`、`author`。字段可被 CLI 参数覆盖。

---

## 3. 元数据优先级

当多个来源提供同一字段时，按以下优先级（高 → 低）：

```
--classify-json  >  --classify (AI 结果)  >  CLI 参数  >  frontmatter  >  默认值
```

示例：frontmatter 写了 `category: Note`，但 CLI 传了 `--category AI`，最终 category 为 `AI`。

---

## 4. AI 自动分类

### 前提

配置 `classify_url` 指向 OpenAI 兼容 API（如 Ollama）：

```json
{
  "classify_url": "http://localhost:11434/v1/chat/completions",
  "classify_model": "qwen2.5:7b"
}
```

### 使用

```bash
./docshub push article.md --classify
```

流程：
1. 文章前 3000 字符发送给 LLM
2. LLM 返回建议的 title、category、tags、author
3. 终端展示 AI 建议值
4. 用户确认后发布

AI 返回的分类固定为以下之一：`AI`、`UE`、`DevOps`、`Research`、`Note`、`Other`。

### 直接传入分类 JSON

跳过 LLM 调用，直接指定元数据（优先级最高）：

```bash
./docshub push article.md --classify-json '{"title":"My Title","category":"AI","tags":["llm"],"author":"radial"}'
```

### 容错

- 未配置 `classify_url` → `--classify` 静默跳过，不阻塞发布
- LLM 调用失败 → 打印警告，回退到手动元数据
- LLM 返回格式异常 → 自动提取 JSON，解析失败则回退

---

## 5. HTML 文章

DocsHub 支持 Markdown 和 HTML 双格式。

### 格式自动检测

| 文件扩展名 | 自动识别为 |
|-----------|-----------|
| `.md` | Markdown |
| `.html` / `.htm` | HTML |
| 其他 | Markdown（默认） |

用 `--format` 手动覆盖：

```bash
./docshub push page.html              # 自动检测为 HTML
./docshub push page.txt --format html # 手动指定 HTML
```

### 渲染差异

| 格式 | 渲染方式 | 访问路径 |
|------|---------|---------|
| Markdown | Docsify 运行时渲染 | `/#/articles/{category}/{slug}` |
| HTML | 浏览器原生渲染 | `/html/{category}/{slug}` |

HTML 文章不经过 Docsify 的 Markdown 管线，直接以 `Content-Type: text/html` 返回。

---

## 6. 版本管理

### 发布新版本

```bash
# 先查看文章 ID
./docshub list

# 基于已有文章发布新版本（通过 API 的 version_of 字段）
```

> 注意：CLI 目前未暴露 `--version-of` 参数，需通过 API 直接调用：
>
> ```bash
> curl -X POST http://localhost:8080/api/articles \
>   -H 'Content-Type: application/json' \
>   -d '{
>     "title": "更新后的标题",
>     "content": "新内容...",
>     "category": "AI",
>     "tags": ["llm"],
>     "author": "radial",
>     "version_of": "original-article-id",
>     "format": "md"
>   }'
> ```

### 版本存储

旧版本归档到 `web/articles/{category}/.versions/{slug}/` 目录：

```
web/articles/AI/.versions/my-article/
├── v1-20260517.md       # 归档的旧版本文件
└── meta.json            # 版本历史记录
```

`meta.json` 内容：

```json
{
  "versions": [
    { "version": 1, "date": "2026-05-17T10:30:00Z", "file": "v1-20260517.md" }
  ]
}
```

新版本原地覆盖当前文件，版本号自增。

---

## 7. 文章管理

### 列表

```bash
# 列出所有文章
./docshub list

# 按分类过滤
./docshub list --category AI

# 按标签过滤
./docshub list --tag llm

# 按作者过滤
./docshub list --author radial
```

### 搜索

```bash
# 全文搜索（标题 + 摘要）
./docshub search "voxel api"
```

### 删除

```bash
# 需确认
./docshub delete <article-id>

# 跳过确认
./docshub delete <article-id> --yes
```

---

## 8. 完整示例

### 示例一：AI 辅助发布 Markdown

```bash
# 1. 确保 Ollama 在运行
ollama serve

# 2. 推送文章，让 AI 建议分类
./docshub push my-research.md --classify

# 终端输出示例：
# AI suggests:
#   Title:    RAG 系统架构对比
#   Category: AI
#   Tags:     rag, architecture, comparison
#   Author:   radial
#
# About to publish:
#   Title:    RAG 系统架构对比
#   Category: AI
#   Tags:     rag, architecture, comparison
#   Author:   radial
#   Format:   md
# Proceed? [Y/n/e]: y
#
# Published: rag-xi-tong-jia-gou-dui-bi-20260517 (v1)
# URL: http://localhost:8080/articles/AI/rag-xi-tong-jia-gou-dui-bi.md
```

### 示例二：发布 HTML 页面

```bash
# HTML 文件自动检测格式
./docshub push dashboard.html --category DevOps --yes

# 输出：
# Published: dashboard-20260517 (v1)
# URL: http://localhost:8080/html/DevOps/dashboard
```

### 示例三：带 Frontmatter 的文章

准备文件 `notes/vue3-composition.md`：

```markdown
---
title: Vue3 Composition API 实战
category: Note
tags: [vue3, frontend, composition-api]
author: radial
---

# Vue3 Composition API 实战

正文内容...
```

发布：

```bash
./docshub push notes/vue3-composition.md --yes
# Frontmatter 中的元数据自动读取，无需重复指定
```

---

## 9. 常见问题

### Q: 中文标题生成的 slug 是什么？

非 ASCII 字符无法生成有意义的 slug，会回退为 `article-{timestamp}` 格式。建议在 frontmatter 中指定英文 `title`，或接受自动生成的 ID。

### Q: Docsify 页面加载空白？

Docsify 的 JS/CSS 从 CDN（jsdelivr）加载。局域网无外网时需 vendor 本地资源（计划在后续版本实现）。

### Q: AI 分类结果不准怎么办？

- 分类仅限预设类别（AI/UE/DevOps/Research/Note/Other），模型可能将文章归入 Other
- 可用 `--category` 覆盖 AI 建议
- 可用 `--classify-json` 完全控制元数据
- 尝试更大的模型（如 qwen2.5:14b）提高准确度

### Q: 如何在同一台机器上运行 CLI 和 Server？

不需要额外配置。Server 默认监听 `:8080`，CLI 默认连接 `http://localhost:8080`，直接使用即可。

### Q: 发布后如何修改文章？

目前没有 update 命令。两种方式：
1. 删除旧文章，重新 push
2. 通过 API 的 `version_of` 字段创建新版本（旧版本自动归档）

---

## 数据流全景

```
                    ┌──────────────────────────────────────────┐
                    │              用户操作                     │
                    └──────┬───────────────────────┬───────────┘
                           │                       │
                    docshub push              docshub list/search/delete
                           │                       │
                           ▼                       ▼
┌──────────┐   HTTP/JSON   ┌──────────────────────────────────────┐
│   CLI    │ ────────────> │            Server (Go HTTP)           │
│          │               │                                      │
│  push:   │               │  Store.Create()                      │
│  1.读文件 │               │    ├─ 解析 frontmatter                │
│  2.解析FM │               │    ├─ slugify 标题                    │
│  3.AI分类 │               │    ├─ 写文件 (articles/{cat}/{slug})  │
│  4.确认   │               │    ├─ 更新 index.json                │
│  5.POST  │               │    └─ 刷新 _sidebar.md               │
│          │               │                                      │
│  list:   │               │  Store.List() → 过滤 + 返回           │
│  search: │               │  Store.List(query) → 全文匹配         │
│  delete: │               │  Store.Delete() → 删文件+更新索引     │
└──────────┘               └────────────┬─────────────────────────┘
                                        │
                               静态文件服务 (/)
                                        │
                                        ▼
                               ┌─────────────────┐
                               │    Docsify 前端   │
                               │                  │
                               │  Markdown: #/…   │
                               │  HTML: /html/…   │
                               │  搜索: 全文检索   │
                               └─────────────────┘
```
