package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/radial-hks/docshub/internal/model"
)

func (s *Store) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/articles", s.handleCreateArticle)
	mux.HandleFunc("GET /api/articles", s.handleListArticles)
	mux.HandleFunc("GET /api/articles/{id}", s.handleGetArticle)
	mux.HandleFunc("DELETE /api/articles/{id}", s.handleDeleteArticle)

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
