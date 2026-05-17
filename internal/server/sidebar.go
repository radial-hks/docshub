package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/radial-hks/docshub/internal/model"
)

const (
	sidebarFileName          = "_sidebar.md"
	unclassifiedSidebarLabel = "Unclassified"
)

func GenerateSidebar(articles []model.Article) string {
	var b strings.Builder
	b.WriteString("- [Home](/)\n")

	groups := make(map[string][]model.Article)
	for _, a := range articles {
		cat := a.Category
		if cat == "" {
			cat = unclassifiedSidebarLabel
		}
		groups[cat] = append(groups[cat], a)
	}

	categories := make([]string, 0, len(groups))
	for cat := range groups {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, cat := range categories {
		fmt.Fprintf(&b, "- %s\n", cat)
		arts := groups[cat]
		sort.SliceStable(arts, func(i, j int) bool {
			return arts[i].Date.After(arts[j].Date)
		})
		for _, a := range arts {
			fmt.Fprintf(&b, "  - [%s](%s)\n", a.Title, a.File)
		}
	}

	return b.String()
}

func (s *Store) refreshSidebar() error {
	content := GenerateSidebar(s.articles)
	path := filepath.Join(s.dataDir, sidebarFileName)
	return os.WriteFile(path, []byte(content), 0o644)
}
