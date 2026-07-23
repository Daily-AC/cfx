package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Daily-AC/cfx/internal/codeforces"
	"github.com/Daily-AC/cfx/internal/model"
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

func TestIntersperseSearchFlags(t *testing.T) {
	got, err := intersperseSearchFlags([]string{"Huawei", "--phase", "FINISHED", "Round", "--json"}, map[string]bool{"phase": true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--phase", "FINISHED", "--json", "Huawei", "Round"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestProblemSearchCommandJSON(t *testing.T) {
	t.Setenv("CFX_HOME", t.TempDir())
	if err := saveCatalog([]model.Problem{
		{ContestID: 4, Index: "A", Name: "Watermelon", Type: "PROGRAMMING", Rating: 800, Tags: []string{"math"}, SolvedCount: 100},
		{ContestID: 5, Index: "A", Name: "Watermelon Returns", Type: "PROGRAMMING", Rating: 1500, Tags: []string{"greedy"}, SolvedCount: 10},
	}); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	c := command{out: out, err: &bytes.Buffer{}, api: codeforces.NewClient()}
	if err := c.problemSearch(context.Background(), []string{"Watermelon", "--max-rating", "1000", "--tag", "math", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"contestId": 4`) || strings.Contains(out.String(), `"contestId": 5`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestContestSearchAndShowCommands(t *testing.T) {
	t.Setenv("CFX_HOME", t.TempDir())
	if err := saveContests([]model.Contest{
		{ID: 100, Name: "Acme Challenge (Div. 2)", Type: "CF", Phase: "FINISHED", StartTimeSeconds: 1704067200},
		{ID: 101, Name: "Acme Challenge (Div. 1)", Type: "CF", Phase: "FINISHED", StartTimeSeconds: 1704067200},
	}); err != nil {
		t.Fatal(err)
	}
	if err := saveCatalog([]model.Problem{{ContestID: 100, Index: "A", Name: "Example", Type: "PROGRAMMING", Rating: 1200}}); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	c := command{out: out, err: &bytes.Buffer{}, api: codeforces.NewClient()}
	if err := c.contestSearch(context.Background(), []string{"Acme", "--year", "2024", "--div", "2", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": 100`) || strings.Contains(out.String(), `"id": 101`) {
		t.Fatalf("unexpected search output: %s", out.String())
	}
	out.Reset()
	if err := c.contestShow(context.Background(), []string{"100", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"name": "Example"`) {
		t.Fatalf("unexpected show output: %s", out.String())
	}
}

func TestCommandHelpIsSuccessful(t *testing.T) {
	if err := Run([]string{"problem", "search", "--help"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("help returned an error: %v", err)
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
