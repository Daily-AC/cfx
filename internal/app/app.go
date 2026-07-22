package app

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/e0-7/cfx/internal/bridge"
	"github.com/e0-7/cfx/internal/codeforces"
	"github.com/e0-7/cfx/internal/config"
	"github.com/e0-7/cfx/internal/extension"
	"github.com/e0-7/cfx/internal/model"
	"github.com/e0-7/cfx/internal/runner"
	"github.com/e0-7/cfx/internal/workspace"
)

var version = "0.1.0"

type command struct {
	out io.Writer
	err io.Writer
	api *codeforces.Client
}

func Run(args []string, stdout, stderr io.Writer) error {
	err := run(args, stdout, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stdout)
		return nil
	}
	c := &command{out: stdout, err: stderr, api: codeforces.NewClient()}
	ctx := context.Background()
	switch args[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	case "version", "--version":
		fmt.Fprintln(stdout, version)
		return nil
	case "config":
		return c.config(args[1:])
	case "doctor":
		return c.doctor(ctx, args[1:])
	case "auth":
		return c.auth(ctx, args[1:])
	case "extension":
		return c.extension(args[1:])
	case "sync":
		return c.sync(ctx, args[1:])
	case "pick":
		return c.pick(ctx, args[1:])
	case "problem":
		return c.problem(ctx, args[1:])
	case "contest":
		return c.contest(ctx, args[1:])
	case "start":
		return c.start(ctx, args[1:])
	case "show":
		return c.show(args[1:])
	case "test":
		return c.test(ctx, args[1:])
	case "status":
		return c.status(ctx, args[1:])
	case "submit":
		return c.submit(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q; run `cfx help`", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `cfx - Codeforces training CLI for humans and agents

Usage:
  cfx auth
  cfx extension install|path
  cfx config get|set <key> [value]
  cfx doctor [--json]
  cfx sync [--json]
  cfx pick [--rating N] [--min-rating N] [--max-rating N] [--tag TAG] [--unsolved] [--json]
  cfx problem search [QUERY] [--min-rating N] [--max-rating N] [--contest ID] [--rated] [--tag TAG]... [--unsolved] [--sort solved|rating|name] [--limit N] [--json]
  cfx contest search [QUERY] [--year N] [--div N] [--phase PHASE] [--type TYPE] [--limit N] [--json]
  cfx contest show <contest-id> [--json]
  cfx contest start <contest-id> [--dir PATH] [--lang cpp|python] [--json]
  cfx start <contest><index> [--dir PATH] [--lang cpp|python] [--json]
  cfx show [--json]
  cfx test [--json]
  cfx status [--wait] [--json]
  cfx submit --yes [--json]
`)
}

func (c *command) config(args []string) error {
	if len(args) < 2 || (args[0] != "get" && args[0] != "set") {
		return errors.New("usage: cfx config get|set <key> [value]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	key := args[1]
	if args[0] == "get" {
		value, err := configValue(cfg, key)
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, value)
		return nil
	}
	if len(args) != 3 {
		return errors.New("usage: cfx config set <key> <value>")
	}
	if err := setConfigValue(&cfg, key, args[2]); err != nil {
		return err
	}
	return config.Save(cfg)
}

func configValue(cfg config.Config, key string) (string, error) {
	switch key {
	case "handle":
		return cfg.Handle, nil
	case "language":
		return cfg.Language, nil
	case "cppCompiler":
		return cfg.CPPCompiler, nil
	case "python":
		return cfg.Python, nil
	case "chromeBinary":
		return cfg.ChromeBinary, nil
	case "browserMode":
		return cfg.BrowserMode, nil
	case "programTypeId":
		return cfg.ProgramType, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func setConfigValue(cfg *config.Config, key, value string) error {
	switch key {
	case "handle":
		cfg.Handle = value
	case "language":
		if value != "cpp" && value != "python" {
			return errors.New("language must be cpp or python")
		}
		cfg.Language = value
	case "cppCompiler":
		cfg.CPPCompiler = value
	case "python":
		cfg.Python = value
	case "chromeBinary":
		cfg.ChromeBinary = value
	case "browserMode":
		if value != "dedicated" && value != "extension" {
			return errors.New("browserMode must be dedicated or extension")
		}
		cfg.BrowserMode = value
	case "programTypeId":
		cfg.ProgramType = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

type doctorResult struct {
	OK            bool   `json:"ok"`
	ConfigPath    string `json:"configPath"`
	Handle        string `json:"handle,omitempty"`
	Chrome        string `json:"chrome,omitempty"`
	BrowserMode   string `json:"browserMode"`
	Extension     string `json:"extension,omitempty"`
	Compiler      string `json:"compiler,omitempty"`
	Python        string `json:"python,omitempty"`
	CodeforcesAPI string `json:"codeforcesApi"`
}

func (c *command) doctor(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(c.err)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	path, _ := config.Path()
	result := doctorResult{OK: true, ConfigPath: path, Handle: cfg.Handle, BrowserMode: cfg.BrowserMode}
	if browser, err := codeforces.NewBrowser(cfg); err == nil {
		result.Chrome = browser.ChromeBinary
	} else {
		result.OK = false
	}
	if cfg.BrowserMode == "extension" {
		if path, err := extension.Path(); err == nil {
			result.Extension = path
		} else {
			result.OK = false
		}
	}
	if path, err := exec.LookPath(cfg.CPPCompiler); err == nil {
		result.Compiler = path
	} else {
		result.OK = false
	}
	if path, err := exec.LookPath(cfg.Python); err == nil {
		result.Python = path
	}
	if err := c.api.Ping(ctx); err != nil {
		result.CodeforcesAPI = err.Error()
		result.OK = false
	} else {
		result.CodeforcesAPI = "OK"
	}
	if *jsonOut {
		if err := writeJSON(c.out, result); err != nil {
			return err
		}
		if !result.OK {
			return errors.New("doctor found missing or unavailable dependencies")
		}
		return nil
	}
	fmt.Fprintf(c.out, "Codeforces API: %s\nChrome: %s\nBrowser mode: %s\nExtension: %s\nC++ compiler: %s\nPython: %s\nHandle: %s\nConfig: %s\n", result.CodeforcesAPI, blank(result.Chrome), result.BrowserMode, blank(result.Extension), blank(result.Compiler), blank(result.Python), blank(result.Handle), result.ConfigPath)
	if !result.OK {
		return errors.New("doctor found missing or unavailable dependencies")
	}
	return nil
}

func (c *command) auth(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: cfx auth")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	handle, err := c.authenticate(ctx, cfg)
	if err != nil {
		return err
	}
	cfg.Handle = handle
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "Authenticated as %s.\n", handle)
	return nil
}

func (c *command) extension(args []string) error {
	if len(args) != 1 || (args[0] != "install" && args[0] != "path") {
		return errors.New("usage: cfx extension install|path")
	}
	if args[0] == "path" {
		path, err := extension.Path()
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, path)
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	path, token, err := extension.Install(cfg.ExtensionToken)
	if err != nil {
		return err
	}
	cfg.BrowserMode = "extension"
	cfg.ExtensionToken = token
	if err := config.Save(cfg); err != nil {
		return err
	}
	if browser, err := codeforces.NewBrowser(cfg); err == nil {
		_ = exec.Command(browser.ChromeBinary, "chrome://extensions/").Start()
	}
	fmt.Fprintf(c.out, "Installed the unpacked extension at:\n%s\n\nIn chrome://extensions, enable Developer mode, choose Load unpacked, and select that directory. If cfx is already loaded, click Reload on its extension card. cfx now uses that Chrome profile.\n", path)
	return nil
}

func (c *command) authenticate(ctx context.Context, cfg config.Config) (string, error) {
	if cfg.BrowserMode == "extension" {
		fmt.Fprintln(c.err, "Waiting for the cfx Chrome extension...")
		data, err := c.runExtension(ctx, cfg, "whoami", nil)
		if err != nil {
			return "", err
		}
		var result struct {
			Handle string `json:"handle"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return "", fmt.Errorf("decode extension response: %w", err)
		}
		if result.Handle == "" {
			return "", errors.New("Chrome extension returned an empty Codeforces handle")
		}
		return result.Handle, nil
	}
	browser, err := codeforces.NewBrowser(cfg)
	if err != nil {
		return "", err
	}
	return browser.Auth(ctx)
}

