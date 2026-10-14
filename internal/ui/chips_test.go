package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/charmbracelet/x/ansi"
)

// Mathias, 2026-10-14 (order 07, revision 2, S1): "los recuadros no están al medio del icono … también esos recuadros
// se superponen, debe haber espacio para que no se superpongan como en el buscar". Every button of the top row is a
// chip whose hover box has the same margin on both sides of the glyph, a Nerd Font glyph being drawn two cells wide,
// and one empty cell between one box and the next.
func TestTheButtonBoxesAreCenteredOnTheirGlyphAndApart(t *testing.T) {
	for _, style := range []icons.Style{icons.Nerd, icons.ASCII} {
		for _, sidebar := range []bool{false, true} {
			var e *env
			width := 120
			if sidebar {
				e, width = projectEnv(t, 60), 60
			} else {
				e = newEnv(t)
			}
			e.m.icons = icons.Set{Style: style}
			e.send(tea.WindowSizeMsg{Width: width, Height: 30})
			if e.m.projectMode() != sidebar {
				t.Fatalf("setup: sidebar=%v", sidebar)
			}
			e.move(10, 8)
			type box struct {
				name  string
				r     hoverRect
				glyph string
			}
			var boxes []box
			for _, a := range e.m.actionSpans() {
				boxes = append(boxes, box{"action", hoverRect{a.x, a.y, a.w, 1}, e.m.actionGlyph(a.idx)})
			}
			for _, b := range []struct {
				name  string
				z     zoneKind
				glyph string
			}{{"mode", zoneMode, e.m.modeGlyph()}, {"gear", zoneSettings, e.m.gearText()}} {
				r, ok := e.m.zoneRect(zone{kind: b.z})
				if !ok {
					t.Fatalf("no %s box", b.name)
				}
				boxes = append(boxes, box{b.name, r, b.glyph})
			}
			if !sidebar {
				r, _ := e.m.zoneRect(zone{kind: zoneSearch})
				boxes = append(boxes, box{"search", r, ""})
			}
			label := map[bool]string{false: "window", true: "sidebar"}[sidebar]
			for _, b := range boxes {
				if b.glyph == "" {
					continue
				}
				line := e.screen()[b.r.y]
				cells := []rune(line)
				g := -1
				for x := b.r.x; x < b.r.x+b.r.w && x < len(cells); x++ {
					if string(cells[x]) == b.glyph {
						g = x
					}
				}
				if g < 0 {
					t.Fatalf("%s %v: the %s glyph %q is not inside its box %+v: %q", label, style, b.name, b.glyph, b.r, line)
				}
				vw := ansi.StringWidth(b.glyph)
				if r := []rune(b.glyph)[0]; style == icons.Nerd && vw == 1 && (r >= 0xe000 && r <= 0xf8ff || r >= 0xf0000) {
					vw = 2 // the Nerd Font glyphs are drawn two cells wide
				}
				if left, right := g-b.r.x, b.r.x+b.r.w-(g+vw); left != right || left < 1 {
					t.Errorf("%s %v: the %s box %+v has margins %d and %d around its glyph at %d", label, style, b.name, b.r, left, right, g)
				}
			}
			sort.Slice(boxes, func(i, j int) bool { return boxes[i].r.x < boxes[j].r.x })
			for i := 1; i < len(boxes); i++ {
				a, b := boxes[i-1], boxes[i]
				if a.r.y == b.r.y && a.r.x+a.r.w >= b.r.x {
					t.Errorf("%s %v: the %s box %+v touches the %s box %+v", label, style, a.name, a.r, b.name, b.r)
				}
			}
		}
	}
}

// S3: the buttons wear the same color in both modes, the dimmed one of the herdr sidebar, not the accent.
func TestTheButtonsHaveTheSameColorInBothModes(t *testing.T) {
	w := newEnv(t)
	w.send(tea.WindowSizeMsg{Width: 120, Height: 30})
	s := projectEnv(t, 60)
	s.move(10, 8)
	if got, want := w.m.actionsText(), s.m.actionsText(); got != want {
		t.Fatalf("the action icons differ between the window and the sidebar:\n%q\n%q", got, want)
	}
	if got, want := w.m.gearButton(), s.m.gearButton(); got != want {
		t.Fatalf("the gear differs: %q / %q", got, want)
	}
	if !strings.Contains(w.m.actionsText(), w.m.styles.dim(" ")[:4]) && w.m.styles.dim("x") != "x" {
		t.Fatalf("the buttons are dimmed: %q", w.m.actionsText())
	}
}

// S3: no color is written down outside the theme package: every color comes from the palette of the theme in use, so
// changing the theme changes everything. termimg is out: its "38;2;" carries the id of an image, not a color.
func TestNoColorIsWrittenOutsideTheThemePackage(t *testing.T) {
	literal := regexp.MustCompile(`#[0-9a-fA-F]{6}\b|0x[0-9a-fA-F]{6}\b|\b38;2;|\b48;2;|\b38;5;|\b48;5;`)
	root := filepath.Join("..", "..")
	var found []string
	for _, dir := range []string{"internal", "cmd"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			if d.IsDir() && (rel == filepath.Join("internal", "theme") || rel == filepath.Join("internal", "termimg")) {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.BasicLit:
					if literal.MatchString(v.Value) {
						found = append(found, fset.Position(v.Pos()).String()+" "+v.Value)
					}
				case *ast.SelectorExpr:
					if x, ok := v.X.(*ast.Ident); ok && x.Name == "lipgloss" && (v.Sel.Name == "Color" || v.Sel.Name == "ANSIColor") {
						found = append(found, fset.Position(v.Pos()).String()+" lipgloss."+v.Sel.Name)
					}
				}
				return true
			})
			return nil
		})
	}
	if len(found) > 0 {
		t.Fatalf("%d colors written down outside internal/theme:\n%s", len(found), strings.Join(found, "\n"))
	}
}
