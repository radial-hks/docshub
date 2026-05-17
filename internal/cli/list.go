package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/radial-hks/docshub/internal/model"
)

type ListOptions struct {
	Category string
	Tag      string
	Author   string
}

// RunList implements `docshub list [flags]`.
func RunList(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts ListOptions
	fs.StringVar(&opts.Category, "category", "", "filter by category")
	fs.StringVar(&opts.Tag, "tag", "", "filter by tag")
	fs.StringVar(&opts.Author, "author", "", "filter by author")
	// Reorder so flags come before positional args.
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	q := url.Values{}
	if opts.Category != "" {
		q.Set("category", opts.Category)
	}
	if opts.Tag != "" {
		q.Set("tag", opts.Tag)
	}
	if opts.Author != "" {
		q.Set("author", opts.Author)
	}
	path := "/api/articles"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

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

// formatList writes a tab-aligned table of articles to w.
func formatList(w io.Writer, articles []model.Article) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tCATEGORY\tTAGS\tAUTHOR\tDATE")
	for _, a := range articles {
		tags := strings.Join(a.Tags, ",")
		if tags == "" {
			tags = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID,
			displayOrDash(a.Title),
			displayOrDash(a.Category),
			tags,
			displayOrDash(a.Author),
			a.Date.Format("2006-01-02"),
		)
	}
	tw.Flush()
}
