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
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
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
			// Mathias, 2026-10-14: "el de configuración y el de cambiar modo no están muy separados?": the mode
			// button and the gear follow each other like the actions; one free cell parts the actions, Search and
			// the mode button, and one free cell is left after the gear for the right edge of its box.
			sort.Slice(boxes, func(i, j int) bool { return boxes[i].r.x < boxes[j].r.x })
			for i := 1; i < len(boxes); i++ {
				a, b := boxes[i-1], boxes[i]
				if a.r.y != b.r.y {
					continue
				}
				want := 1
				if a.name == b.name || a.name == "mode" && b.name == "gear" {
					want = 0
				}
				if gap := b.r.x - (a.r.x + a.r.w); gap != want {
					t.Errorf("%s %v: the %s box %+v and the %s box %+v are %d cells apart, want %d", label, style, a.name, a.r, b.name, b.r, gap, want)
				}
			}
			if last := boxes[len(boxes)-1]; last.r.x+last.r.w != e.m.w-1 {
				t.Errorf("%s %v: the last box %+v must leave one cell at the right edge (width %d)", label, style, last.r, e.m.w)
			}
		}
	}
}

// Mathias, 2026-10-14, zooming on the hover of the mode button: "no se puede centrar el icono con el recuadro gris …
// el icono está más a la derecha y la izquierda está más grande". A measurement of b46b985 in Ghostty (order 07,
// revision 3, S7): margins of 9 and 20 pixels. And then: "revisa también los tamaños de los recuadros gris … este es
// mucho más grande que los otros" (the collapse box, 57 pixels against 43). The box is read back from the screen,
// turned into pixels with the ink measured in Ghostty, and its two margins must differ by 2 pixels at most and every
// box must be as wide as the others within 3 pixels, for every Nerd Font button in both modes.
func TestTheLitBoxIsCenteredOnTheInkOfANerdGlyphAndAllBoxesAreAlike(t *testing.T) {
	eighths := map[rune]int{'▏': 1, '▎': 2, '▍': 3, '▌': 4, '▋': 5, '▊': 6, '▉': 7}
	rightEighthsOf := map[rune]int{'▕': 1, '🮇': 2, '🮈': 3, '▐': 4, '🮉': 5, '🮊': 6, '🮋': 7}
	cell := float64(inkCellPx)
	var widths []float64
	for _, sidebar := range []bool{false, true} {
		var e *env
		if sidebar {
			e = projectEnv(t, 60)
		} else {
			e = newEnv(t)
		}
		e.m.icons = icons.Set{Style: icons.Nerd}
		e.move(10, 8)
		var rects []hoverRect
		for _, a := range e.m.actionSpans() {
			rects = append(rects, hoverRect{a.x, a.y, a.w, 1})
		}
		for _, z := range []zoneKind{zoneMode, zoneSettings} {
			r, _ := e.m.zoneRect(zone{kind: z})
			rects = append(rects, r)
		}
		p, _ := theme.Lookup(e.m.deps.Cfg.Theme)
		bg := theme.BackgroundSeq(chipHoverBg(p))
		lit := func(raw string, x int) bool { return strings.Contains(ansi.Cut(raw, x, x+1), bg) }
		for _, r := range rects {
			e.move(r.x+1, r.y)
			raw := strings.Split(e.m.View().Content, "\n")[r.y]
			cells := []rune(ansi.Strip(raw))
			glyph := string(cells[r.x+1])
			ink, ok := nerdInkPx[glyph]
			if !ok {
				t.Errorf("sidebar=%v: the glyph %q of the button at %+v has no measured ink width", sidebar, glyph, r)
				continue
			}
			if !lit(raw, r.x+1) || !lit(raw, r.x+2) || cells[r.x+2] != ' ' {
				t.Errorf("sidebar=%v: the glyph %q and the space after it must be lit: %q", sidebar, glyph, raw)
				continue
			}
			// The part of the cell before the glyph that is lit: a space on the box color, or a block on the right of
			// the cell in the box color. No reversed colors: with the terminal's own background (Mathias's
			// "screen_background": "terminal") the unlit part showed as a white strip.
			var fill float64
			switch c := cells[r.x]; {
			case lit(raw, r.x) && c == ' ':
				fill = cell
			case rightEighthsOf[c] > 0:
				fill = float64(rightEighthsOf[c]) * cell / 8
			}
			tail := 0.0
			if r.x+3 < len(cells) && !lit(raw, r.x+3) {
				tail = float64(eighths[cells[r.x+3]]) * cell / 8
			}
			left := inkLeftPx + fill
			right := 2*cell - inkLeftPx - float64(ink) + tail
			if d := left - right; d > 2 || d < -2 {
				t.Errorf("sidebar=%v: the box of %q leaves %.1f px left and %.1f px right of its ink, want at most 2 px apart", sidebar, glyph, left, right)
			}
			widths = append(widths, fill+2*cell+tail)
		}
	}
	lo, hi := widths[0], widths[0]
	for _, w := range widths {
		lo, hi = min(lo, w), max(hi, w)
	}
	if hi-lo > 3 {
		t.Errorf("the boxes are from %.1f to %.1f px wide, want them alike within 3 px: %v", lo, hi, widths)
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

// The sidebar lights the mode button and the gear as a tall chip, three rows like herdr's tabs. When the middle row
// starts the box at the half block ▐, the rows above and below start at the same half (▗ and ▝): with whole cells
// there the box showed a notch on its left side in Ghostty (order 07, revision 3).
func TestTheTallChipOfTheSidebarHasStraightSides(t *testing.T) {
	e := projectEnv(t, 60)
	e.m.icons = icons.Set{Style: icons.Nerd}
	e.move(10, 8)
	for _, z := range []zoneKind{zoneMode, zoneSettings} {
		r, _ := e.m.zoneRect(zone{kind: z})
		e.move(r.x+1, r.y)
		s := e.screen()
		mid := []rune(s[1])
		if mid[r.x] != '▐' {
			t.Fatalf("zone %d: the middle row starts at the half block: %q", z, s[1])
		}
		if top, bottom := []rune(s[0])[r.x], []rune(s[2])[r.x]; top != '▗' || bottom != '▝' {
			t.Errorf("zone %d: the rows above and below start at the same half, got %q and %q:\n%s\n%s\n%s", z, top, bottom, s[0], s[1], s[2])
		}
	}
}
