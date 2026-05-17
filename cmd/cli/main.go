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
	fmt.Fprintln(w, "  init                       Configure server URL and author")
	fmt.Fprintln(w, "  push <file> [flags]        Publish a Markdown article")
	fmt.Fprintln(w, "  list [flags]               List published articles")
}
