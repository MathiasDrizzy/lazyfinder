package preview

import (
	"strings"
	"testing"
	"time"
)

// Some chroma lexers are quadratic on hostile input: 512 KB of [ in a .cpp file kept a core busy for half an hour
// while the cursor rested on it. Highlighting has a time budget, after which the text shows plain.
func TestHighlightingAHostileFileEndsSoon(t *testing.T) {
	body := []byte(strings.Repeat("[", 64<<10))
	start := time.Now()
	lines := highlight("x.cpp", body, false)
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("highlighting took %v", took)
	}
	if got := strings.Join(lines, ""); !strings.Contains(got, strings.Repeat("[", 100)) {
		t.Fatal("the text is lost")
	}
}
