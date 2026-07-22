package search

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/e0-7/cfx/internal/model"
)

type ProblemFilter struct {
	Query     string
	MinRating int
	MaxRating int
	Tags      []string
	ContestID int
	RatedOnly bool
	Solved    map[string]bool
	Unsolved  bool
	Sort      string
	Limit     int
}

func Problems(all []model.Problem, filter ProblemFilter) []model.Problem {
	result := make([]model.Problem, 0)
	query := words(filter.Query)
	for _, problem := range all {
		if problem.Type != "PROGRAMMING" {
			continue
		}
		haystack := strings.ToLower(strings.Join(append([]string{problem.ID(), problem.Name}, problem.Tags...), " "))
		if !matchesWords(haystack, query) || !hasAllTags(problem.Tags, filter.Tags) {
			continue
		}
		if filter.ContestID != 0 && problem.ContestID != filter.ContestID {
			continue
		}
		if filter.RatedOnly && problem.Rating == 0 {
			continue
		}
		if filter.MinRating != 0 && problem.Rating < filter.MinRating {
			continue
		}
		if filter.MaxRating != 0 && (problem.Rating == 0 || problem.Rating > filter.MaxRating) {
			continue
		}
		if filter.Unsolved && filter.Solved[problem.ID()] {
			continue
		}
		result = append(result, problem)
	}
	sort.SliceStable(result, func(i, j int) bool {
		switch filter.Sort {
		case "rating":
			if result[i].Rating != result[j].Rating {
				if result[i].Rating == 0 {
					return false
				}
				if result[j].Rating == 0 {
					return true
				}
				return result[i].Rating < result[j].Rating
			}
		case "name":
			if result[i].Name != result[j].Name {
				return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
			}
		default:
			if result[i].SolvedCount != result[j].SolvedCount {
				return result[i].SolvedCount > result[j].SolvedCount
			}
		}
		if result[i].ContestID != result[j].ContestID {
			return result[i].ContestID > result[j].ContestID
		}
		return result[i].Index < result[j].Index
	})
	return limitProblems(result, filter.Limit)
}

type ContestFilter struct {
	Query string
	Year  int
	Div   int
	Phase string
	Type  string
	Limit int
}

func Contests(all []model.Contest, filter ContestFilter) []model.Contest {
	result := make([]model.Contest, 0)
	query := words(filter.Query)
	phase := strings.ToUpper(filter.Phase)
	typeName := strings.ToUpper(filter.Type)
	divNeedle := ""
	if filter.Div != 0 {
		divNeedle = fmt.Sprintf("div. %d", filter.Div)
	}
	for _, contest := range all {
		name := strings.ToLower(contest.Name)
		if !matchesWords(name, query) || (divNeedle != "" && !strings.Contains(name, divNeedle)) {
			continue
		}
		if filter.Year != 0 && time.Unix(contest.StartTimeSeconds, 0).UTC().Year() != filter.Year {
			continue
		}
		if phase != "" && strings.ToUpper(contest.Phase) != phase {
			continue
		}
		if typeName != "" && strings.ToUpper(contest.Type) != typeName {
			continue
		}
		result = append(result, contest)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].StartTimeSeconds != result[j].StartTimeSeconds {
			return result[i].StartTimeSeconds > result[j].StartTimeSeconds
		}
		return result[i].ID > result[j].ID
	})
	return limitContests(result, filter.Limit)
}

func words(value string) []string {
	return strings.Fields(strings.ToLower(strings.TrimSpace(value)))
}

func matchesWords(haystack string, query []string) bool {
	for _, word := range query {
		if !strings.Contains(haystack, word) {
			return false
		}
	}
	return true
}

func hasAllTags(values, wanted []string) bool {
	for _, tag := range wanted {
		found := false
		for _, value := range values {
			if strings.EqualFold(value, tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func limitProblems(values []model.Problem, limit int) []model.Problem {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func limitContests(values []model.Contest, limit int) []model.Contest {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}
