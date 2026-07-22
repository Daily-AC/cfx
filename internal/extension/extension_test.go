package extension

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallEmbedsToken(t *testing.T) {
	t.Setenv("CFX_HOME", t.TempDir())
	dir, token, err := Install("")
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d", len(token))
	}
	body, err := os.ReadFile(filepath.Join(dir, "service-worker.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), token) || strings.Contains(string(body), tokenPlaceholder) {
		t.Fatal("service worker does not contain the generated token")
	}
	if got, err := Path(); err != nil || got != dir {
		t.Fatalf("Path() = %q, %v", got, err)
	}
}
