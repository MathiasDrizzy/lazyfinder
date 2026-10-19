package preview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// A CSV or TSV shows as a table: its columns line up, a rule follows the header, and a quoted comma stays in its cell.
func TestACSVShowsAsATable(t *testing.T) {
	for name, body := range map[string]string{
		"t.csv": "name,size,note\nalpha,1,\"a, b\"\nlonger name,22,x\n",
		"t.tsv": "name\tsize\tnote\nalpha\t1\ta, b\nlonger name\t22\tx\n",
	} {
		r := loadWith(t, write(t, name, body), Options{NoColor: true})
		lines := r.StripANSI()
		if r.Kind != KindText || len(lines) != 4 {
			t.Fatalf("%s: kind %v, lines %q", name, r.Kind, lines)
		}
		col := strings.Index(lines[0], "size")
		for _, l := range []string{lines[2], lines[3]} {
			if i := strings.Index(l, "│"); ansi.StringWidth(l[:i]) != ansi.StringWidth(lines[0][:strings.Index(lines[0], "│")]) {
				t.Errorf("%s: columns do not line up:\n%s", name, strings.Join(lines, "\n"))
			}
		}
		if col < 0 || !strings.Contains(lines[1], "─") || !strings.Contains(lines[2], "a, b") {
			t.Errorf("%s: not a table:\n%s", name, strings.Join(lines, "\n"))
		}
	}
}

// A wide cell is cut at the cell limit, and a file that is not valid CSV shows as text.
func TestATableCutsWideCellsAndFallsBackToText(t *testing.T) {
	r := loadWith(t, write(t, "w.csv", "a,b\n"+strings.Repeat("x", 200)+",y\n"), Options{NoColor: true})
	for _, l := range r.StripANSI() {
		if ansi.StringWidth(l) > 2*maxCellWidth+10 {
			t.Fatalf("a cell was not cut: %d cells", ansi.StringWidth(l))
		}
	}
	r = loadWith(t, write(t, "bad.csv", "a,\"b\nc"), Options{NoColor: true})
	if got := strings.Join(r.StripANSI(), "\n"); !strings.Contains(got, "a,\"b") {
		t.Fatalf("invalid CSV must show as text:\n%s", got)
	}
}

// A CSV longer than the preview's budget still shows as a table of its whole rows, marked as cut.
func TestATruncatedCSVShowsItsWholeRows(t *testing.T) {
	body := "a,b\n" + strings.Repeat("1,2\n", 100)
	r := loadWith(t, write(t, "long.csv", body), Options{NoColor: true, MaxTextBytes: 50})
	lines := r.StripANSI()
	if !r.Truncated || len(lines) < 3 || !strings.Contains(lines[1], "┼") || lines[len(lines)-1] != "1 │ 2" {
		t.Fatalf("truncated=%v lines=%q", r.Truncated, lines)
	}
}