func (c *command) sync(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(c.err)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	problems, err := c.api.Problems(ctx)
	if err != nil {
		return err
	}
	if err := saveCatalog(problems); err != nil {
		return err
	}
	contests, err := c.api.Contests(ctx)
	if err != nil {
		return err
	}
	if err := saveContests(contests); err != nil {
		return err
	}
	result := map[string]any{"problems": len(problems), "contests": len(contests), "syncedAt": time.Now()}
	if *jsonOut {
		return writeJSON(c.out, result)
	}
	fmt.Fprintf(c.out, "Synced %d problems and %d contests.\n", len(problems), len(contests))
	return nil
}

func (c *command) pick(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("pick", flag.ContinueOnError)
	fs.SetOutput(c.err)
	rating := fs.Int("rating", 0, "exact rating")
	minRating := fs.Int("min-rating", 0, "minimum rating")
	maxRating := fs.Int("max-rating", 0, "maximum rating")
	tag := fs.String("tag", "", "required tag")
	unsolved := fs.Bool("unsolved", false, "exclude accepted problems")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	problems, err := loadCatalog()
	if errors.Is(err, os.ErrNotExist) {
		problems, err = c.api.Problems(ctx)
		if err == nil {
			err = saveCatalog(problems)
		}
	}
	if err != nil {
		return err
	}
	solved := map[string]bool{}
	if *unsolved {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.Handle == "" {
			return errors.New("--unsolved requires `cfx config set handle YOUR_HANDLE`")
		}
		submissions, err := c.api.UserStatus(ctx, cfg.Handle, 1, 10000)
		if err != nil {
			return err
		}
		for _, submission := range submissions {
			if submission.Verdict == "OK" {
				solved[submission.Problem.ID()] = true
			}
		}
	}
	filtered := make([]model.Problem, 0)
	for _, problem := range problems {
		if problem.Type != "PROGRAMMING" || problem.Rating == 0 || solved[problem.ID()] {
			continue
		}
		if *rating != 0 && problem.Rating != *rating {
			continue
		}
		if *minRating != 0 && problem.Rating < *minRating {
			continue
		}
		if *maxRating != 0 && problem.Rating > *maxRating {
			continue
		}
		if *tag != "" && !contains(problem.Tags, *tag) {
			continue
		}
		filtered = append(filtered, problem)
	}
	if len(filtered) == 0 {
		return errors.New("no matching problems")
	}
	problem := filtered[randomIndex(len(filtered))]
	if *jsonOut {
		return writeJSON(c.out, problem)
	}
	fmt.Fprintf(c.out, "%s  %s  rating=%d  solved=%d  tags=%s\nhttps://codeforces.com/problemset/problem/%d/%s\n", problem.ID(), problem.Name, problem.Rating, problem.SolvedCount, strings.Join(problem.Tags, ","), problem.ContestID, problem.Index)
	return nil
}

