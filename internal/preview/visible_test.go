package preview

import "testing"

// The bidi controls of a file's text reorder what the terminal draws (Trojan Source): the preview shows them as �.
func TestThePreviewShowsBidiControlsAsPictures(t *testing.T) {
	if got := Visible("a\u202eb\u2066c\u200fd"); got != "a�b�c�d" {
		t.Fatalf("%q", got)
	}
}
