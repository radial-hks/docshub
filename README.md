# Docshub

LAN document sharing hub: publish Markdown articles via CLI to a central Go server, read them through a Docsify frontend.

## Architecture

```
+--------+    HTTP/JSON    +------------+    static files    +---------+
|  CLI   | --------------> |   Server   | -----------------> | Docsify |
| (push) |                 | (Go HTTP)  |                    | (browser)|
+--------+                 +------------+                    +---------+
                                 |
                                 v
                          web/articles/*.md
                          web/_sidebar.md
                          index.json
```

- **CLI** (`docshub`) reads a local Markdown file, parses frontmatter, and POSTs it to the server.
- **Server** (`docshub-server`) stores articles on disk, maintains `index.json`, regenerates `_sidebar.md`, and serves the Docsify frontend.
- **Docsify** renders `web/` as a documentation site with sidebar navigation and full-text search.

## Quick Start

### Prerequisites

- Go 1.22+

### Build

```bash
make build
```

This produces two binaries at the project root: `docshub-server` and `docshub`.

### Start the server

```bash
./docshub-server
```

Defaults: listens on `:8080`, data directory `./web`. Override with `DOCSHUB_PORT` and `DOCSHUB_DATA`.

Open `http://localhost:8080` in a browser to see the Docsify frontend.

### Configure the CLI

```bash
./docshub init
```

Prompts for server URL (e.g. `http://localhost:8080`) and author name, saves to `~/.docshub.json`.

### Publish an article

```bash
./docshub push article.md
```

## CLI Commands

### `docshub init`

Interactive setup. Writes `~/.docshub.json`.

### `docshub push <file> [flags]`

Publish a Markdown file to the server.

Flags:
- `--category CAT` — set the article category
- `--tags t1,t2` — comma-separated tags
- `--yes` — skip the confirmation prompt
- `--classify JSON` — supply pre-computed classification (e.g. `{"category":"AI","tags":["llm"]}`)

Values from frontmatter, flags, and `--classify` are merged; flags take precedence.

### `docshub list [flags]`

List articles known to the server.

Flags:
- `--category CAT` — filter by category
- `--tag TAG` — filter by tag
- `--author AUTHOR` — filter by author

## API Endpoints

| Method | Path                  | Description                                                                |
|--------|-----------------------|----------------------------------------------------------------------------|
| POST   | `/api/articles`       | Create/publish an article. Body: `PublishRequest` JSON.                    |
| GET    | `/api/articles`       | List articles. Query params: `category`, `tag`, `author`, `q` (search).    |
| GET    | `/api/articles/{id}`  | Get a single article's metadata.                                           |
| DELETE | `/api/articles/{id}`  | Delete an article.                                                         |

Static files under `web/` (including `index.html`, `_sidebar.md`, and `articles/`) are served from `/`.

## Configuration

CLI config lives at `~/.docshub.json`:

```json
{
  "server_url": "http://localhost:8080",
  "author": "radial"
}
```

Server environment variables:

- `DOCSHUB_PORT` — listen port (default `8080`)
- `DOCSHUB_DATA` — data directory (default `./web`)

## Frontmatter

Markdown files may begin with a YAML frontmatter block. Recognised fields:

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

## Versioning

Re-publishing an article (by passing `version_of` in the API request, or pushing with the same target ID) archives the previous file under `web/articles/<category>/.versions/<slug>/v<N>-<date>.md` and writes the new content in place. Version history is tracked in `meta.json` next to the archived files, and the article's `version` field is incremented.

## Development

### Run tests

```bash
make test
```

Or directly:

```bash
go test ./... -v
```

### Build

```bash
make build       # both binaries
make server      # server only
make cli         # CLI only
make clean       # remove built binaries
```

### Project structure

```
cmd/
  server/         # docshub-server entry point
  cli/            # docshub entry point
internal/
  model/          # shared types (Article, PublishRequest, ...)
  server/         # store, sidebar, HTTP handlers
  cli/            # config, push, list
test/             # integration tests
web/              # Docsify frontend, served as static files
  index.html
  articles/       # generated at runtime
  _sidebar.md     # generated at runtime
docs/             # design notes and plans
```
