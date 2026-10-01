package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/patrickdappollonio/mockingjay/internal/config"
	"github.com/patrickdappollonio/mockingjay/internal/server"
)

func TestIsConfigChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configFile string
		event      fsnotify.Event
		want       bool
	}{
		{name: "write to the config file", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Write}, want: true},
		{name: "config file replaced by rename", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Create}, want: true},
		{name: "unclean config path", configFile: "./dir//config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Write}, want: true},
		{name: "write to a sibling file", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml.swp", Op: fsnotify.Write}, want: false},
		{name: "config file removed", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Remove}, want: false},
		{name: "config file renamed away", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Rename}, want: false},
		{name: "config file chmod", configFile: "dir/config.yaml", event: fsnotify.Event{Name: "dir/config.yaml", Op: fsnotify.Chmod}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isConfigChange(tt.event, tt.configFile); got != tt.want {
				t.Errorf("isConfigChange(%v, %q) = %v, want %v", tt.event, tt.configFile, got, tt.want)
			}
		})
	}
}

func TestStartConfigWatcher_ReloadsOnSave(t *testing.T) {
	tests := []struct {
		name string
		save func(t *testing.T, path, content string)
	}{
		{
			name: "file written in place",
			save: func(t *testing.T, path, content string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("failed to write %q: %v", path, err)
				}
			},
		},
		{
			name: "file replaced by rename",
			save: func(t *testing.T, path, content string) {
				t.Helper()
				tmp := path + ".tmp"
				if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
					t.Fatalf("failed to write %q: %v", tmp, err)
				}
				if err := os.Rename(tmp, path); err != nil {
					t.Fatalf("failed to rename %q to %q: %v", tmp, path, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configFile, []byte("routes:\n  - path: /old\n    template: old\n"), 0o644); err != nil {
				t.Fatalf("failed to write %q: %v", configFile, err)
			}
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				t.Fatalf("LoadConfig(%q) returned unexpected error: %v", configFile, err)
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			srv, err := server.NewServer(cfg, configFile, ":0", logger, "test")
			if err != nil {
				t.Fatalf("NewServer() returned unexpected error: %v", err)
			}
			if err := startConfigWatcher(configFile, srv, logger, t.Context()); err != nil {
				t.Fatalf("startConfigWatcher(%q) returned unexpected error: %v", configFile, err)
			}

			tt.save(t, configFile, "routes:\n  - path: /new\n    template: new\n")

			deadline := time.Now().Add(3 * time.Second)
			for {
				rec := httptest.NewRecorder()
				srv.ServeHTTP(rec, httptest.NewRequest("GET", "/new", nil))
				if rec.Code == http.StatusOK {
					return
				}
				if time.Now().After(deadline) {
					t.Fatalf("GET /new status = %d after saving the config, want 200 once reloaded", rec.Code)
				}
				time.Sleep(20 * time.Millisecond)
			}
		})
	}
}
