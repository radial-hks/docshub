package server

import "net/http"

// TODO: implement (Task 4)

func (s *Store) Handler() http.Handler {
	return http.NewServeMux()
}
