package model

import (
	"fmt"
	"time"
)

type Problem struct {
	ContestID   int      `json:"contestId"`
	Index       string   `json:"index"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Points      float64  `json:"points,omitempty"`
	Rating      int      `json:"rating,omitempty"`
	Tags        []string `json:"tags"`
	SolvedCount int      `json:"solvedCount,omitempty"`
}

type Contest struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	Phase               string `json:"phase"`
	Frozen              bool   `json:"frozen"`
	DurationSeconds     int    `json:"durationSeconds"`
	StartTimeSeconds    int64  `json:"startTimeSeconds,omitempty"`
	RelativeTimeSeconds int64  `json:"relativeTimeSeconds,omitempty"`
}

func (c Contest) StartTime() time.Time {
	if c.StartTimeSeconds == 0 {
		return time.Time{}
	}
	return time.Unix(c.StartTimeSeconds, 0)
}

func (p Problem) ID() string { return formatID(p.ContestID, p.Index) }

type Sample struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Statement struct {
	Problem
	TimeLimit   string   `json:"timeLimit"`
	MemoryLimit string   `json:"memoryLimit"`
	Body        string   `json:"body"`
	Input       string   `json:"input"`
	Output      string   `json:"output"`
	Note        string   `json:"note,omitempty"`
	Samples     []Sample `json:"samples"`
	URL         string   `json:"url"`
}

type Attempt struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Success bool      `json:"success"`
	Detail  string    `json:"detail"`
}

type Session struct {
	Version   int       `json:"version"`
	Problem   Problem   `json:"problem"`
	Language  string    `json:"language"`
	Source    string    `json:"source"`
	StartedAt time.Time `json:"startedAt"`
	Attempts  []Attempt `json:"attempts"`
}

type ContestProblemWorkspace struct {
	Problem   Problem `json:"problem"`
	Directory string  `json:"directory"`
}

type ContestWorkspace struct {
	Version   int                       `json:"version"`
	Contest   Contest                   `json:"contest"`
	Language  string                    `json:"language"`
	CreatedAt time.Time                 `json:"createdAt"`
	Problems  []ContestProblemWorkspace `json:"problems"`
}

type Submission struct {
	ID                  int64   `json:"id"`
	ContestID           int     `json:"contestId"`
	CreationTimeSeconds int64   `json:"creationTimeSeconds"`
	Problem             Problem `json:"problem"`
	ProgrammingLanguage string  `json:"programmingLanguage"`
	Verdict             string  `json:"verdict,omitempty"`
	Testset             string  `json:"testset,omitempty"`
	PassedTestCount     int     `json:"passedTestCount"`
	TimeConsumedMillis  int     `json:"timeConsumedMillis"`
	MemoryConsumedBytes int64   `json:"memoryConsumedBytes"`
}

func formatID(contestID int, index string) string {
	return fmt.Sprintf("%d%s", contestID, index)
}
