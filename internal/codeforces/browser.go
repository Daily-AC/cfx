package codeforces

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Daily-AC/cfx/internal/config"
	"github.com/Daily-AC/cfx/internal/model"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/net/html"
)

type Browser struct {
	ChromeBinary string
	ProfileDir   string
	Headless     bool
}

func NewBrowser(cfg config.Config) (*Browser, error) {
	profile, err := config.ChromeProfile()
	if err != nil {
		return nil, err
	}
	bin := cfg.ChromeBinary
	if bin == "" {
		bin = findChrome()
	}
	if bin == "" {
		return nil, errors.New("Chrome not found; set chromeBinary with `cfx config set chromeBinary PATH`")
	}
	return &Browser{ChromeBinary: bin, ProfileDir: profile}, nil
}

func (b *Browser) Auth(ctx context.Context) (string, error) {
	page, cleanup, err := b.open(ctx, "https://codeforces.com/enter?back=%2Fproblemset")
	if err != nil {
		return "", err
	}
	defer cleanup()
	info, err := page.Info()
	if err != nil {
		return "", err
	}
	if strings.Contains(info.URL, "/enter") {
		fmt.Fprintln(os.Stderr, "Chrome opened with the cfx profile. Complete Codeforces login, then press Enter here.")
		var line string
		_, _ = fmt.Scanln(&line)
		info, err = page.Info()
		if err != nil {
			return "", err
		}
		if strings.Contains(info.URL, "/enter") {
			return "", errors.New("Codeforces still shows the login page")
		}
	}
	links, err := page.Elements(`.lang-chooser a[href^="/profile/"]`)
	if err != nil {
		return "", fmt.Errorf("find logged-in handle: %w", err)
	}
	for _, link := range links {
		href, err := link.Attribute("href")
		if err != nil || href == nil {
			continue
		}
		if handle := handleFromProfileHref(*href); handle != "" {
			return handle, nil
		}
	}
	return "", errors.New("login succeeded but the Codeforces handle was not found")
}

func handleFromProfileHref(href string) string {
	path := strings.TrimPrefix(strings.TrimSpace(href), "/profile/")
	if path == href || path == "" || strings.Contains(path, "/") {
		return ""
	}
	return path
}

func (b *Browser) FetchStatement(ctx context.Context, problem model.Problem) (model.Statement, error) {
	u := fmt.Sprintf("https://codeforces.com/problemset/problem/%d/%s?locale=en", problem.ContestID, problem.Index)
	page, cleanup, err := b.open(ctx, u)
	if err != nil {
		return model.Statement{}, err
	}
	defer cleanup()
	if err := page.Timeout(30*time.Second).WaitElementsMoreThan(".problem-statement", 0); err != nil {
		info, _ := page.Info()
		return model.Statement{}, fmt.Errorf("problem statement unavailable at %s; run `cfx auth` and retry: %w", info.URL, err)
	}
	html, err := page.HTML()
	if err != nil {
		return model.Statement{}, err
	}
	return ParseStatement(strings.NewReader(html), problem, u)
}

