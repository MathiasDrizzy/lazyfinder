package preview

import (
	"context"
	"strings"
	"testing"
	"time"
)

// glamour took minutes or gigabytes on a few hundred KB of nested quotes, unclosed links or a table of thousands of
// columns, and nested emphasis overflowed the stack, which killed the program; guessing those from the input missed
// some and refused good files. The render runs in a child process with limits: each of these ends quickly as source.
func TestMarkdownBombsShowTheirSource(t *testing.T) {
	bombs := map[string]string{
		"nested quotes":   strings.Repeat(">", 200),
		"quoted lists":    strings.Repeat("> - ", 300) + "x",
		"a fake fence":    "    ```\n" + strings.Repeat(">", 200<<10),
		"nested emphasis": strings.Repeat("**a ", 40000) + strings.Repeat("a** ", 40000),
		"unclosed links":  strings.Repeat("[a](", 50000),
		"a wide table":    "|" + strings.Repeat("a|", 50000) + "\n|" + strings.Repeat("-|", 50000) + "\n|" + strings.Repeat("b|", 50000) + "\n",
	}
	for name, src := range bombs {
		start := time.Now()
		lines, ok := renderMarkdown(context.Background(), src, 80, false)
		if took := time.Since(start); took > markdownTimeout+time.Second || ok {
			t.Errorf("%s: rendered %v in %v (%d lines)", name, ok, took, len(lines))
		}
	}
}

// The guesses refused good files: a blank line with trailing spaces, a rule written "- - - …", code indented deep in
// an indented block or a fence. They render.
func TestOrdinaryMarkdownRenders(t *testing.T) {
	good := map[string]string{
		"trailing spaces": "# Title\n\ntext\n" + strings.Repeat(" ", 60) + "\nmore\n",
		"a spaced rule":   "# T\n\n" + strings.Repeat("- ", 20) + "\n",
		"indented code":   "# T\n\n" + strings.Repeat(" ", 70) + "code\n",
		"a fenced block":  "# Title\n\n```yaml\n" + strings.Repeat(" ", 80) + "- deep: yaml\n```\n",
		"some nesting":    strings.Repeat("> - ", 4) + "x\n",
	}
	for name, src := range good {
		if lines, ok := renderMarkdown(context.Background(), src, 80, false); !ok || len(lines) == 0 {
			t.Errorf("%s did not render", name)
		}
	}
}

// Moving on cancels the render: the child is killed with the preview's context.
func TestACancelledRenderStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := renderMarkdown(ctx, "# Title\n", 80, false); ok {
		t.Fatal("a cancelled render must not count")
	}
}
