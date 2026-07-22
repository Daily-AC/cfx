package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e0-7/cfx/internal/codeforces"
)

func TestParseProblemID(t *testing.T) {
	id, index, err := parseProblemID("1795b")
	if err != nil || id != 1795 || index != "B" {
		t.Fatalf("got %d %q %v", id, index, err)
	}
	if _, _, err := parseProblemID("bad"); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func TestDefaultProgramType(t *testing.T) {
	if got := defaultProgramType("cpp"); got != "91" {
		t.Fatalf("C++ program type = %s", got)
	}
	if got := defaultProgramType("python"); got != "70" {
		t.Fatalf("Python program type = %s", got)
	}
}

func TestDoctorJSONReturnsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	api := codeforces.NewClient()
	api.BaseURL = server.URL
	c := command{out: &bytes.Buffer{}, err: &bytes.Buffer{}, api: api}
	if err := c.doctor(context.Background(), []string{"--json"}); err == nil {
		t.Fatal("expected doctor failure")
	}
}
