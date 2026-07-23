package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Daily-AC/cfx/internal/config"
	"github.com/Daily-AC/cfx/internal/model"
)

type CaseResult struct {
	Name           string `json:"name"`
	Passed         bool   `json:"passed"`
	DurationMillis int64  `json:"durationMillis"`
	Expected       string `json:"expected,omitempty"`
	Actual         string `json:"actual,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
	Error          string `json:"error,omitempty"`
}

type Result struct {
	Passed bool         `json:"passed"`
	Cases  []CaseResult `json:"cases"`
}

func Run(ctx context.Context, cfg config.Config, dir string, session model.Session) (Result, error) {
	command, cleanup, err := build(ctx, cfg, dir, session)
	if err != nil {
		return Result{}, err
	}
	defer cleanup()
	inputs, err := filepath.Glob(filepath.Join(dir, "samples", "*.in"))
	if err != nil {
		return Result{}, err
	}
	sort.Strings(inputs)
	if len(inputs) == 0 {
		return Result{}, fmt.Errorf("no sample inputs in %s", filepath.Join(dir, "samples"))
	}
	result := Result{Passed: true}
	for _, inputPath := range inputs {
		caseResult := runCase(ctx, dir, command, inputPath)
		result.Cases = append(result.Cases, caseResult)
		result.Passed = result.Passed && caseResult.Passed
	}
	return result, nil
}

func build(ctx context.Context, cfg config.Config, dir string, session model.Session) ([]string, func(), error) {
	source := filepath.Join(dir, session.Source)
	if session.Language == "python" {
		return []string{cfg.Python, source}, func() {}, nil
	}
	binary := filepath.Join(dir, ".cfx", "solution")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, cfg.CPPCompiler, "-std=c++23", "-O2", "-pipe", "-Wall", "-Wextra", "-o", binary, source)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("compile failed: %s", strings.TrimSpace(stderr.String()))
	}
	return []string{binary}, func() {}, nil
}

func runCase(parent context.Context, dir string, command []string, inputPath string) CaseResult {
	name := strings.TrimSuffix(filepath.Base(inputPath), ".in")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return CaseResult{Name: name, Error: err.Error()}
	}
	expectedPath := strings.TrimSuffix(inputPath, ".in") + ".out"
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		return CaseResult{Name: name, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	started := time.Now()
	err = cmd.Run()
	result := CaseResult{Name: name, DurationMillis: time.Since(started).Milliseconds(), Expected: string(expected), Actual: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "time limit exceeded (5s local limit)"
		return result
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Passed = normalize(expected) == normalize(stdout.Bytes())
	return result
}

func normalize(value []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(value), "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