func (c *command) start(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(c.err)
	dir := fs.String("dir", "", "workspace directory")
	lang := fs.String("lang", "", "cpp or python")
	jsonOut := fs.Bool("json", false, "JSON output")
	problemArg := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		problemArg = args[0]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if problemArg == "" && fs.NArg() == 1 {
		problemArg = fs.Arg(0)
	}
	if problemArg == "" || fs.NArg() != 0 {
		return errors.New("usage: cfx start <contest><index> [--dir PATH] [--lang cpp|python]")
	}
	contestID, index, err := parseProblemID(problemArg)
	if err != nil {
		return err
	}
	problems, err := loadCatalog()
	if errors.Is(err, os.ErrNotExist) {
		problems, err = c.api.Problems(ctx)
		if err == nil {
			err = saveCatalog(problems)
		}
	}
	if err != nil {
		return err
	}
	problem, ok := findProblem(problems, contestID, index)
	if !ok {
		return fmt.Errorf("problem %d%s not found in catalog; run `cfx sync`", contestID, index)
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
	statement, err := c.fetchStatement(ctx, cfg, problem)
	if err != nil {
		return err
	}
	target := *dir
	if target == "" {
		target = "cf-" + strings.ToLower(problem.ID())
	}
	session, err := workspace.Create(target, statement, language)
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(target)
	result := map[string]any{"directory": abs, "session": session, "samples": len(statement.Samples)}
	if *jsonOut {
		return writeJSON(c.out, result)
	}
	fmt.Fprintf(c.out, "Created %s with %d samples.\n", abs, len(statement.Samples))
	return nil
}

func (c *command) show(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(c.err)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, session, err := workspace.Find(".")
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(c.out, map[string]any{"directory": dir, "session": session})
	}
	fmt.Fprintf(c.out, "%s  %s  rating=%d\nSource: %s\nAttempts: %d\n", session.Problem.ID(), session.Problem.Name, session.Problem.Rating, filepath.Join(dir, session.Source), len(session.Attempts))
	return nil
}

