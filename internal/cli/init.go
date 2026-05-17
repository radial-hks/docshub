package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// RunInit implements `docshub init`, prompting interactively for the
// server URL and author, and writing the result to ~/.docshub.json.
// ClassifyURL and ClassifyModel are only needed for --classify (legacy)
// and can be set by editing ~/.docshub.json directly.
func RunInit(stdin io.Reader, stdout io.Writer) error {
	existing, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	sc := bufio.NewScanner(stdin)

	serverURL := prompt(sc, stdout, "Server URL", existing.ServerURL)
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	author := prompt(sc, stdout, "Author", existing.Author)

	cfg := &Config{
		ServerURL:     serverURL,
		Author:        author,
		ClassifyURL:   existing.ClassifyURL,
		ClassifyModel: existing.ClassifyModel,
	}
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintln(stdout, "Saved configuration to ~/.docshub.json")
	return nil
}

func prompt(sc *bufio.Scanner, stdout io.Writer, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(stdout, "%s [%s]: ", label, defaultVal)
	} else {
		fmt.Fprintf(stdout, "%s: ", label)
	}
	if !sc.Scan() {
		return defaultVal
	}
	line := strings.TrimSpace(sc.Text())
	if line == "" {
		return defaultVal
	}
	return line
}
