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

func TestCreateContest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cf-contest-4")
	contest := model.Contest{ID: 4, Name: "Codeforces Beta Round 4", Phase: "FINISHED", Type: "CF"}
	statements := []model.Statement{
		{Problem: model.Problem{ContestID: 4, Index: "A", Name: "Watermelon", Type: "PROGRAMMING", Rating: 800}, Samples: []model.Sample{{Input: "8\n", Output: "YES\n"}}},
		{Problem: model.Problem{ContestID: 4, Index: "B", Name: "Before an Exam", Type: "PROGRAMMING", Rating: 1200}, Samples: []model.Sample{{Input: "1\n", Output: "YES\n"}}},
	}
	result, err := CreateContest(dir, contest, statements, "cpp")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Problems) != 2 || result.Problems[1].Directory != "B" {
		t.Fatalf("unexpected contest workspace: %+v", result)
	}
	for _, path := range []string{"contest.json", "README.md", "A/session.json", "B/main.cpp"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
}

func TestCreateContestDoesNotLeavePartialTarget(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "contest")
	statements := []model.Statement{
		{Problem: model.Problem{ContestID: 1, Index: "A", Name: "Valid"}},
		{Problem: model.Problem{ContestID: 1, Index: "../B", Name: "Unsafe"}},
	}
	if _, err := CreateContest(dir, model.Contest{ID: 1}, statements, "cpp"); err == nil {
		t.Fatal("expected unsafe index error")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("partial target exists: %v", err)
	}
}