func (b *Browser) Submit(ctx context.Context, problem model.Problem, source, programType string) error {
	u := fmt.Sprintf("https://m1.codeforces.com/contest/%d/submit", problem.ContestID)
	page, cleanup, err := b.open(ctx, u)
	if err != nil {
		return err
	}
	defer cleanup()
	if strings.Contains(page.MustInfo().URL, "/enter") {
		return errors.New("Codeforces login required; run `cfx auth`")
	}
	index, err := page.Timeout(30 * time.Second).Element("[name=problemIndex]")
	if err != nil {
		return fmt.Errorf("submission form did not load: %w", err)
	}
	if tag := index.MustEval(`() => this.tagName`).String(); strings.EqualFold(tag, "select") {
		if err := index.Select([]string{fmt.Sprintf(`[value=%q]`, problem.Index)}, true, rod.SelectorTypeCSSSector); err != nil {
			return fmt.Errorf("select problem %s: %w", problem.Index, err)
		}
	} else {
		if err := index.SelectAllText(); err == nil {
			_ = index.Input("")
		}
		if err := index.Input(problem.Index); err != nil {
			return fmt.Errorf("set problem %s: %w", problem.Index, err)
		}
	}
	language, err := page.Element("select[name=programTypeId]")
	if err != nil {
		return err
	}
	if err := language.Select([]string{fmt.Sprintf(`[value=%q]`, programType)}, true, rod.SelectorTypeCSSSector); err != nil {
		return fmt.Errorf("select Codeforces language id %s: %w", programType, err)
	}
	textarea, err := page.Element("textarea[name=source]")
	if err != nil {
		return err
	}
	if err := textarea.SelectAllText(); err == nil {
		_ = textarea.Input("")
	}
	if err := textarea.Input(source); err != nil {
		return err
	}
	button, err := page.Element("form button[type=submit], form input[type=submit]")
	if err != nil {
		return err
	}
	if err := button.Click("left", 1); err != nil {
		return err
	}
	page.Timeout(30 * time.Second).MustWaitLoad()
	if strings.Contains(page.MustInfo().URL, "/submit") {
		if message, err := page.Element(".error"); err == nil {
			return fmt.Errorf("Codeforces rejected submission: %s", strings.TrimSpace(message.MustText()))
		}
		return errors.New("submission stayed on the form without a visible error")
	}
	return nil
}

func (b *Browser) open(ctx context.Context, target string) (*rod.Page, func(), error) {
	if err := os.MkdirAll(b.ProfileDir, 0o700); err != nil {
		return nil, nil, err
	}
	if err := clearStaleProfileLock(b.ProfileDir); err != nil {
		return nil, nil, err
	}
	l := launcher.New().Context(ctx).Bin(b.ChromeBinary).UserDataDir(b.ProfileDir).Headless(b.Headless)
	l.Set("no-first-run").Set("disable-background-networking")
	controlURL, err := l.Launch()
	if err != nil {
		if strings.Contains(err.Error(), "SingletonLock") || strings.Contains(err.Error(), "ProcessSingleton") {
			return nil, nil, errors.New("the dedicated cfx Chrome profile is already in use; close its Chrome window, or run `cfx extension install` to use your existing Chrome profile")
		}
		return nil, nil, fmt.Errorf("launch Chrome: %w", err)
	}
	browser := rod.New().Context(ctx).ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Kill()
		return nil, nil, fmt.Errorf("connect Chrome: %w", err)
	}
	page, err := browser.Page(proto.TargetCreateTarget{URL: target})
	if err != nil {
		_ = browser.Close()
		l.Kill()
		return nil, nil, err
	}
	cleanup := func() {
		_ = browser.Close()
		l.Kill()
	}
	return page, cleanup, nil
}

