package cli

import (
	"testing"
)

func TestLoadConfigReturnsDefaultsWhenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ServerURL != defaultServerURL {
		t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, defaultServerURL)
	}
	if cfg.Author != "" {
		t.Errorf("Author = %q, want empty", cfg.Author)
	}
}

func TestConfigSaveLoadRoundtrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	in := &Config{ServerURL: "http://10.0.0.1:9000", Author: "radial"}
	if err := SaveConfig(in); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	out, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if out.ServerURL != in.ServerURL {
		t.Errorf("ServerURL = %q, want %q", out.ServerURL, in.ServerURL)
	}
	if out.Author != in.Author {
		t.Errorf("Author = %q, want %q", out.Author, in.Author)
	}
}
