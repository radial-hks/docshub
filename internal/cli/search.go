package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"

	"github.com/radial-hks/docshub/internal/model"
)

// RunSearch implements `docshub search <query>`.
func RunSearch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// Reorder so flags come before positional args.
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: docshub search <query>")
	}
	query := rest[0]

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	path := "/api/articles?q=" + url.QueryEscape(query)

	client := newClient(cfg.ServerURL)
	data, status, err := client.doGet(path)
	if err != nil {
		return fmt.Errorf("request server: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("server: %s", errorBody(data, status))
	}
	var articles []model.Article
	if err := json.Unmarshal(data, &articles); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	formatList(stdout, articles)
	return nil
}
