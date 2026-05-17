package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/radial-hks/docshub/internal/model"
)

func (s *Store) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/articles", s.handleCreateArticle)
	mux.HandleFunc("GET /api/articles", s.handleListArticles)
	mux.HandleFunc("GET /api/articles/{id}", s.handleGetArticle)
	mux.HandleFunc("DELETE /api/articles/{id}", s.handleDeleteArticle)
	mux.HandleFunc("GET /html/", s.handleHTMLIndex)
	mux.HandleFunc("GET /html/{category}/{slug}", s.handleServeHTML)

	mux.Handle("/", http.FileServer(http.Dir(s.dataDir)))

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Store) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	var req model.PublishRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	resp, err := s.Create(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Store) handleListArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := ListFilter{
		Category: q.Get("category"),
		Tag:      q.Get("tag"),
		Author:   q.Get("author"),
		Query:    q.Get("q"),
	}
	articles, err := s.List(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, articles)
}

func (s *Store) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	article, err := s.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, article)
}

func (s *Store) handleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Store) handleServeHTML(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	slug := r.PathValue("slug")
	if category == "" || slug == "" {
		writeError(w, http.StatusBadRequest, "category and slug required")
		return
	}
	fullPath := filepath.Join(s.dataDir, articlesDirName, category, slug+".html")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "article not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "read error")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleHTMLIndex renders an index page listing all HTML articles grouped by category.
func (s *Store) handleHTMLIndex(w http.ResponseWriter, r *http.Request) {
	// Only match exact /html/ path, not /html/something
	if r.URL.Path != "/html/" && r.URL.Path != "/html" {
		http.NotFound(w, r)
		return
	}

	type htmlArticle struct {
		Title    string
		Category string
		Date     string
		Link     string
	}

	groups := make(map[string][]htmlArticle)
	s.mu.RLock()
	for _, a := range s.articles {
		if a.Format != "html" {
			continue
		}
		cat := a.Category
		if cat == "" {
			cat = "Unclassified"
		}
		catDir := a.Category
		if catDir == "" {
			catDir = unclassifiedDir
		}
		slug := Slugify(a.Title)
		groups[cat] = append(groups[cat], htmlArticle{
			Title:    a.Title,
			Category: cat,
			Date:     a.Date.Format("2006-01-02"),
			Link:     "/html/" + catDir + "/" + slug,
		})
	}
	s.mu.RUnlock()

	var page strings.Builder
	page.WriteString("<!DOCTYPE html>\n<html lang=\"en\"><head><meta charset=\"UTF-8\">")
	page.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	page.WriteString("<title>HTML Articles - DocsHub</title>")
	page.WriteString("<style>")
	page.WriteString("body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:2rem auto;padding:0 1rem;color:#333}")
	page.WriteString("h1{color:#2c3e50;border-bottom:2px solid #3498db;padding-bottom:.5rem}")
	page.WriteString("h2{color:#34495e;margin-top:2rem}")
	page.WriteString("a{color:#3498db;text-decoration:none}")
	page.WriteString("a:hover{text-decoration:underline}")
	page.WriteString(".date{color:#999;font-size:.85em;margin-left:.5rem}")
	page.WriteString("ul{list-style:none;padding-left:0}")
	page.WriteString("li{padding:.3rem 0;border-bottom:1px solid #eee}")
	page.WriteString(".back{margin-bottom:2rem}")
	page.WriteString("</style></head><body>")
	page.WriteString("<p class=\"back\"><a href=\"/\">&larr; Back to DocsHub</a></p>")
	page.WriteString("<h1>HTML Articles</h1>")

	if len(groups) == 0 {
		page.WriteString("<p>No HTML articles yet.</p>")
	} else {
		cats := make([]string, 0, len(groups))
		for cat := range groups {
			cats = append(cats, cat)
		}
		sort.Strings(cats)
		for _, cat := range cats {
			fmt.Fprintf(&page, "<h2>%s</h2><ul>", cat)
			for _, a := range groups[cat] {
				fmt.Fprintf(&page, "<li><a href=\"%s\">%s</a><span class=\"date\">%s</span></li>", a.Link, a.Title, a.Date)
			}
			page.WriteString("</ul>")
		}
	}

	page.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(page.String()))
}
