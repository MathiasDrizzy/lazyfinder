package ui

import (
	"path/filepath"
	"testing"
)

// Leaving the sidebar mode folds every expanded folder, so their watches must go too: they stayed until some
// later reload, each one costing a watch the user can no longer see.
func TestLeavingTheSidebarModeStopsWatchingTheFoldedFolders(t *testing.T) {
	e, fw := liveEnv(t, "")
	e.m.toggleProject()
	e.clickText("alpha")
	e.key("+")
	alpha := filepath.Join(e.root, "alpha")
	if !fw.has(alpha) {
		t.Fatalf("watched = %v, want alpha once expanded", fw.watched)
	}
	e.m.toggleProject()
	if fw.has(alpha) {
		t.Fatalf("watched = %v after leaving the sidebar mode", fw.watched)
	}
}
