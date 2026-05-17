# Docshub

LAN document sharing hub — publish Markdown/HTML articles via CLI to a central Go server, read them through a Docsify frontend with AI-powered classification.

English | [中文](./README_zh.md)

---

## Features

- Single binary, zero-dependency deployment
- Markdown + HTML dual format support
- AI auto-classification (local LLM suggests title/category/tags)
- Frontmatter metadata parsing
- Article versioning
- Docsify frontend with full-text search
- LAN-only, no authentication needed

## Architecture

```
┌──────────┐   HTTP/JSON   ┌────────────┐   static files   ┌─────────┐
│   CLI    │ ────────────> │   Server   │ ───────────────> │ Docsify │
│ (push等) │               │ (Go HTTP)  │                   │(browser)│
└──────────┘               └────────────┘                   └─────────┘
                                  │
                                  ▼
                           web/articles/*.md
                           web/articles/*.html
                           web/_sidebar.md
                           index.json
```

- **CLI** (`docshub`) reads a local file, parses frontmatter, and POSTs it to the server.
- **Server** stores articles on disk, maintains `index.json`, regenerates `_sidebar.md`, and serves the Docsify frontend.
- **Docsify** renders `web/` as a documentation site with sidebar navigation and full-text search.

## Quick Start

### Install

Download the binary for your platform from [Releases](https://github.com/radial-hks/docshub/releases), or build from source:

```bash
# Go 1.22+ required
go build -o docshub ./cmd/docshub
```

### Start the server

```bash
./docshub serve
```

Defaults: listens on `:8080`, data directory `./web`. Override with environment variables:

```bash
DOCSHUB_PORT=9090 DOCSHUB_DATA=/data/docs ./docshub serve
```

Open `http://localhost:8080` in a browser to see the Docsify frontend.

### Configure the CLI

```bash
./docshub init
```

Prompts for server URL, author name, and AI classification settings. Saves to `~/.docshub.json`.

### Publish an article

```bash
# Publish a Markdown article
./docshub push article.md

# Publish an HTML article (auto-detected from extension)
./docshub push page.html

# Specify category and tags
./docshub push article.md --category AI --tags llm,rag

# Use AI auto-classification
./docshub push article.md --classify

# Skip confirmation and publish directly
./docshub push article.md --yes
```

## CLI Commands

### `docshub init`

Interactive setup. Writes `~/.docshub.json`.

### `docshub push <file> [flags]`

Publish an article to the server.

| Flag | Description |
|------|-------------|
| `--category <cat>` | Set the article category |
| `--tags <tags>` | Comma-separated tags |
| `--format <fmt>` | Article format: `html` or `md` (auto-detected from file extension by default) |
| `--classify` | Call a local LLM to suggest title/category/tags |
| `--classify-json <json>` | Supply classification JSON directly, e.g. `{"category":"AI","tags":["llm"]}` |
| `--yes` | Skip the confirmation prompt |

Metadata priority (highest to lowest):

`--classify-json` > `--classify` (AI result) > CLI flags > frontmatter > defaults

### `docshub list [flags]`

List published articles.

| Flag | Description |
|------|-------------|
| `--category <cat>` | Filter by category |
| `--tag <tag>` | Filter by tag |
| `--author <author>` | Filter by author |

### `docshub search <query>`

Full-text search across article titles and summaries.

### `docshub delete <id> [--yes]`

Delete an article. Confirmation required unless `--yes` is passed.

### `docshub serve`

Start the DocsHub server.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/articles` | Create/publish an article. Body: `PublishRequest` JSON. |
| GET | `/api/articles` | List articles. Query params: `category`, `tag`, `author`, `q` (search). |
| GET | `/api/articles/{id}` | Get a single article's metadata. |
| DELETE | `/api/articles/{id}` | Delete an article. |
| GET | `/html/{category}/{slug}` | Serve HTML article with browser-native rendering. |

Static files under `web/` (including `index.html`, `_sidebar.md`, and `articles/`) are served from `/`.

## Configuration

### CLI config (`~/.docshub.json`)

```json
{
  "server_url": "http://localhost:8080",
  "author": "radial",
  "classify_url": "http://localhost:11434/v1/chat/completions",
  "classify_model": "qwen2.5:7b"
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `server_url` | Server URL | `http://localhost:8080` |
| `author` | Default author name | empty |
| `classify_url` | AI classification API URL (OpenAI-compatible) | empty (disabled) |
| `classify_model` | Model to use for classification | `qwen2.5:7b` |

### Server environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DOCSHUB_PORT` | Listen port | `8080` |
| `DOCSHUB_DATA` | Data directory | `./web` |

## Frontmatter

Markdown files may begin with a YAML frontmatter block:

```markdown
---
title: Article Title
category: AI
tags: [llm, rag, workflow]
author: radial
---

# Article body starts here
```

Any field can be overridden by CLI flags.

## HTML Articles

DocsHub supports HTML articles alongside Markdown:

- Files with `.html`/`.htm` extension are auto-detected as HTML format
- HTML articles are served via the `/html/{category}/{slug}` route with `Content-Type: text/html` for browser-native rendering
- Use `--format html` to manually specify format
- Markdown articles continue to be rendered by Docsify; HTML articles are rendered directly by the browser

## AI Auto-Classification

Once `classify_url` is configured, use the `--classify` flag when pushing:

```bash
# Configure AI classification first (via init or editing ~/.docshub.json)
# classify_url should point to Ollama or any OpenAI-compatible API

./docshub push article.md --classify
```

Flow:
1. Article content (first 3000 chars) is sent to the LLM
2. LLM returns suggested title, category, and tags
3. AI suggestion is displayed for user confirmation
4. Article is published after confirmation

If the LLM is unavailable, the flow falls back to manual metadata — no blocking.

## Versioning

Re-publishing an article (by passing `version_of` in the API request) archives the previous file under `web/articles/<category>/.versions/<slug>/v<N>-<date>.md` and writes the new content in place. Version history is tracked in `meta.json` next to the archived files, and the article's `version` field is incremented.

## Development

### Run tests

```bash
make test
# or
go test ./...
```

### Build

```bash
make build          # build single binary
make dist           # cross-compile all platforms + generate SHA256 checksums
make clean          # remove build artifacts
```

### Project structure

```
cmd/
  docshub/              # single entry point
internal/
  model/                # shared types (Article, PublishRequest, ...)
  server/               # store, sidebar, HTTP handlers
  cli/                  # config, push, list, delete, search, classify, serve
test/                   # integration tests
web/                    # Docsify frontend, served as static files
  index.html
  articles/             # generated at runtime
  _sidebar.md           # generated at runtime
```

## License

MIT
