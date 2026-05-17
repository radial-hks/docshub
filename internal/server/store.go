package server

import "net/http"

// TODO: implement

type Store struct {
	dataDir string
}

func New(dataDir string) (*Store, error) {
	return &Store{dataDir: dataDir}, nil
}

func (s *Store) Handler() http.Handler {
	return http.NewServeMux()
}
