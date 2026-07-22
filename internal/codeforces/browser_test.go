package codeforces

import (
	"strings"
	"testing"

	"github.com/e0-7/cfx/internal/model"
)

func TestParseStatement(t *testing.T) {
	const page = `<!doctype html><div class="problem-statement">
<div class="header"><div class="title">A. Watermelon</div><div class="time-limit">time limit per test 1 second</div><div class="memory-limit">memory limit per test 64 megabytes</div></div>
<div><p>One hot summer day Pete and Billy bought a <span class="MathJax"><span>w</span><span class="MJX_Assistive_MathML">w</span></span><script type="math/tex">w</script> kilo watermelon.</p><p>Determine whether it can be split.</p></div>
<div class="input-specification"><div class="section-title">Input</div><p>The only line contains w.</p></div>
<div class="output-specification"><div class="section-title">Output</div><p>Print YES or NO.</p></div>
<div class="sample-tests"><div class="input"><pre><div>8</div><div>10</div></pre></div><div class="output"><pre>YES<br />YES<br /></pre></div></div>
<div class="note"><div class="section-title">Note</div><p>Two and six works.</p></div>
</div>`
	problem := model.Problem{ContestID: 4, Index: "A", Name: "Watermelon", Rating: 800}
	statement, err := ParseStatement(strings.NewReader(page), problem, "https://codeforces.com/problemset/problem/4/A")
	if err != nil {
		t.Fatal(err)
	}
	if statement.Name != "Watermelon" || statement.TimeLimit != "1 second" || len(statement.Samples) != 1 {
		t.Fatalf("unexpected statement: %+v", statement)
	}
	if !strings.Contains(statement.Body, "$w$ kilo") || strings.Contains(statement.Body, "ww") {
		t.Fatalf("unexpected math rendering: %s", statement.Body)
	}
	if statement.Samples[0].Input != "8\n10\n" || statement.Samples[0].Output != "YES\nYES\n" {
		t.Fatalf("unexpected sample: %#v", statement.Samples[0])
	}
}
