package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/e0-7/cfx/internal/model"
)

const sessionFile = "session.json"

func Create(dir string, statement model.Statement, language string) (model.Session, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return model.Session{}, err
	}
	if entries, err := os.ReadDir(abs); err == nil && len(entries) > 0 {
		return model.Session{}, fmt.Errorf("target directory is not empty: %s", abs)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return model.Session{}, err
	}
	if err := os.MkdirAll(filepath.Join(abs, "samples"), 0o755); err != nil {
		return model.Session{}, err
	}
	source := "main.cpp"
	template := cppTemplate
	if language == "python" {
		source = "main.py"
		template = pythonTemplate
	}
	files := map[string]string{
		"problem.md": renderStatement(statement),
		"AGENTS.md":  agentInstructions(statement, source),
		source:       template,
	}
	for i, sample := range statement.Samples {
		files[filepath.Join("samples", fmt.Sprintf("%02d.in", i+1))] = sample.Input
		files[filepath.Join("samples", fmt.Sprintf("%02d.out", i+1))] = sample.Output
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(abs, name), []byte(body), 0o644); err != nil {
			return model.Session{}, err
		}
	}
	session := model.Session{
		Version:   1,
		Problem:   statement.Problem,
		Language:  language,
		Source:    source,
		StartedAt: time.Now(),
		Attempts:  []model.Attempt{},
	}
	if err := Save(abs, session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func CreateContest(dir string, contest model.Contest, statements []model.Statement, language string) (model.ContestWorkspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return model.ContestWorkspace{}, err
	}
	if entries, err := os.ReadDir(abs); err == nil && len(entries) > 0 {
		return model.ContestWorkspace{}, fmt.Errorf("target directory is not empty: %s", abs)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return model.ContestWorkspace{}, err
	}
	parent := filepath.Dir(abs)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return model.ContestWorkspace{}, err
	}
	staging, err := os.MkdirTemp(parent, ".cfx-contest-")
	if err != nil {
		return model.ContestWorkspace{}, err
	}
	defer os.RemoveAll(staging)

	result := model.ContestWorkspace{
		Version: 1, Contest: contest, Language: language, CreatedAt: time.Now(),
		Problems: make([]model.ContestProblemWorkspace, 0, len(statements)),
	}
	for _, statement := range statements {
		directory, err := safeProblemDirectory(statement.Index)
		if err != nil {
			return model.ContestWorkspace{}, err
		}
		if _, err := Create(filepath.Join(staging, directory), statement, language); err != nil {
			return model.ContestWorkspace{}, err
		}
		result.Problems = append(result.Problems, model.ContestProblemWorkspace{Problem: statement.Problem, Directory: directory})
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return model.ContestWorkspace{}, err
	}
	if err := os.WriteFile(filepath.Join(staging, "contest.json"), append(body, '\n'), 0o644); err != nil {
		return model.ContestWorkspace{}, err
	}
	if err := os.WriteFile(filepath.Join(staging, "README.md"), []byte(renderContest(result)), 0o644); err != nil {
		return model.ContestWorkspace{}, err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return model.ContestWorkspace{}, fmt.Errorf("prepare empty target directory: %w", err)
	}
	if err := os.Rename(staging, abs); err != nil {
		return model.ContestWorkspace{}, err
	}
	return result, nil
}

func safeProblemDirectory(index string) (string, error) {
	if index == "" {
		return "", errors.New("problem index is empty")
	}
	for _, r := range index {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return "", fmt.Errorf("problem index %q is unsafe as a directory name", index)
		}
	}
	return index, nil
}

func renderContest(session model.ContestWorkspace) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %d. %s\n\n", session.Contest.ID, session.Contest.Name)
	fmt.Fprintf(&b, "- Phase: %s\n- Type: %s\n- Problems: %d\n- Source: https://codeforces.com/contest/%d\n\n", session.Contest.Phase, session.Contest.Type, len(session.Problems), session.Contest.ID)
	b.WriteString("## Problems\n\n")
	for _, problem := range session.Problems {
		fmt.Fprintf(&b, "- [%s. %s](%s/problem.md) - rating %d\n", problem.Problem.Index, problem.Problem.Name, problem.Directory, problem.Problem.Rating)
	}
	return b.String()
}

func Find(start string) (string, model.Session, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", model.Session{}, err
	}
	for {
		path := filepath.Join(dir, sessionFile)
		if _, err := os.Stat(path); err == nil {
			session, err := Load(dir)
			return dir, session, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", model.Session{}, errors.New("not inside a cfx training workspace")
}

func Load(dir string) (model.Session, error) {
	b, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return model.Session{}, err
	}
	var session model.Session
	if err := json.Unmarshal(b, &session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func Save(dir string, session model.Session) error {
	b, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(dir, sessionFile), b, 0o644)
}

func Record(dir string, session *model.Session, attempt model.Attempt) error {
	session.Attempts = append(session.Attempts, attempt)
	return Save(dir, *session)
}

func renderStatement(s model.Statement) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s. %s\n\n", s.ID(), s.Name)
	fmt.Fprintf(&b, "- Rating: %d\n- Time limit: %s\n- Memory limit: %s\n- Tags: %s\n- Source: %s\n\n", s.Rating, s.TimeLimit, s.MemoryLimit, strings.Join(s.Tags, ", "), s.URL)
	fmt.Fprintf(&b, "## Statement\n\n%s\n\n## Input\n\n%s\n\n## Output\n\n%s\n", s.Body, s.Input, s.Output)
	if s.Note != "" {
		fmt.Fprintf(&b, "\n## Note\n\n%s\n", s.Note)
	}
	return b.String()
}

func agentInstructions(s model.Statement, source string) string {
	code := "`"
	return fmt.Sprintf(`# Codeforces training contract

This directory is an active training session for %s (%s).

## Goal

Coach the user toward an independently understood accepted solution. Use
problem.md and the sample files as the source of truth. Edit %s only when the
user explicitly asks for implementation or has already supplied an approach to
implement.

## Interaction policy

- Start by asking for or reviewing the user's current idea.
- When asked for help, give the smallest useful hint first. Do not reveal a full
  solution, proof, or implementation unless explicitly requested.
- Challenge complexity claims and edge cases with concrete counterexamples.
- Before submission, run %scfx test --json%s and inspect the actual output.
- Submit only after the user explicitly asks; %scfx submit --yes%s is an external
  action on their Codeforces account.
- After a verdict, summarize the mistake pattern or reusable technique in two or
  three sentences. Do not overwrite the user's solution with an editorial one.

## Evidence

%ssession.json%s records test and submission attempts. Never claim a test passed
or a submission was accepted without a matching command result.
`, s.ID(), s.Name, source, code, code, code, code, code, code)
}

const cppTemplate = `#include <algorithm>
#include <array>
#include <cmath>
#include <cstdint>
#include <iostream>
#include <map>
#include <numeric>
#include <queue>
#include <set>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    return 0;
}
`

const pythonTemplate = `import sys


def solve() -> None:
    pass


if __name__ == "__main__":
    solve()
`
