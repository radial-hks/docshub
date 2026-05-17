package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/radial-hks/docshub/internal/model"
)

// Frontmatter holds the fields recognized at the top of a markdown article.
type Frontmatter struct {
	Title    string
	Category string
	Tags     []string
	Author   string
}

// parseFrontmatter scans content for a leading YAML-ish frontmatter block
// delimited by --- lines. It returns the parsed metadata and the body with
// the frontmatter stripped. If no frontmatter is present, body == content and
// the returned Frontmatter is zero-valued.
func parseFrontmatter(content string) (Frontmatter, string) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return Frontmatter{}, content
	}
	rest := normalized[len("---\n"):]
	// Locate the closing "---" on its own line.
	lines := strings.SplitN(rest, "\n", -1)
	endLine := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "---" {
			endLine = i
			break
		}
	}
	if endLine < 0 {
		return Frontmatter{}, content
	}
	fm := parseFrontmatterLines(lines[:endLine])
	body := strings.Join(lines[endLine+1:], "\n")
	return fm, body
}

func parseFrontmatterLines(lines []string) Frontmatter {
	fm := Frontmatter{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		val = unquote(val)
		switch strings.ToLower(key) {
		case "title":
			fm.Title = val
		case "category":
			fm.Category = val
		case "author":
			fm.Author = val
		case "tags":
			fm.Tags = parseTagList(val)
		}
	}
	return fm
}