func clearStaleProfileLock(profile string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	target, err := os.Readlink(filepath.Join(profile, "SingletonLock"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Chrome profile lock: %w", err)
	}
	host, err := os.Hostname()
	if err != nil {
		return nil
	}
	prefix := host + "-"
	if !strings.HasPrefix(target, prefix) {
		return nil
	}
	pid, err := strconv.Atoi(strings.TrimPrefix(target, prefix))
	if err != nil || pid <= 0 {
		return nil
	}
	if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err == nil {
		return nil
	}
	for _, name := range []string{"SingletonLock", "SingletonCookie", "SingletonSocket"} {
		path := filepath.Join(profile, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale Chrome profile lock %s: %w", path, err)
		}
	}
	return nil
}

func findChrome() string {
	candidates := []string{"google-chrome", "chromium", "chromium-browser"}
	switch runtime.GOOS {
	case "darwin":
		candidates = append([]string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}, candidates...)
	case "windows":
		for _, base := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
			if base != "" {
				candidates = append([]string{filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe")}, candidates...)
			}
		}
	}
	for _, candidate := range candidates {
		if filepath.IsAbs(candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

func ParseStatement(r *strings.Reader, problem model.Problem, sourceURL string) (model.Statement, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return model.Statement{}, err
	}
	root := doc.Find(".problem-statement").First()
	if root.Length() == 0 {
		return model.Statement{}, errors.New("HTML has no .problem-statement")
	}
	normalizeMath(root)
	statement := model.Statement{Problem: problem, URL: sourceURL}
	if title := clean(root.Find(".title").First().Text()); title != "" {
		if _, rest, ok := strings.Cut(title, ". "); ok {
			statement.Name = rest
		}
	}
	statement.TimeLimit = trimLabel(root.Find(".time-limit").First().Text(), "time limit per test")
	statement.MemoryLimit = trimLabel(root.Find(".memory-limit").First().Text(), "memory limit per test")
	statement.Body = selectionText(root.ChildrenFiltered("div").Not(".header,.input-specification,.output-specification,.sample-tests,.note"))
	statement.Input = selectionText(root.Find(".input-specification").First().Children().Not(".section-title"))
	statement.Output = selectionText(root.Find(".output-specification").First().Children().Not(".section-title"))
	statement.Note = selectionText(root.Find(".note").First().Children().Not(".section-title"))
	inputs := root.Find(".sample-tests .input pre")
	outputs := root.Find(".sample-tests .output pre")
	inputs.Each(func(i int, input *goquery.Selection) {
		output := ""
		if i < outputs.Length() {
			output = preText(outputs.Eq(i))
		}
		statement.Samples = append(statement.Samples, model.Sample{Input: preText(input), Output: output})
	})
	if len(statement.Samples) == 0 {
		return model.Statement{}, errors.New("problem statement has no samples")
	}
	return statement, nil
}

func normalizeMath(root *goquery.Selection) {
	root.Find("span.MathJax").Each(func(_ int, rendered *goquery.Selection) {
		script := rendered.NextFiltered(`script[type="math/tex"]`).First()
		if script.Length() == 0 {
			return
		}
		tex := strings.TrimSpace(script.Text())
		rendered.ReplaceWithNodes(&html.Node{Type: html.TextNode, Data: "$" + tex + "$"})
	})
	root.Find(`script[type="math/tex"], .MathJax_Preview, .MJX_Assistive_MathML`).Remove()
}

func selectionText(s *goquery.Selection) string {
	parts := make([]string, 0, s.Length())
	s.Each(func(_ int, item *goquery.Selection) {
		name := goquery.NodeName(item)
		blocks := item.ChildrenFiltered("p,ul,ol,pre")
		if name == "p" || name == "ul" || name == "ol" || name == "pre" {
			blocks = item
		}
		if blocks.Length() == 0 {
			return
		}
		blocks.Each(func(_ int, block *goquery.Selection) {
			text := ""
			if goquery.NodeName(block) == "ul" || goquery.NodeName(block) == "ol" {
				items := make([]string, 0)
				block.ChildrenFiltered("li").Each(func(_ int, item *goquery.Selection) {
					items = append(items, "- "+clean(item.Text()))
				})
				text = strings.Join(items, "\n")
			} else {
				text = clean(block.Text())
			}
			if text != "" {
				parts = append(parts, text)
			}
		})
	})
	return strings.Join(parts, "\n\n")
}

func preText(s *goquery.Selection) string {
	if s.Length() == 0 || s.Get(0) == nil {
		return ""
	}
	var b strings.Builder
	for node := s.Get(0).FirstChild; node != nil; node = node.NextSibling {
		renderPreNode(&b, node)
	}
	return strings.TrimSpace(strings.ReplaceAll(b.String(), "\r\n", "\n")) + "\n"
}

func renderPreNode(b *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		b.WriteString(node.Data)
		return
	}
	if node.Type != html.ElementNode {
		return
	}
	if strings.EqualFold(node.Data, "br") {
		b.WriteByte('\n')
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		renderPreNode(b, child)
	}
	if strings.EqualFold(node.Data, "div") && b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
		b.WriteByte('\n')
	}
}

func clean(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func trimLabel(value, label string) string {
	return strings.TrimSpace(strings.TrimPrefix(clean(value), label))
}
