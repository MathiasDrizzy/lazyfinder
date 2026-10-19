package preview

import (
	"strings"

	"charm.land/glamour/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// maxMarkdownDepth is the deepest nesting of quotes and lists that is rendered. glamour's output grows exponentially
// with it (80 nested quotes took 37 s, 84 took 5 GB) and takes no context, so deeper Markdown shows as its source.
const maxMarkdownDepth = 12

// markersTooDeep tells, in linear time, whether a line outside a fenced code block starts with more quote and list
// markers (>, -, *, +, 1. or 1)) than maxMarkdownDepth, or is indented further than that many lists can indent it.
func markersTooDeep(src string) bool {
	fenced := false
	for line := range strings.Lines(src) {
		if f := strings.TrimLeft(line, " \t"); strings.HasPrefix(f, "```") || strings.HasPrefix(f, "~~~") {
			fenced = !fenced
			continue
		}
		if !fenced && lineTooDeep(line) {
			return true
		}
	}
	return false
}

// maxMarkdownIndent is the widest indentation, in columns, that maxMarkdownDepth nested lists need: a marker such as
// "10. " and the four columns of an indented code block inside the deepest one.
const maxMarkdownIndent = 4*maxMarkdownDepth + 4

func lineTooDeep(line string) bool {
	depth, indent := 0, 0
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case c == ' ':
			indent++
		case c == '\t':
			indent += 4
		case c == '>':
			depth++
		case (c == '-' || c == '*' || c == '+') && i+1 < len(line) && (line[i+1] == ' ' || line[i+1] == '\t'):
			depth++
		case c >= '0' && c <= '9':
			j := i
			for j < len(line) && line[j] >= '0' && line[j] <= '9' {
				j++
			}
			if j+1 >= len(line) || line[j] != '.' && line[j] != ')' || line[j+1] != ' ' && line[j+1] != '\t' {
				return false
			}
			depth++
			i = j
		default:
			return false
		}
		if depth > maxMarkdownDepth || indent > maxMarkdownIndent {
			return true
		}
	}
	return false
}

// tooDeep tells whether src nests quotes and lists deeper than maxMarkdownDepth. Parsing takes linear time; the walk
// keeps its own stack and stops at the first node past the limit.
func tooDeep(src string) bool {
	if markersTooDeep(src) { // goldmark itself is quadratic in the nesting: 512 KB of > took 84 s to parse
		return true
	}
	type at struct {
		n     ast.Node
		depth int
	}
	stack := []at{{goldmark.DefaultParser().Parse(text.NewReader([]byte(src))), 0}}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		depth := top.depth
		switch top.n.Kind() {
		case ast.KindBlockquote, ast.KindList:
			depth++
		}
		if depth > maxMarkdownDepth {
			return true
		}
		for c := top.n.FirstChild(); c != nil; c = c.NextSibling() {
			stack = append(stack, at{c, depth})
		}
	}
	return false
}

// renderMarkdown turns Markdown into styled lines wrapped at width. Any failure returns ok=false and the caller
// shows the source instead.
func renderMarkdown(src string, width int, noColor bool) (lines []string, ok bool) {
	if tooDeep(src) {
		return nil, false
	}
	style := "dark"
	if noColor {
		style = "notty"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(max(20, width)))
	if err != nil {
		return nil, false
	}
	out, err := r.Render(src)
	if err != nil {
		return nil, false
	}
	return strings.Split(strings.Trim(out, "\n"), "\n"), true
}