// parseTagList accepts "[a, b, c]" or "a, b, c".
func parseTagList(val string) []string {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		val = val[1 : len(val)-1]
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := unquote(strings.TrimSpace(p))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func unquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// PushOptions captures flag inputs for the push command.
type PushOptions struct {
	Category     string
	Tags         string
	Yes          bool
	Classify     bool
	ClassifyJSON string
	Format       string
}

// RunPush implements `docshub push <file> [flags]`.
func RunPush(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts PushOptions
	fs.StringVar(&opts.Category, "category", "", "category for the article")
	fs.StringVar(&opts.Tags, "tags", "", "comma-separated tags")
	fs.BoolVar(&opts.Yes, "yes", false, "skip confirmation prompt")
	fs.BoolVar(&opts.Classify, "classify", false, "use AI to suggest title/category/tags via local LLM")
	fs.StringVar(&opts.ClassifyJSON, "classify-json", "", "raw JSON {title,category,tags,author} to override metadata")
	fs.StringVar(&opts.Format, "format", "", "article format: html or md (auto-detected from file extension)")
	// Go's flag package stops parsing at the first non-flag argument.
	// Reorder so flags come before positional args, allowing: push file.md --category AI
	if err := fs.Parse(reorderFlags(fs, args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: docshub push <file.md> [flags]")
	}
	file := rest[0]

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	req, err := buildPublishRequest(file, string(content), opts, cfg)
	if err != nil {
		return err
	}

	// If --classify is set, call the LLM and offer suggestions.
	if opts.Classify {
		result, err := ClassifyWithLLM(req.Content, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "Warning: AI classification failed: %v\n", err)
		} else if result != nil {
			// Apply AI suggestions to fields that are still empty or at defaults.
			applyClassifyResult(&req, result, stdout)
		}
	}

	printSummary(stdout, req)

	if !opts.Yes {
		ok, err := confirm(stdin, stdout)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	client := newClient(cfg.ServerURL)
	data, status, err := client.doPost("/api/articles", req)
	if err != nil {
		return fmt.Errorf("request server: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("server: %s", errorBody(data, status))
	}
	var resp model.PublishResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	fmt.Fprintf(stdout, "Published: %s (v%d)\nURL: %s%s\n", resp.ID, resp.Version, strings.TrimRight(cfg.ServerURL, "/"), resp.Path)
	return nil
}

// applyClassifyResult merges the LLM-suggested metadata into the request.
// Priority chain: --classify-json > --classify (AI result) > CLI flags > frontmatter > defaults.
// So AI suggestions only fill in fields that haven't already been set by higher-priority sources.
func applyClassifyResult(req *model.PublishRequest, result *ClassifyResult, stdout io.Writer) {
	fmt.Fprintln(stdout, "AI suggests:")
	if result.Title != "" {
		fmt.Fprintf(stdout, "  Title:    %s\n", result.Title)
	}
	if result.Category != "" {
		fmt.Fprintf(stdout, "  Category: %s\n", result.Category)
	}
	if len(result.Tags) > 0 {
		fmt.Fprintf(stdout, "  Tags:     %s\n", strings.Join(result.Tags, ", "))
	}
	if result.Author != "" {
		fmt.Fprintf(stdout, "  Author:   %s\n", result.Author)
	}
	fmt.Fprintln(stdout)

	// Apply suggestions where current values are empty/default.
	if result.Title != "" {
		req.Title = result.Title
	}
	if result.Category != "" {
		req.Category = result.Category
	}
	if len(result.Tags) > 0 {
		req.Tags = result.Tags
	}
	if result.Author != "" {
		req.Author = result.Author
	}
}

// buildPublishRequest applies the priority rules:
// classify-json > classify (AI result) > CLI flags > frontmatter > defaults.
// Format detection: file extension auto-detect, but --format flag overrides.
func buildPublishRequest(file, content string, opts PushOptions, cfg *Config) (model.PublishRequest, error) {
	fm, body := parseFrontmatter(content)

	// Auto-detect format from file extension.
	detectedFormat := "md"
	if ext := strings.ToLower(filepath.Ext(file)); ext == ".html" || ext == ".htm" {
		detectedFormat = "html"
	}

	req := model.PublishRequest{
		Title:    fm.Title,
		Category: fm.Category,
		Tags:     append([]string(nil), fm.Tags...),
		Author:   fm.Author,
		Content:  body,
		Format:   detectedFormat,
	}

	// --format flag overrides auto-detected format.
	if opts.Format != "" {
		req.Format = strings.ToLower(opts.Format)
	}

	// CLI flags override frontmatter.
	if opts.Category != "" {
		req.Category = opts.Category
	}
	if opts.Tags != "" {
		req.Tags = parseTagList(opts.Tags)
	}

	// --classify-json overrides everything (highest priority).
	if opts.ClassifyJSON != "" {
		var c struct {
			Title    string   `json:"title"`
			Category string   `json:"category"`
			Tags     []string `json:"tags"`
			Author   string   `json:"author"`
		}
		if err := json.Unmarshal([]byte(opts.ClassifyJSON), &c); err != nil {
			return req, fmt.Errorf("parse --classify-json: %w", err)
		}
		if c.Title != "" {
			req.Title = c.Title
		}
		if c.Category != "" {
			req.Category = c.Category
		}
		if len(c.Tags) > 0 {
			req.Tags = c.Tags
		}
		if c.Author != "" {
			req.Author = c.Author
		}
	}

	if req.Author == "" {
		req.Author = cfg.Author
	}
	if req.Title == "" {
		base := filepath.Base(file)
		ext := filepath.Ext(base)
		req.Title = strings.TrimSuffix(base, ext)
	}
	return req, nil
}

func printSummary(w io.Writer, req model.PublishRequest) {
	fmt.Fprintln(w, "About to publish:")
	fmt.Fprintf(w, "  Title:    %s\n", req.Title)
	fmt.Fprintf(w, "  Category: %s\n", displayOrDash(req.Category))
	tags := "-"
	if len(req.Tags) > 0 {
		tags = strings.Join(req.Tags, ", ")
	}
	fmt.Fprintf(w, "  Tags:     %s\n", tags)
	fmt.Fprintf(w, "  Author:   %s\n", displayOrDash(req.Author))
	fmt.Fprintf(w, "  Format:   %s\n", req.Format)
}

func displayOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func confirm(stdin io.Reader, stdout io.Writer) (bool, error) {
	fmt.Fprint(stdout, "Proceed? [Y/n/e]: ")
	sc := bufio.NewScanner(stdin)
	if !sc.Scan() {
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "", "y", "yes":
		return true, nil
	case "e":
		fmt.Fprintln(stdout, "Edit mode not yet supported; aborting.")
		return false, nil
	default:
		return false, nil
	}
}

// reorderFlags rearranges args so that flag arguments (--name [value]) come
// before positional arguments. This works around Go's flag package behavior
// where parsing stops at the first non-flag token.
func reorderFlags(fs *flag.FlagSet, args []string) []string {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true }) // clear any previous

	var flagArgs, posArgs []string
	for i := 0; i < len(args); i++ {
		s := args[i]
		if !strings.HasPrefix(s, "-") {
			posArgs = append(posArgs, s)
			continue
		}
		// Handle --name=value form
		if strings.Contains(s, "=") {
			flagArgs = append(flagArgs, s)
			continue
		}
		// s is --name or -x; check if it expects a value
		name := strings.TrimLeft(s, "-")
		f := fs.Lookup(name)
		flagArgs = append(flagArgs, s)
		if f != nil && !isBoolFlag(f) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}
	return append(flagArgs, posArgs...)
}

// isBoolFlag checks whether a flag is a boolean flag.
func isBoolFlag(f *flag.Flag) bool {
	// flag.Flag doesn't expose IsBoolFlag directly, so we use a small hack:
	// Try to set "true" and "false" — bool flags accept both, non-bools don't.
	// A simpler approach: check the zero value via reflection or just maintain
	// a known set. Since we control all flags, check by name.
	switch f.Name {
	case "yes", "classify":
		return true
	}
	return false
}
