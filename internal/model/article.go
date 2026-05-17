package model

import "time"

type Article struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	File     string    `json:"file"`
	Category string    `json:"category"`
	Tags     []string  `json:"tags"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	Summary  string    `json:"summary"`
	Version  int       `json:"version"`
	Format   string    `json:"format"`
}

type PublishRequest struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	Author    string   `json:"author"`
	VersionOf string   `json:"version_of,omitempty"`
	Format    string   `json:"format,omitempty"`
}

type PublishResponse struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	Path             string   `json:"path"`
	Version          int      `json:"version"`
	PreviousVersions []string `json:"previous_versions,omitempty"`
}
