package main

import (
	"fmt"
	"os"

	"github.com/radial-hks/docshub/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = cli.RunInit(os.Stdin, os.Stdout)
	case "push":
		err = cli.RunPush(args, os.Stdin, os.Stdout, os.Stderr)
	case "list":
		err = cli.RunList(args, os.Stdout, os.Stderr)
	case "serve":
		err = cli.RunServe()
	case "delete":
		err = cli.RunDelete(args, os.Stdin, os.Stdout, os.Stderr)
	case "search":
		err = cli.RunSearch(args, os.Stdout, os.Stderr)
	case "-h", "--help", "help":
		usage(os.Stdout)
		return
	default:
		fmt.Fprintf(os.Stderr, "docshub: unknown command %q\n", cmd)
		usage(os.Stderr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, "Usage: docshub <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init                       Configure server URL, author, and AI classification")
	fmt.Fprintln(w, "  push <file> [flags]        Publish a Markdown article")
	fmt.Fprintln(w, "  list [flags]               List published articles")
	fmt.Fprintln(w, "  serve                      Start the docshub server")
	fmt.Fprintln(w, "  delete <id> [--yes]         Delete an article")
	fmt.Fprintln(w, "  search <query>              Search articles")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Push flags:")
	fmt.Fprintln(w, "  --classify                 Use AI to suggest title/category/tags via local LLM")
	fmt.Fprintln(w, "  --classify-json <json>     Override metadata with raw JSON {title,category,tags,author}")
	fmt.Fprintln(w, "  --category <cat>           Set article category")
	fmt.Fprintln(w, "  --tags <tags>              Set comma-separated tags")
	fmt.Fprintln(w, "  --format <fmt>             Article format: html or md (auto-detected by default)")
	fmt.Fprintln(w, "  --yes                      Skip confirmation prompt")
}
