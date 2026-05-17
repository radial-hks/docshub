# DocsHub 文章发布工作流

> 完整 Skill 定义见 Hermes skill: `docshub-publish`
> 本文档作为仓库内的参考文档，描述 CLI 命令和发布流程细节。

---

## 快速开始

```bash
# 发布 Markdown
docshub push article.md

# 带 AI 自动分类
docshub push article.md --classify

# 指定分类和标签
docshub push article.md --category AI --tags llm,rag --yes

# 发布 HTML
docshub push page.html
```

## 前提条件

1. 服务器运行：`docshub serve`（默认 `:8080`）
2. CLI 配置完成：`docshub init`（保存到 `~/.docshub.json`）

## Frontmatter 格式

```markdown
---
title: 文章标题
category: AI
tags: [tag1, tag2]
author: radial
---

正文内容...
```

字段均为可选，可被 CLI 参数覆盖。

## 元数据优先级

```
--classify-json  >  --classify (AI)  >  CLI flags  >  frontmatter  >  defaults
```

## AI 自动分类

配置 `~/.docshub.json`：

```json
{
  "classify_url": "http://localhost:11434/v1/chat/completions",
  "classify_model": "qwen2.5:7b"
}
```

- `--classify` 将前 3000 字符发送给 LLM，获取 title/category/tags 建议
- LLM 不可用时静默降级
- 预设分类：AI、UE、DevOps、Research、Note、Other

## HTML 文章

| 格式 | 渲染方式 | 访问路径 |
|------|---------|---------|
| Markdown | Docsify 运行时渲染 | `/#/articles/{category}/{slug}` |
| HTML | 浏览器原生渲染 | `/html/{category}/{slug}` |

## 版本管理

通过 API `version_of` 字段创建新版本，旧版归档到 `.versions/`：

```bash
curl -X POST http://localhost:8080/api/articles \
  -H 'Content-Type: application/json' \
  -d '{"title":"...","content":"...","version_of":"<article-id>","format":"md"}'
```

也可简化为 delete + 重新 push。

## 管理命令

```bash
docshub list --category AI     # 按分类
docshub list --tag llm         # 按标签
docshub search "关键词"         # 全文搜索
docshub delete <id> --yes      # 删除
```
