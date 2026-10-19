package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

// minContrast is the WCAG floor for bold text and interface parts (3:1).
const minContrast = 3.0

type segment struct {
	text   string
	fg, bg color.Color
}

// colorSegments splits a line with truecolor SGR sequences into runs of text with their colors.
func colorSegments(line string) []segment {
	var out []segment
	var fg, bg color.Color
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, segment{buf.String(), fg, bg})
			buf.Reset()
		}
	}
	for i := 0; i < len(line); {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && (line[j] == ';' || line[j] == ':' || (line[j] >= '0' && line[j] <= '9')) {
				j++
			}
			if j < len(line) && line[j] == 'm' {
				flush()
				fg, bg = applySGR(line[i+2:j], fg, bg)
			}
			for j < len(line) && !(line[j] >= '@' && line[j] <= '~') {
				j++
			}
			i = j + 1
			continue
		}
		buf.WriteByte(line[i])
		i++
	}
	flush()
	return out
}

func applySGR(params string, fg, bg color.Color) (color.Color, color.Color) {
	parts := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(parts) == 0 {
		return nil, nil
	}
	num := func(k int) uint8 {
		if k >= len(parts) {
			return 0
		}
		n, _ := strconv.Atoi(parts[k])
		return uint8(n)
	}
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "0", "00":
			fg, bg = nil, nil
		case "39":
			fg = nil
		case "49":
			bg = nil
		case "38", "48":
			isFg := parts[i] == "38"
			var c color.Color
			switch {
			case i+1 < len(parts) && parts[i+1] == "2":
				c = color.RGBA{R: num(i + 2), G: num(i + 3), B: num(i + 4), A: 255}
				i += 4
			case i+1 < len(parts) && parts[i+1] == "5":
				i += 2
			}
			if isFg {
				fg = c
			} else {
				bg = c
			}
		}
	}
	return fg, bg
}

func contrast(a, b color.Color) float64 { return theme.Contrast(a, b) }

// hasWords reports whether s has letters or digits: runs of box-drawing glyphs and icons are decoration.
func hasWords(s string) bool {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n >= 2
}

// TestTextReadsInEveryTheme draws the main screens in every built-in theme and checks each run of text against the
// background it sits on: at least 3:1, the WCAG floor for bold text and interface parts.
func TestTextReadsInEveryTheme(t *testing.T) {
	screens := []struct {
		name string
		keys []string
	}{{"list", nil}, {"columns", []string{"M"}}, {"search", []string{"/", "n", "o"}}, {"info", []string{"i"}}, {"preview", []string{"space"}},
		{"settings", []string{","}}, {"open with", []string{"O"}}, {"recents", []string{"E"}}, {"marked", []string{"v", "v"}},
		{"help", []string{"?"}}, {"sidebar", []string{"ctrl+b"}}, {"sidebar tree", []string{"ctrl+b", "j", "l"}}}
	var failures []string
	for _, name := range theme.ThemeNames() {
		for _, v := range screens {
			e := newEnv(t)
			d := e.m.deps
			d.Cfg.Theme = name
			e.m = New(d)
			e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
			for _, k := range v.keys {
				e.key(k)
			}
			seen := map[string]bool{}
			for _, line := range strings.Split(e.m.View().Content, "\n") {
				for _, sg := range colorSegments(line) {
					txt := strings.TrimSpace(ansi.Strip(sg.text))
					if sg.fg == nil || sg.bg == nil || !hasWords(txt) {
						continue
					}
					if r := contrast(sg.fg, sg.bg); r < minContrast && !seen[txt] {
						seen[txt] = true
						failures = append(failures, fmt.Sprintf("%-22s %-10s %.1f:1 %q", name, v.name, r, txt))
					}
				}
			}
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		shown := failures[:min(len(failures), 300)]
		t.Errorf("%d runs of text below %.0f:1:\n%s", len(failures), minContrast, strings.Join(shown, "\n"))
	}
}
