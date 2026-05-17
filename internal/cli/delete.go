package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/radial-hks/docshub/internal/model"
)

// DeleteOptions captures flag inputs for the delete command.
type DeleteOptions struct {
	Yes bool
}

// RunDelete implements `docshub delete <id> [flags]`.
func RunDelete(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts DeleteOptions
	fs.BoolVar(&opts.Yes, "yes", false, "skip confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: docshub delete <id> [--yes]")
	}
	id := rest[0]

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client := newClient(cfg.ServerURL)

	// Without --yes, fetch the article first so the user can confirm.
	if !opts.Yes {
		data, status, err := client.doGet("/api/articles/" + url.PathEscape(id))
		if err != nil {
			return fmt.Errorf("request server: %w", err)
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("server: %s", errorBody(data, status))
		}
		var article model.Article
		if err := json.Unmarshal(data, &article); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		fmt.Fprintf(stdout, "About to delete:\n")
		fmt.Fprintf(stdout, "  ID:       %s\n", article.ID)
		fmt.Fprintf(stdout, "  Title:    %s\n", displayOrDash(article.Title))
		fmt.Fprintf(stdout, "  Category: %s\n", displayOrDash(article.Category))
		fmt.Fprintf(stdout, "  Author:   %s\n", displayOrDash(article.Author))
		fmt.Fprintf(stdout, "  Date:     %s\n", article.Date.Format("2006-01-02"))

		ok, err := deleteConfirm(stdin, stdout)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	data, status, err := client.doDelete("/api/articles/" + url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("request server: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("server: %s", errorBody(data, status))
	}

	fmt.Fprintf(stdout, "Deleted: %s\n", id)
	return nil
}

// deleteConfirm prompts the user to confirm deletion.
func deleteConfirm(stdin io.Reader, stdout io.Writer) (bool, error) {
	fmt.Fprint(stdout, "Proceed? [y/N]: ")
	sc := bufio.NewScanner(stdin)
	if !sc.Scan() {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
