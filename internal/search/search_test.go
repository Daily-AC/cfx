package search

import (
	"testing"

	"github.com/Daily-AC/cfx/internal/model"
)

func TestProblemsFiltersAndSorts(t *testing.T) {
	problems := []model.Problem{
		{ContestID: 10, Index: "A", Name: "Graph Paths", Type: "PROGRAMMING", Rating: 1200, Tags: []string{"graphs", "dp"}, SolvedCount: 20},
		{ContestID: 11, Index: "B", Name: "Graph Coloring", Type: "PROGRAMMING", Rating: 1400, Tags: []string{"graphs", "greedy"}, SolvedCount: 100},
		{ContestID: 12, Index: "C", Name: "Graph Query", Type: "PROGRAMMING", Rating: 1300, Tags: []string{"graphs", "dp"}, SolvedCount: 50},
	}
	got := Problems(problems, ProblemFilter{Query: "graph", MinRating: 1200, MaxRating: 1350, Tags: []string{"dp"}, Solved: map[string]bool{"10A": true}, Unsolved: true, Sort: "rating", Limit: 10})
	if len(got) != 1 || got[0].ID() != "12C" {
		t.Fatalf("unexpected problems: %+v", got)
	}
}

func TestContestsSearchesSponsorYearAndDivision(t *testing.T) {
	contests := []model.Contest{
		{ID: 1, Name: "Codeforces Round sponsored by Huawei (Div. 2)", Type: "CF", Phase: "FINISHED", StartTimeSeconds: 1704067200},
		{ID: 2, Name: "Codeforces Round sponsored by Huawei (Div. 1)", Type: "CF", Phase: "FINISHED", StartTimeSeconds: 1735689600},
		{ID: 3, Name: "Educational Codeforces Round (Div. 2)", Type: "ICPC", Phase: "FINISHED", StartTimeSeconds: 1704067200},
	}
	got := Contests(contests, ContestFilter{Query: "huawei", Year: 2024, Div: 2, Phase: "finished", Type: "cf", Limit: 10})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unexpected contests: %+v", got)
	}
}
