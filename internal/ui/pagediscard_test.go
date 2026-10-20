package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/preview"
)

// onPreview removed the page image of a PDF only on its way to the terminal: a result that came too late, or one
// with images off, left its lazyfinder-page-* folder in $TMPDIR, one per PDF looked at. Every page image goes.
func TestAPDFPageThatIsNotShownIsRemoved(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	page := func() string {
		t.Helper()
		dir, err := os.MkdirTemp("", "lazyfinder-page-")
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "page.png")
		if err := os.WriteFile(p, []byte("png"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	e := newEnv(t)
	e.m.pv.open, e.m.pv.id = true, 7
	stale := page()
	e.m.onPreview(previewMsg{id: 6, res: preview.Result{Kind: preview.KindImage, ImagePath: stale}})
	e.m.deps.Cfg.Preview.Images = "off"
	off := page()
	e.m.onPreview(previewMsg{id: 7, res: preview.Result{Kind: preview.KindImage, ImagePath: off}})
	for _, p := range []string{stale, off} {
		if _, err := os.Stat(filepath.Dir(p)); !os.IsNotExist(err) {
			t.Errorf("%s stayed: %v", filepath.Dir(p), err)
		}
	}
}
