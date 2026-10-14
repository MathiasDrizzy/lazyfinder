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
)

// Mathias, 2026-10-14 (order 07, revision 2, S1): "los recuadros no están al medio del icono … se superponen", and
// after trying the four-cell boxes one apart: "ahora están muy separados, los recuadros son muy grandes … comparado a
// sidebar de herdr, ellos sí lograron generar el recuadro perfecto y que quede al medio el icono". Every button is
// herdr's title_action_chip, " g ": the glyph in the middle of three cells, so a Nerd Font glyph, drawn wider than its
// cell, spills into the right margin and sits in the middle. The action buttons follow each other with no gap, as in
// herdr; one free cell parts the groups (the actions, Search, the mode button, the gear); no two boxes overlap.
func TestTheButtonBoxesAreHerdrChipsWithTheGlyphInTheMiddle(t *testing.T) {
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
			spans := e.m.actionSpans()
			for _, a := range spans {
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
				line := []rune(e.screen()[b.r.y])
				if b.r.w != 3 || b.r.x+1 >= len(line) || string(line[b.r.x+1]) != b.glyph {
					t.Errorf("%s %v: the %s box %+v is not \" %s \" (three cells, the glyph in the middle): %q", label, style, b.name, b.r, b.glyph, string(line))
				}
			}
			for i := 1; i < len(spans); i++ {
				if gap := spans[i].x - (spans[i-1].x + spans[i-1].w); gap != 0 {
					t.Errorf("%s %v: action buttons %d and %d are %d cells apart, want 0 as in herdr", label, style, i-1, i, gap)
				}
			}
			sort.Slice(boxes, func(i, j int) bool { return boxes[i].r.x < boxes[j].r.x })
			for i := 1; i < len(boxes); i++ {
				a, b := boxes[i-1], boxes[i]
				if a.r.y != b.r.y {
					continue
				}
				if gap := b.r.x - (a.r.x + a.r.w); gap < 0 || a.name != b.name && gap != 1 {
					t.Errorf("%s %v: the %s box %+v and the %s box %+v are %d cells apart, want one cell between groups and no overlap", label, style, a.name, a.r, b.name, b.r, gap)
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