func (c *command) test(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(c.err)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, session, err := workspace.Find(".")
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	result, runErr := runner.Run(ctx, cfg, dir, session)
	detail := "samples passed"
	if runErr != nil {
		detail = runErr.Error()
	} else if !result.Passed {
		detail = "sample mismatch"
	}
	_ = workspace.Record(dir, &session, model.Attempt{At: time.Now(), Kind: "test", Success: runErr == nil && result.Passed, Detail: detail})
	if runErr != nil {
		return runErr
	}
	if *jsonOut {
		_ = writeJSON(c.out, result)
	} else {
		printTestResult(c.out, result)
	}
	if !result.Passed {
		return errors.New("sample tests failed")
	}
	return nil
}

func (c *command) status(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(c.err)
	wait := fs.Bool("wait", false, "poll until judged")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, session, err := workspace.Find(".")
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Handle == "" {
		return errors.New("set handle with `cfx config set handle YOUR_HANDLE`")
	}
	var submission model.Submission
	for {
		submissions, err := c.api.UserStatus(ctx, cfg.Handle, 1, 20)
		if err != nil {
			return err
		}
		found := false
		for _, candidate := range submissions {
			if candidate.Problem.ID() == session.Problem.ID() {
				submission = candidate
				found = true
				break
			}
		}
		if !found {
			return errors.New("no submission found for this problem")
		}
		if !*wait || (submission.Verdict != "" && submission.Verdict != "TESTING") {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2100 * time.Millisecond):
		}
	}
	_ = workspace.Record(dir, &session, model.Attempt{At: time.Now(), Kind: "verdict", Success: submission.Verdict == "OK", Detail: fmt.Sprintf("%d %s", submission.ID, submission.Verdict)})
	if *jsonOut {
		return writeJSON(c.out, submission)
	}
	fmt.Fprintf(c.out, "%d  %s  tests=%d  time=%dms  memory=%dB\n", submission.ID, blank(submission.Verdict), submission.PassedTestCount, submission.TimeConsumedMillis, submission.MemoryConsumedBytes)
	return nil
}

func (c *command) submit(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	fs.SetOutput(c.err)
	yes := fs.Bool("yes", false, "confirm external submission")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return errors.New("submission changes your Codeforces account; rerun with --yes")
	}
	dir, session, err := workspace.Find(".")
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Handle == "" {
		return errors.New("set handle before submitting")
	}
	result, err := runner.Run(ctx, cfg, dir, session)
	if err != nil {
		return err
	}
	if !result.Passed {
		return errors.New("sample tests failed; submission refused")
	}
	source, err := os.ReadFile(filepath.Join(dir, session.Source))
	if err != nil {
		return err
	}
	programType := cfg.ProgramType
	if programType == "" {
		programType = defaultProgramType(session.Language)
	}
	if err := c.submitSource(ctx, cfg, session.Problem, string(source), programType); err != nil {
		return err
	}
	_ = workspace.Record(dir, &session, model.Attempt{At: time.Now(), Kind: "submit", Success: true, Detail: "sent to Codeforces"})
	resultOut := map[string]any{"submitted": true, "problem": session.Problem.ID(), "handle": cfg.Handle}
	if *jsonOut {
		return writeJSON(c.out, resultOut)
	}
	fmt.Fprintf(c.out, "Submitted %s as %s. Run `cfx status --wait`.\n", session.Problem.ID(), cfg.Handle)
	return nil
}

