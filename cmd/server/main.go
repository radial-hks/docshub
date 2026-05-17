package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/radial-hks/docshub/internal/server"
)

func main() {
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
		log.Fatalf("Failed to initialize server: %v", err)
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Docshub server starting on %s, data dir: %s", addr, dataDir)
	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
