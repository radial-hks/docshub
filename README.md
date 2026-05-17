# Docshub

LAN document sharing hub with Docsify frontend.

## Quick Start

### Server

```bash
go build -o docshub-server ./cmd/server
./docshub-server
```

### CLI

```bash
go build -o docshub ./cmd/cli
docshub push article.md
```

## Architecture

- **Server**: Go HTTP server serving Docsify frontend + article CRUD API
- **CLI**: Client tool for publishing articles with AI-assisted classification
- **Frontend**: Docsify for reading/browsing/searching articles
