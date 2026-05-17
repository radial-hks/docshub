package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/radial-hks/docshub/internal/model"
)

const (
	indexFileName    = "index.json"
	articlesDirName  = "articles"
	versionsDirName  = ".versions"
	unclassifiedDir  = "_unclassified"
	summaryMaxLength = 200
)

type Store struct {
	dataDir  string
	mu       sync.RWMutex
	articles []model.Article
}

type ListFilter struct {
	Category string
	Tag      string
	Author   string
	Query    string
}

type versionMeta struct {
	Versions []versionEntry `json:"versions"`
}

type versionEntry struct {
	Version int       `json:"version"`
	Date    time.Time `json:"date"`
	File    string    `json:"file"`
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, articlesDirName), 0o755); err != nil {
		return nil, fmt.Errorf("create articles dir: %w", err)
	}
	s := &Store{dataDir: dataDir, articles: []model.Article{}}
	if err := s.loadIndex(); err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	return s, nil
}

func (s *Store) loadIndex() error {
	path := filepath.Join(s.dataDir, indexFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.articles = []model.Article{}
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		s.articles = []model.Article{}
		return nil
	}
	return json.Unmarshal(data, &s.articles)
}

func (s *Store) saveIndex() error {
	path := filepath.Join(s.dataDir, indexFileName)
	data, err := json.MarshalIndent(s.articles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) Create(req model.PublishRequest) (*model.PublishResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var (
		article          model.Article
		previousVersions []string
	)

	if req.VersionOf != "" {
		oldIdx := -1
		for i, a := range s.articles {
			if a.ID == req.VersionOf {
				oldIdx = i
				break
			}
		}
		if oldIdx == -1 {
			return nil, fmt.Errorf("article %q not found for versioning", req.VersionOf)
		}
		old := s.articles[oldIdx]

		oldCategoryDir := old.Category
		if oldCategoryDir == "" {
			oldCategoryDir = unclassifiedDir
		}
		oldSlug := s.slugify(old.Title)
		oldFilePath := filepath.Join(s.dataDir, articlesDirName, oldCategoryDir, oldSlug+".md")
		versionDir := filepath.Join(s.dataDir, articlesDirName, oldCategoryDir, versionsDirName, oldSlug)
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			return nil, fmt.Errorf("create versions dir: %w", err)
		}
		archivedName := fmt.Sprintf("v%d-%s.md", old.Version, old.Date.Format("20060102"))
		archivedPath := filepath.Join(versionDir, archivedName)
		if err := os.Rename(oldFilePath, archivedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("archive old version: %w", err)
		}

		metaPath := filepath.Join(versionDir, "meta.json")
		var meta versionMeta
		if data, err := os.ReadFile(metaPath); err == nil {
			_ = json.Unmarshal(data, &meta)
		}
		meta.Versions = append(meta.Versions, versionEntry{
			Version: old.Version,
			Date:    old.Date,
			File:    archivedName,
		})
		metaData, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal version meta: %w", err)
		}
		if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
			return nil, fmt.Errorf("write version meta: %w", err)
		}

		for _, v := range meta.Versions {
			rel := filepath.ToSlash(filepath.Join(articlesDirName, oldCategoryDir, versionsDirName, oldSlug, v.File))
			previousVersions = append(previousVersions, "/"+rel)
		}

		article = old
		article.Version = old.Version + 1
		article.Title = req.Title
		article.Category = req.Category
		article.Tags = req.Tags
		article.Author = req.Author
		article.Date = now

		s.articles = append(s.articles[:oldIdx], s.articles[oldIdx+1:]...)
	} else {
		article = model.Article{
			ID:       s.generateID(req.Title),
			Title:    req.Title,
			Category: req.Category,
			Tags:     req.Tags,
			Author:   req.Author,
			Date:     now,
			Version:  1,
		}
	}

	categoryDir := article.Category
	if categoryDir == "" {
		categoryDir = unclassifiedDir
	}
	slug := s.slugify(article.Title)
	relPath := filepath.Join(articlesDirName, categoryDir, slug+".md")
	fullPath := filepath.Join(s.dataDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, fmt.Errorf("create category dir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(req.Content), 0o644); err != nil {
		return nil, fmt.Errorf("write article file: %w", err)
	}

	article.File = "/" + filepath.ToSlash(relPath)
	article.Summary = s.extractSummary(req.Content)

	s.articles = append(s.articles, article)
	if err := s.saveIndex(); err != nil {
		return nil, fmt.Errorf("save index: %w", err)
	}

	return &model.PublishResponse{
		ID:               article.ID,
		Status:           "ok",
		Path:             article.File,
		Version:          article.Version,
		PreviousVersions: previousVersions,
	}, nil
}

func (s *Store) Get(id string) (*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.articles {
		if s.articles[i].ID == id {
			a := s.articles[i]
			return &a, nil
		}
	}
	return nil, fmt.Errorf("article %q not found", id)
}

func (s *Store) List(filter ListFilter) ([]model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Article, 0, len(s.articles))
	q := strings.ToLower(filter.Query)
	for _, a := range s.articles {
		if filter.Category != "" && a.Category != filter.Category {
			continue
		}
		if filter.Author != "" && a.Author != filter.Author {
			continue
		}
		if filter.Tag != "" {
			found := false
			for _, t := range a.Tags {
				if t == filter.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(a.Title), q) &&
			!strings.Contains(strings.ToLower(a.Summary), q) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, a := range s.articles {
		if a.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("article %q not found", id)
	}
	article := s.articles[idx]
	categoryDir := article.Category
	if categoryDir == "" {
		categoryDir = unclassifiedDir
	}
	slug := s.slugify(article.Title)
	fullPath := filepath.Join(s.dataDir, articlesDirName, categoryDir, slug+".md")
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove article file: %w", err)
	}
	s.articles = append(s.articles[:idx], s.articles[idx+1:]...)
	return s.saveIndex()
}

var slugReplaceRE = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Store) slugify(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	slug := slugReplaceRE.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "article"
	}
	return slug
}

func (s *Store) generateID(title string) string {
	return fmt.Sprintf("%s-%s", s.slugify(title), time.Now().Format("20060102"))
}

var (
	mdFenceRE    = regexp.MustCompile("(?s)```[^\n]*\n.*?```")
	mdImageRE    = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	mdLinkRE     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdHeadingRE  = regexp.MustCompile(`(?m)^#+\s*`)
	mdEmphasisRE = regexp.MustCompile(`[*_]{1,3}([^*_\n]+)[*_]{1,3}`)
	mdCodeRE     = regexp.MustCompile("`+([^`]*)`+")
	mdQuoteRE    = regexp.MustCompile(`(?m)^>\s*`)
	mdListRE     = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	whitespaceRE = regexp.MustCompile(`\s+`)
)

func (s *Store) extractSummary(content string) string {
	c := content
	c = mdFenceRE.ReplaceAllString(c, "")
	c = mdImageRE.ReplaceAllString(c, "")
	c = mdLinkRE.ReplaceAllString(c, "$1")
	c = mdHeadingRE.ReplaceAllString(c, "")
	c = mdEmphasisRE.ReplaceAllString(c, "$1")
	c = mdCodeRE.ReplaceAllString(c, "$1")
	c = mdQuoteRE.ReplaceAllString(c, "")
	c = mdListRE.ReplaceAllString(c, "")
	c = whitespaceRE.ReplaceAllString(c, " ")
	c = strings.TrimSpace(c)
	runes := []rune(c)
	if len(runes) > summaryMaxLength {
		return string(runes[:summaryMaxLength])
	}
	return c
}
