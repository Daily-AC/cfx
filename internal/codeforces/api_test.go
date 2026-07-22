package codeforces

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProblemsParsesRealResponseShape(t *testing.T) {
	const fixture = `{"status":"OK","result":{"problems":[{"contestId":2245,"index":"D2","name":"Construct an Array (Hard Version)","type":"PROGRAMMING","points":1000.0,"rating":2400,"tags":["2-sat","constructive algorithms","dfs and similar","graphs","implementation"]}],"problemStatistics":[{"contestId":2245,"index":"D2","solvedCount":1574}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/problemset.problems" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, fixture)
	}))
	defer server.Close()
	client := NewClient()
	client.BaseURL = server.URL
	problems, err := client.Problems(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || problems[0].ID() != "2245D2" || problems[0].SolvedCount != 1574 || problems[0].Rating != 2400 {
		t.Fatalf("unexpected problems: %+v", problems)
	}
}

func TestAPIFailureIncludesComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"FAILED","comment":"Call limit exceeded"}`)
	}))
	defer server.Close()
	client := NewClient()
	client.BaseURL = server.URL
	if err := client.Ping(context.Background()); err == nil || err.Error() != "Codeforces API system.status: Call limit exceeded" {
		t.Fatalf("error = %v", err)
	}
}