func (c *command) fetchStatement(ctx context.Context, cfg config.Config, problem model.Problem) (model.Statement, error) {
	if cfg.BrowserMode == "extension" {
		fmt.Fprintln(c.err, "Waiting for the cfx Chrome extension...")
		data, err := c.runExtension(ctx, cfg, "statement", map[string]any{
			"contestId": problem.ContestID,
			"index":     problem.Index,
		})
		if err != nil {
			return model.Statement{}, err
		}
		var result struct {
			HTML string `json:"html"`
			URL  string `json:"url"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return model.Statement{}, fmt.Errorf("decode extension response: %w", err)
		}
		return codeforces.ParseStatement(strings.NewReader(result.HTML), problem, result.URL)
	}
	browser, err := codeforces.NewBrowser(cfg)
	if err != nil {
		return model.Statement{}, err
	}
	return browser.FetchStatement(ctx, problem)
}

func (c *command) submitSource(ctx context.Context, cfg config.Config, problem model.Problem, source, programType string) error {
	if cfg.BrowserMode == "extension" {
		fmt.Fprintln(c.err, "Waiting for the cfx Chrome extension; the submission tab will be visible...")
		_, err := c.runExtension(ctx, cfg, "submit", map[string]any{
			"contestId":     problem.ContestID,
			"index":         problem.Index,
			"programTypeId": programType,
			"source":        source,
		})
		return err
	}
	browser, err := codeforces.NewBrowser(cfg)
	if err != nil {
		return err
	}
	return browser.Submit(ctx, problem, source, programType)
}

func (c *command) runExtension(ctx context.Context, cfg config.Config, jobType string, payload any) (json.RawMessage, error) {
	browser, err := codeforces.NewBrowser(cfg)
	if err != nil {
		return nil, err
	}
	go func() {
		timer := time.NewTimer(300 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			_ = exec.Command(browser.ChromeBinary, bridge.WakeURL).Start()
		}
	}()
	return bridge.Run(ctx, cfg.ExtensionToken, jobType, payload, 2*time.Minute)
}

type catalog struct {
	SyncedAt time.Time       `json:"syncedAt"`
	Problems []model.Problem `json:"problems"`
}

func saveCatalog(problems []model.Problem) error {
	path, err := config.CachePath("problems.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(catalog{SyncedAt: time.Now(), Problems: problems})
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func loadCatalog() ([]model.Problem, error) {
	path, err := config.CachePath("problems.json")
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data catalog
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data.Problems, nil
}

var problemIDPattern = regexp.MustCompile(`(?i)^(\d+)([a-z][a-z0-9]*)$`)

func parseProblemID(value string) (int, string, error) {
	match := problemIDPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return 0, "", fmt.Errorf("invalid problem id %q; expected e.g. 4A or 1795B", value)
	}
	id, _ := strconv.Atoi(match[1])
	return id, strings.ToUpper(match[2]), nil
}

func findProblem(problems []model.Problem, contestID int, index string) (model.Problem, bool) {
	for _, problem := range problems {
		if problem.ContestID == contestID && strings.EqualFold(problem.Index, index) {
			return problem, true
		}
	}
	return model.Problem{}, false
}

func randomIndex(limit int) int {
	var value uint64
	if err := binary.Read(rand.Reader, binary.LittleEndian, &value); err != nil {
		return int(time.Now().UnixNano() % int64(limit))
	}
	return int(value % uint64(limit))
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}

func defaultProgramType(language string) string {
	if language == "python" {
		return "70" // PyPy 3-64
	}
	return "91" // GNU G++23 14.2 (64 bit, msys2)
}
func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
func blank(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}

func printTestResult(w io.Writer, result runner.Result) {
	cases := append([]runner.CaseResult(nil), result.Cases...)
	sort.SliceStable(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	for _, item := range cases {
		status := "PASS"
		if !item.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "%s %s (%s)\n", status, item.Name, (time.Duration(item.DurationMillis) * time.Millisecond).String())
		if !item.Passed {
			fmt.Fprintf(w, "expected:\n%sactual:\n%s", item.Expected, item.Actual)
			if item.Error != "" {
				fmt.Fprintf(w, "error: %s\n", item.Error)
			}
		}
	}
}
