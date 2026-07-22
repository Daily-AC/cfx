package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e0-7/cfx/internal/model"
)

func TestCreateAndFind(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cf-4a")
	statement := model.Statement{
		Problem:   model.Problem{ContestID: 4, Index: "A", Name: "Watermelon", Rating: 800, Tags: []string{"brute force", "math"}},
		TimeLimit: "1 second", MemoryLimit: "64 megabytes", Body: "Statement", Input: "Input", Output: "Output",
		Samples: []model.Sample{{Input: "8\n", Output: "YES\n"}}, URL: "https://codeforces.com/problemset/problem/4/A",
	}
	session, err := Create(dir, statement, "cpp")
	if err != nil {
		t.Fatal(err)
	}
	if session.Source != "main.cpp" {
		t.Fatalf("source = %s", session.Source)
	}
	_, found, err := Find(filepath.Join(dir, "samples"))
	if err != nil || found.Problem.ID() != "4A" {
		t.Fatalf("find = %+v, %v", found, err)
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "cfx test --json") || !strings.Contains(string(agents), "smallest useful hint") {
		t.Fatalf("unexpected AGENTS.md: %s", agents)
	}
}
