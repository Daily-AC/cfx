package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Daily-AC/cfx/internal/config"
	"github.com/Daily-AC/cfx/internal/model"
	searcher "github.com/Daily-AC/cfx/internal/search"
	"github.com/Daily-AC/cfx/internal/workspace"
)

type stringListFlag []string

func (f *stringListFlag) String() string { return strings.Join(*f, ",") }
func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

func (c *command) problem(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "search" {
		return errors.New("usage: cfx problem search [QUERY] [flags]")
	}
	return c.problemSearch(ctx, args[1:])
}

func (c *command) problemSearch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("problem search", flag.ContinueOnError)
	fs.SetOutput(c.err)
	minRating := fs.Int("min-rating", 0, "minimum rating")
	maxRating := fs.Int("max-rating", 0, "maximum rating")
	contestID := fs.Int("contest", 0, "contest id")
	ratedOnly := fs.Bool("rated", false, "exclude unrated problems")
	unsolved := fs.Bool("unsolved", false, "exclude accepted problems")
	sortBy := fs.String("sort", "solved", "solved, rating, or name")
	limit := fs.Int("limit", 20, "maximum results; 0 means all")
	jsonOut := fs.Bool("json", false, "JSON output")
	var tags stringListFlag
	fs.Var(&tags, "tag", "required tag; repeat for multiple tags")
	args, err := intersperseSearchFlags(args, map[string]bool{
		"min-rating": true, "max-rating": true, "contest": true, "tag": true,
		"sort": true, "limit": true,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *minRating < 0 || *maxRating < 0 || (*maxRating != 0 && *minRating > *maxRating) {
		return errors.New("invalid rating range")
	}
	if *limit < 0 {
		return errors.New("limit cannot be negative")
	}
	if *sortBy != "solved" && *sortBy != "rating" && *sortBy != "name" {
		return errors.New("sort must be solved, rating, or name")
	}
	problems, err := c.problems(ctx)
	if err != nil {
		return err
	}
	solved, err := c.solvedProblems(ctx, *unsolved)
	if err != nil {
		return err
	}
	result := searcher.Problems(problems, searcher.ProblemFilter{
		Query: strings.Join(fs.Args(), " "), MinRating: *minRating, MaxRating: *maxRating,
		Tags: tags, ContestID: *contestID, RatedOnly: *ratedOnly, Solved: solved,
		Unsolved: *unsolved, Sort: *sortBy, Limit: *limit,
	})
	if len(result) == 0 {
		return errors.New("no matching problems")
	}
	if *jsonOut {
		return writeJSON(c.out, result)
	}
	for _, problem := range result {
		rating := "unrated"
		if problem.Rating != 0 {
			rating = strconv.Itoa(problem.Rating)
		}
		fmt.Fprintf(c.out, "%-7s  %7s  %7d  %s  [%s]\n", problem.ID(), rating, problem.SolvedCount, problem.Name, strings.Join(problem.Tags, ", "))
	}
	return nil
}

func (c *command) contest(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cfx contest search|show|start")
	}
	switch args[0] {
	case "search":
		return c.contestSearch(ctx, args[1:])
	case "show":
		return c.contestShow(ctx, args[1:])
	case "start":
		return c.contestStart(ctx, args[1:])
	default:
		return errors.New("usage: cfx contest search|show|start")
	}
}

func (c *command) contestSearch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("contest search", flag.ContinueOnError)
	fs.SetOutput(c.err)
	year := fs.Int("year", 0, "start year in UTC")
	division := fs.Int("div", 0, "division number")
	phase := fs.String("phase", "", "BEFORE, CODING, PENDING_SYSTEM_TEST, SYSTEM_TEST, or FINISHED")
	typeName := fs.String("type", "", "contest type, e.g. CF or ICPC")
	limit := fs.Int("limit", 20, "maximum results; 0 means all")
	jsonOut := fs.Bool("json", false, "JSON output")
	args, err := intersperseSearchFlags(args, map[string]bool{
		"year": true, "div": true, "phase": true, "type": true, "limit": true,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *year < 0 || *division < 0 || *limit < 0 {
		return errors.New("year, div, and limit cannot be negative")
	}
	contests, err := c.contests(ctx)
	if err != nil {
		return err
	}
	result := searcher.Contests(contests, searcher.ContestFilter{
		Query: strings.Join(fs.Args(), " "), Year: *year, Div: *division,
		Phase: *phase, Type: *typeName, Limit: *limit,
	})
	if len(result) == 0 {
		return errors.New("no matching contests")
	}
	if *jsonOut {
		return writeJSON(c.out, result)
	}
	for _, contest := range result {
		start := "unknown"
		if !contest.StartTime().IsZero() {
			start = contest.StartTime().UTC().Format("2006-01-02 15:04")
		}
		fmt.Fprintf(c.out, "%5d  %-20s  %-8s  %s\n", contest.ID, start+" UTC", contest.Phase, contest.Name)
	}
	return nil
}

type contestDetails struct {
	Contest  model.Contest   `json:"contest"`
	Problems []model.Problem `json:"problems"`
}

func (c *command) contestShow(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("contest show", flag.ContinueOnError)
	fs.SetOutput(c.err)
	jsonOut := fs.Bool("json", false, "JSON output")
	id, rest, err := leadingInt(args, "usage: cfx contest show <contest-id> [--json]")
	if err != nil {
		return err
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cfx contest show <contest-id> [--json]")
	}
	details, err := c.getContestDetails(ctx, id)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(c.out, details)
	}
	start := details.Contest.StartTime().UTC().Format(time.RFC3339)
	fmt.Fprintf(c.out, "%d  %s\nPhase: %s  Type: %s  Start: %s  Duration: %s\n\n", details.Contest.ID, details.Contest.Name, details.Contest.Phase, details.Contest.Type, start, (time.Duration(details.Contest.DurationSeconds) * time.Second).String())
	for _, problem := range details.Problems {
		fmt.Fprintf(c.out, "%-4s  rating=%-4d  solved=%-7d  %s\n", problem.Index, problem.Rating, problem.SolvedCount, problem.Name)
	}
	return nil
}

