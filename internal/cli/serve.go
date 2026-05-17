package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/radial-hks/docshub/internal/server"
)

func RunServe() error {
	port := os.Getenv("DOCSHUB_PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := os.Getenv("DOCSHUB_DATA")
	if dataDir == "" {
		dataDir = "./web"
	}

	s, err := server.New(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Docshub server starting on %s, data dir: %s", addr, dataDir)
	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
