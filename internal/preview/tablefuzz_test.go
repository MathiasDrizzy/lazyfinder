package preview

import "testing"

// Whatever a CSV or TSV holds, laying it out never panics and never passes the output budget.
func FuzzRenderTable(f *testing.F) {
	for _, seed := range []string{"a,b\n1,2\n", "\"x\ny\",z\n", ",,,,\na\n", "a\tb\n", "\"unterminated\n", "\x1b[2J,é,日本\n"} {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, body string, tsv bool) {
		sep := ','
		if tsv {
			sep = '\t'
		}
		lines, ok := renderTable(body, sep, true)
		if !ok {
			return
		}
		size := 0
		for _, l := range lines {
			size += len(l) + 1
		}
		if size > tableOutputBudget(len(body))+len("…")+1 {
			t.Fatalf("%d bytes laid out as %d", len(body), size)
		}
	})
}