func (c *command) contestStart(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("contest start", flag.ContinueOnError)
	fs.SetOutput(c.err)
	dir := fs.String("dir", "", "workspace directory")
	lang := fs.String("lang", "", "cpp or python")
	jsonOut := fs.Bool("json", false, "JSON output")
	id, rest, err := leadingInt(args, "usage: cfx contest start <contest-id> [--dir PATH] [--lang cpp|python] [--json]")
	if err != nil {
		return err
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cfx contest start <contest-id> [--dir PATH] [--lang cpp|python] [--json]")
	}
	details, err := c.getContestDetails(ctx, id)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	language := cfg.Language
	if *lang != "" {
		language = *lang
	}
	if language != "cpp" && language != "python" {
		return errors.New("language must be cpp or python")
	}
	statements := make([]model.Statement, 0, len(details.Problems))
	for i, problem := range details.Problems {
		fmt.Fprintf(c.err, "Fetching %s (%d/%d)...\n", problem.ID(), i+1, len(details.Problems))
		statement, err := c.fetchStatement(ctx, cfg, problem)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", problem.ID(), err)
		}
		statements = append(statements, statement)
	}
	target := *dir
	if target == "" {
		target = fmt.Sprintf("cf-contest-%d", id)
	}
	result, err := workspace.CreateContest(target, details.Contest, statements, language)
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(target)
	output := map[string]any{"directory": abs, "workspace": result}
	if *jsonOut {
		return writeJSON(c.out, output)
	}
	fmt.Fprintf(c.out, "Created %s with %d problems.\n", abs, len(result.Problems))
	return nil
}

func (c *command) getContestDetails(ctx context.Context, id int) (contestDetails, error) {
	contests, err := c.contests(ctx)
	if err != nil {
		return contestDetails{}, err
	}
	var contest model.Contest
	found := false
	for _, candidate := range contests {
		if candidate.ID == id {
			contest = candidate
			found = true
			break
		}
	}
	if !found {
		return contestDetails{}, fmt.Errorf("contest %d not found", id)
	}
	problems, err := c.problems(ctx)
	if err != nil {
		return contestDetails{}, err
	}
	selected := make([]model.Problem, 0)
	for _, problem := range problems {
		if problem.ContestID == id && problem.Type == "PROGRAMMING" {
			selected = append(selected, problem)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].Index < selected[j].Index })
	if len(selected) == 0 {
		return contestDetails{}, fmt.Errorf("contest %d has no problems in the public problemset", id)
	}
	return contestDetails{Contest: contest, Problems: selected}, nil
}

func (c *command) problems(ctx context.Context) ([]model.Problem, error) {
	problems, err := loadCatalog()
	if !errors.Is(err, os.ErrNotExist) {
		return problems, err
	}
	problems, err = c.api.Problems(ctx)
	if err == nil {
		err = saveCatalog(problems)
	}
	return problems, err
}

func (c *command) contests(ctx context.Context) ([]model.Contest, error) {
	contests, err := loadContests()
	if !errors.Is(err, os.ErrNotExist) {
		return contests, err
	}
	contests, err = c.api.Contests(ctx)
	if err == nil {
		err = saveContests(contests)
	}
	return contests, err
}

func (c *command) solvedProblems(ctx context.Context, enabled bool) (map[string]bool, error) {
	solved := make(map[string]bool)
	if !enabled {
		return solved, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Handle == "" {
		return nil, errors.New("--unsolved requires authentication or a configured handle")
	}
	submissions, err := c.api.UserStatus(ctx, cfg.Handle, 1, 10000)
	if err != nil {
		return nil, err
	}
	for _, submission := range submissions {
		if submission.Verdict == "OK" {
			solved[submission.Problem.ID()] = true
		}
	}
	return solved, nil
}

func leadingInt(args []string, usage string) (int, []string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return 0, nil, errors.New(usage)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id <= 0 {
		return 0, nil, errors.New(usage)
	}
	return id, args[1:], nil
}

func intersperseSearchFlags(args []string, valueFlags map[string]bool) ([]string, error) {
	flags := make([]string, 0, len(args))
	query := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			query = append(query, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			name = before
			continue
		}
		if valueFlags[name] {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag needs an argument: %s", arg)
			}
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, query...), nil
}

type contestCatalog struct {
	SyncedAt time.Time       `json:"syncedAt"`
	Contests []model.Contest `json:"contests"`
}

func saveContests(contests []model.Contest) error {
	path, err := config.CachePath("contests.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(contestCatalog{SyncedAt: time.Now(), Contests: contests})
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o600)
}

func loadContests() ([]model.Contest, error) {
	path, err := config.CachePath("contests.json")
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data contestCatalog
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Contests, nil
}
