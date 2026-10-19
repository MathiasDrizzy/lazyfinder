package search

import (
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

// Whatever is typed in the search bar is parsed and then matched against entries: neither may panic.
func FuzzParseAndMatch(f *testing.F) {
	for _, s := range []string{"", "notes", "kind:image", "size:>10mb", "modified:<7d", "ext:go in:main", `"two words"`, "kind:", "size:>", "-x", "in:\x00", "created:2026-01-01"} {
		f.Add(s, "name")
		f.Add(s, "content")
	}
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	entries := []fsmodel.Entry{
		{Name: "a.go", Path: "/x/a.go", Size: 10, Modified: now, Created: now},
		{Name: "photo.PNG", Path: "/x/photo.PNG", Size: 1 << 30, Modified: now.Add(-48 * time.Hour)},
		{Name: "dir", Path: "/x/dir", IsDir: true},
		{Name: "", Path: ""},
	}
	f.Fuzz(func(t *testing.T, s, by string) {
		q, err := ParseBy(s, now, by)
		if err != nil {
			return
		}
		for _, e := range entries {
			q.Match(e)
		}
		q.Only("files").Match(entries[0])
	})
}
