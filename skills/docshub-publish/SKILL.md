---
name: docshub-publish
version: 1.0.0
description: "Use docshub CLI to publish or update articles on LAN DocsHub server. Covers push, AI classify, HTML, versioning."
triggers:
  - publish article
  - push article
  - 发布文章
  - 推送文章
  - docshub push
  - 更新文章
  - 文章发布
---

# DocsHub 文章发布

使用 `docshub` CLI 将 Markdown/HTML 文章发布到局域网 DocsHub 服务器。

## 适用场景

- 用户说"发布文章"、"推送文章"、"publish"
- AI 生成的内容需要推送到 DocsHub
- 需要更新已发布的文章
- 批量发布多篇文档

## 前提条件

1. DocsHub 服务器运行中：`docshub serve`（默认 `:8080`）
2. CLI 已配置：`~/.docshub.json` 中 `server_url` 和 `author` 已填写
3. 二进制在 PATH 中或用绝对路径

## 发布流程

### 步骤 1：准备文件

文章可以是：
- **Markdown**（`.md`）— 主力格式，Docsify 渲染
- **HTML**（`.html`/`.htm`）— 浏览器原生渲染

如需指定元数据，在文件开头加 frontmatter：

```markdown
---
title: 文章标题
category: AI
tags: [tag1, tag2]
author: radial
---

正文内容...
```

Frontmatter 字段：`title`、`category`、`tags`、`author`，均为可选。

### 步骤 2：执行推送

```bash
# 基本发布
docshub push <file.md>

# Agent 辅助分类（推荐）
docshub push <file.md> --classify-json '{"title":"...","category":"AI","tags":["t1"]}' --yes

# 手动指定元数据（覆盖 frontmatter）
docshub push <file.md> --category AI --tags llm,rag

# 跳过确认
docshub push <file.md> --yes

# HTML 文章（扩展名自动检测，也可 --format html 强制指定）
docshub push <page.html>
```

### 步骤 3：确认输出

成功后 CLI 输出：
```
Published: <slug>-<date> (v1)
URL: http://localhost:8080/articles/<category>/<slug>.md
```

验证：用浏览器或 curl 访问该 URL 确认文章可读。

## 元数据优先级

从高到低：

```
--classify-json  >  --classify (AI)  >  CLI flags  >  frontmatter  >  defaults
```

- `--classify-json '{"title":"X","category":"AI","tags":["t1"]}'` — 最高优先级，跳过 LLM
- `--classify` — AI 建议值填充空字段
- `--category` / `--tags` — CLI 参数覆盖 frontmatter
- frontmatter — 文件内声明
- defaults — title 取文件名，author 取配置，category 空

## AI 辅助分类

发布时不需要额外配置 LLM 端点。分类由当前 Agent 或外部工具（Claude Code、Copilot 等）完成，结果通过 `--classify-json` 传入。

### 推荐流程

1. Agent 读取文章内容，生成分类结果
2. 将结果作为 JSON 传入 push 命令：

```bash
docshub push <file.md> --classify-json '{"title":"文章标题","category":"AI","tags":["llm","rag"],"author":"radial"}' --yes
```

### 手动分类

不用 AI 时，直接通过 CLI 参数指定：

```bash
docshub push <file.md> --category AI --tags llm,rag
```

### 备选：内置 LLM 分类

CLI 内置 `--classify` 标志，可调用本地 Ollama 等服务自动分类（需在 `~/.docshub.json` 配置 `classify_url`）。但在 Agent 工作流中不推荐——Agent 自身已具备分类能力，无需再调用外部模型。

预设分类：`AI`、`UE`、`DevOps`、`Research`、`Note`、`Other`

## 更新已发布文章

当前 CLI 未暴露 `--version-of` 参数，需通过 API：

```bash
# 1. 查找文章 ID
docshub list --category AI

# 2. 通过 curl 创建新版本
curl -X POST http://localhost:8080/api/articles \
  -H 'Content-Type: application/json' \
  -d "{
    \"title\": \"更新后的标题\",
    \"content\": \"$(cat new-content.md | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')\",
    \"category\": \"AI\",
    \"tags\": [\"llm\"],
    \"author\": \"radial\",
    \"version_of\": \"<article-id>\",
    \"format\": \"md\"
  }"
```

旧版本自动归档到 `web/articles/<category>/.versions/<slug>/`，版本号自增。

也可简化为：先 `docshub delete <id> --yes`，再 `docshub push <file> --yes`。

## 批量发布

```bash
# 发布目录下所有 .md 文件
for f in /path/to/articles/*.md; do
  docshub push "$f" --yes
done

# 带 AI 分类批量发布
for f in /path/to/articles/*.md; do
  docshub push "$f" --classify --yes
done
```

## 管理命令

```bash
docshub list                        # 列出所有文章
docshub list --category AI          # 按分类过滤
docshub list --tag llm              # 按标签过滤
docshub search "关键词"              # 全文搜索（标题+摘要）
docshub delete <article-id> --yes   # 删除文章
```

## 硬性规则

- category 必须是预设值之一：AI / UE / DevOps / Research / Note / Other
- 中文标题的 slug 会回退为 `article-{timestamp}`，建议 frontmatter 中用英文 title
- HTML 文章通过 `/html/{category}/{slug}` 访问，Markdown 通过 `/#/articles/{category}/{slug}` 访问

## 常见问题

| 问题 | 解决方案 |
|------|---------|
| Docsify 页面空白 | CDN 需要外网，LAN 无网时需 vendor 本地资源 |
| AI 分类不准 | 用 `--category` 覆盖，或 `--classify-json` 完全控制 |
| 修改已发布文章 | delete + 重新 push，或通过 API version_of |
| slug 是 article-timestamp | 中文标题无 ASCII slug，改用英文 title |

## 验证清单

- [ ] 服务器运行中（`curl http://localhost:8080` 返回 200）
- [ ] `~/.docshub.json` 中 server_url 和 author 已配置
- [ ] push 输出显示 Published + URL
- [ ] 浏览器访问 URL 确认文章可读
