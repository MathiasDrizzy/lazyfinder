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

// One row with a huge number of columns padded every other row to that width: a hostile CSV of a few hundred KB grew
// into gigabytes of preview. The table keeps the first maxTableColumns columns and says how many it left out.
func TestATableWithTooManyColumnsStaysSmall(t *testing.T) {
	body := strings.Repeat(",", 200_000) + "\n" + strings.Repeat("a\n", 5_000)
	lines, ok := renderTable(body, ',', true)
	if !ok {
		t.Fatal("the body is valid CSV")
	}
	size := 0
	for _, l := range lines {
		size += len(l)
	}
	if size > 4<<20 {
		t.Fatalf("the preview of %d bytes grew to %d bytes", len(body), size)
	}
	if !strings.Contains(lines[0], "…") {
		t.Fatalf("the header must say columns were left out: %q", lines[0][:min(len(lines[0]), 80)])
	}
}

// A wide header over many short rows padded each row to the header: the output is capped near the size of the input,
// and the table then ends with a "…" row.
func TestATableOfManyShortRowsStaysSmall(t *testing.T) {
	header := strings.TrimSuffix(strings.Repeat(strings.Repeat("h", 40)+",", 64), ",")
	body := header + "\n" + strings.Repeat("a\n", 100_000)
	lines, ok := renderTable(body, ',', true)
	if !ok {
		t.Fatal("the body is valid CSV")
	}
	size := 0
	for _, l := range lines {
		size += len(l)
	}
	if size > tableOutputBudget(len(body)) || lines[len(lines)-1] != "…" {
		t.Fatalf("%d bytes became %d in %d lines, last %q", len(body), size, len(lines), lines[len(lines)-1])
	}
}
