package ui

import (
	"strings"
	"testing"
	"time"
)

// Mathias, 2026-10-13: "porque en la configuracion dice kitty? ... eso deberia ser automatico como lazymark" and
// "porque dice raton? el raton va con la opcion de atajos ... como esta en lazymark". lazymark has neither row: the
// terminal is asked for Kitty support by itself, and the mouse is `mouse_click` in the file and `--no-mouse`.
func TestSettingsListHasNoTerminalProbeRows(t *testing.T) {
	e := newEnv(t)
	e.key(",")
	e.mustHave("Theme")
	e.mustNotHave("Kitty")
	e.mustNotHave("Mouse")
	e.mustNotHave("Images")
}

func TestImagesStayAutomaticAndMouseKeepsItsConfig(t *testing.T) {
	e := newEnv(t)
	if e.m.deps.Cfg.Preview.Images != "auto" {
		t.Fatalf("images default to auto, got %q", e.m.deps.Cfg.Preview.Images)
	}
	if !e.m.mouse {
		t.Fatal("the mouse is on by default")
	}
}

// Mathias, 2026-10-13: "hay uno que dice ayer, en vez de la fecha, hay uno que dice 1979". One format for the whole
// column, and a folder without a real date shows "--".
func TestDateColumnUsesOneFormatAndHidesEpochDates(t *testing.T) {
	e := newEnv(t)
	now := e.m.deps.Now()
	e.m.deps.Cfg.DateFormat = "absolute"
	for _, tm := range []time.Time{now, now.AddDate(0, 0, -1), now.AddDate(-1, 0, 0), time.Unix(0, 0)} {
		got := e.m.date(tm)
		for _, bad := range []string{"Today", "Yesterday", "Hoy", "Ayer", "1969", "1970", "1979"} {
			if strings.Contains(got, bad) {
				t.Errorf("%v shown as %q", tm, got)
			}
		}
	}
	if got := e.m.date(time.Unix(0, 0)); got != "--" {
		t.Fatalf("the epoch is a missing date: %q", got)
	}
	if got := e.m.date(now.AddDate(0, 0, -1)); !strings.Contains(got, ":") {
		t.Fatalf("an absolute date carries its time: %q", got)
	}
}
