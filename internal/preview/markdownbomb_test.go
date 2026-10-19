package preview

import (
	"strings"
	"testing"
	"time"
)

// Nested quotes made glamour's output grow exponentially: 80 > took 37 s and 84 took 5 GB, and the preview could not
// be cancelled. Markdown nested deeper than maxMarkdownDepth shows as its source; shallower nesting still renders.
func TestDeeplyNestedMarkdownShowsItsSource(t *testing.T) {
	var indented strings.Builder // lists nested by indentation alone
	for i := range 600 {
		indented.WriteString(strings.Repeat("  ", i) + "- x\n")
	}
	for _, src := range []string{strings.Repeat(">", 200), strings.Repeat("> - ", 300) + "x", strings.Repeat(">", 512<<10), indented.String()} {
		start := time.Now()
		lines, ok := renderMarkdown(src, 20, false)
		if took := time.Since(start); took > 2*time.Second || ok {
			t.Fatalf("%.10q…: rendered %v in %v (%d lines)", src, ok, took, len(lines))
		}
	}
	for _, width := range []int{20, 80} {
		start := time.Now()
		if _, ok := renderMarkdown(strings.Repeat("> - ", maxMarkdownDepth/2)+"x", width, false); !ok {
			t.Fatal("nesting within the limit must render")
		}
		if took := time.Since(start); took > time.Second {
			t.Fatalf("width %d: nesting within the limit took %v", width, took)
		}
	}
	fenced := "# Title\n\n```yaml\n" + strings.Repeat(" ", 80) + "- deep: yaml\n```\n"
	if _, ok := renderMarkdown(fenced, 80, false); !ok {
		t.Fatal("deep indentation inside a fenced block must still render")
	}
}
