package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/charmbracelet/x/ansi"
)

// S3: "debe todo cambiar con respecto al color del tema". The icons, the hover box, the selection, the borders, the
// branch and the git letters take another color in each theme.
func TestEveryPartChangesColorWithTheTheme(t *testing.T) {
	type look map[string]string
	read := func(name string) look {
		e := projectEnv(t, 60)
		e.key("ctrl+b") // the window: borders, branch and toolbar
		d := e.m.deps
		d.Cfg.Theme = name
		e.m.deps = d
		e.m.restyle()
		e.m.icons = icons.Set{Style: icons.Nerd}
		e.touch("x.go", "package x")
		e.touch("doc.pdf", "x")
		e.m.reload()
		e.send(tea.WindowSizeMsg{Width: 120, Height: 30})
		raw := strings.Split(e.m.View().Content, "\n")
		at := func(text string) (string, int) {
			for _, l := range raw {
				if i := strings.Index(ansi.Strip(l), text); i >= 0 {
					return l, ansi.StringWidth(ansi.Strip(l)[:i])
				}
			}
			t.Fatalf("%s: no %q", name, text)
			return "", 0
		}
		colors := func(line string, x int) (fg, bg string) {
			col := 0
			for _, sg := range colorSegments(line) {
				w := ansi.StringWidth(sg.text)
				if x < col+w {
					return fmt.Sprint(sg.fg), fmt.Sprint(sg.bg)
				}
				col += w
			}
			return "", ""
		}
		l := look{}
		line, x := at("x.go")
		l["go icon"], _ = colors(line, x-2)
		line, x = at("doc.pdf")
		l["pdf icon"], _ = colors(line, x-2)
		line, x = at("alpha")
		l["folder"], _ = colors(line, x)
		_, l["selection"] = colors(raw[e.m.firstRow()+e.m.sel-e.m.top], e.m.listX()+2)
		l["border"], _ = colors(raw[1], e.m.listX()-1)
		line, x = at("main")
		l["branch"], _ = colors(line, x)
		a := e.m.actionSpans()[0]
		e.move(a.x+1, a.y)
		_, l["hover"] = colors(strings.Split(e.m.View().Content, "\n")[a.y], a.x)
		return l
	}
	themes := []string{"catppuccin-mocha", "catppuccin-latte", "nord"}
	looks := map[string]look{}
	for _, th := range themes {
		looks[th] = read(th)
	}
	for part := range looks[themes[0]] {
		seen := map[string]string{}
		for _, th := range themes {
			c := looks[th][part]
			if c == "" || c == "<nil>" {
				t.Errorf("%s: %s has no color of its own", th, part)
				continue
			}
			if other, dup := seen[c]; dup {
				t.Errorf("%s keeps the color %s in %s and %s: it does not follow the theme", part, c, other, th)
			}
			seen[c] = th
		}
	}
}
