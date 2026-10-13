package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
)

// minReadable is the contrast every text keeps against its background (WCAG asks 3:1 for bold text and interface).
const minReadable = 3.1

// look says how to draw. It comes from the config and the environment; the model rebuilds the styles when it changes.
type look struct {
	Theme            string // a palette name or theme.Terminal
	NoColor          bool   // NO_COLOR: no colors at all, only bold, faint and reverse
	ScreenBackground string // config.ScreenBackgroundTheme or ...Terminal
}

// styles are the render functions the views use. A style never relies on color alone: the selection is a bar plus
// a pointer, folders carry a ">" marker, and errors start with "Error:" (NO_COLOR and monochrome terminals keep every
// distinction).
type styles struct {
	selected, selectedInactive func(string) string
	// stripe paints the soft alternate background of every second row (identity without a palette); stripeBg is
	// its color.
	stripe                    func(string) string
	stripeBg                  color.Color
	dim, bold, folder, accent func(string) string
	fresh                     func(string) string
	errText, border           func(string) string
	borderActive, titleActive func(string) string
	title, footerKey, footer  func(string) string
	icon                      func(role icons.Role, glyph string, dim bool) string
	popupBorder, popupTitle   func(string) string
	// gitLetter colors the git state next to a name
	gitLetter func(code string) string
	// bg is painted behind the whole screen (nil: the terminal's own background)
	bg color.Color
	// text is the color of text that has none of its own, painted with bg so a light theme is readable on a dark terminal
	text color.Color
}

func newStyles(l look) styles {
	r := func(s lipgloss.Style) func(string) string { return func(t string) string { return s.Render(t) } }
	plain := func(s string) string { return s }
	noIconColor := func(_ icons.Role, g string, dim bool) string {
		if dim {
			return lipgloss.NewStyle().Faint(true).Render(g)
		}
		return g
	}
	mono := l.NoColor
	if mono {
		return styles{
			stripe:   plain,
			selected: r(lipgloss.NewStyle().Reverse(true)), selectedInactive: r(lipgloss.NewStyle().Bold(true)),
			dim: r(lipgloss.NewStyle().Faint(true)), bold: r(lipgloss.NewStyle().Bold(true)), folder: r(lipgloss.NewStyle().Bold(true)),
			accent: plain, fresh: r(lipgloss.NewStyle().Bold(true)), errText: r(lipgloss.NewStyle().Bold(true)), border: r(lipgloss.NewStyle().Faint(true)),
			borderActive: plain, titleActive: r(lipgloss.NewStyle().Bold(true)), title: r(lipgloss.NewStyle().Faint(true)),
			footerKey: r(lipgloss.NewStyle().Bold(true)), footer: r(lipgloss.NewStyle().Faint(true)),
			popupBorder: plain, popupTitle: r(lipgloss.NewStyle().Bold(true)), icon: noIconColor,
			gitLetter: plain,
		}
	}
	p, ok := theme.Lookup(l.Theme)
	if !ok { // "terminal": the terminal's own 16 colors
		blue, red := lipgloss.Color("12"), lipgloss.Color("1")
		return styles{
			stripe:   plain,
			selected: r(lipgloss.NewStyle().Reverse(true)), selectedInactive: r(lipgloss.NewStyle().Bold(true)),
			dim: r(lipgloss.NewStyle().Faint(true)), bold: r(lipgloss.NewStyle().Bold(true)),
			folder: r(lipgloss.NewStyle().Foreground(blue).Bold(true)), accent: r(lipgloss.NewStyle().Foreground(blue)), fresh: r(lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)),
			errText: r(lipgloss.NewStyle().Foreground(red).Bold(true)), border: r(lipgloss.NewStyle().Faint(true)),
			borderActive: r(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), titleActive: r(lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)),
			title: r(lipgloss.NewStyle().Faint(true)), footerKey: r(lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)),
			footer: r(lipgloss.NewStyle().Faint(true)), popupBorder: r(lipgloss.NewStyle().Foreground(blue)),
			popupTitle: r(lipgloss.NewStyle().Foreground(blue).Bold(true)), icon: noIconColor,
			gitLetter: plain,
		}
	}
	// Every text color is lifted when the palette leaves it under minReadable on the background it is drawn on.
	fg := func(c color.Color, bold bool) func(string) string {
		return r(lipgloss.NewStyle().Foreground(theme.Readable(c, p.Base, minReadable)).Bold(bold))
	}
	stripeBg := theme.Mix(p.Base, p.Surface0, 0.5)
	stripeSeq := theme.BackgroundSeq(stripeBg)
	s := styles{
		stripeBg: stripeBg,
		// A row is striped by setting the background after the start and after every reset inside it, so the colored
		// icons and git letters do not cut the stripe.
		stripe: func(t string) string {
			t = strings.ReplaceAll(t, "\x1b[m", "\x1b[m"+stripeSeq)
			t = strings.ReplaceAll(t, "\x1b[0m", "\x1b[0m"+stripeSeq)
			return stripeSeq + t + "\x1b[m"
		},
		// The selection bar is the theme's surface1, a blue-gray, as in lazymark; the inactive one is a step darker.
		selected:         r(lipgloss.NewStyle().Background(p.Surface1).Foreground(theme.Readable(p.Text, p.Surface1, minReadable)).Bold(true)),
		selectedInactive: r(lipgloss.NewStyle().Background(p.Surface0).Foreground(theme.Readable(p.Text, p.Surface0, minReadable))),
		dim:              fg(p.Overlay0, false), bold: fg(p.Text, true), folder: fg(p.Blue, true), accent: fg(p.Peach, false), fresh: fg(p.Green, true),
		errText: fg(p.Red, true), border: fg(p.Overlay0, false),
		borderActive: fg(p.Green, false), titleActive: fg(p.Green, true), title: fg(p.Subtext0, false),
		footerKey: fg(p.Mauve, true), footer: fg(p.Blue, false),
		popupBorder: fg(p.Peach, false), popupTitle: fg(p.Peach, true),
	}
	roles := map[icons.Role]color.Color{icons.Folder: p.Blue, icons.Green: p.Green, icons.Peach: p.Peach, icons.Red: p.Red,
		icons.Yellow: p.Yellow, icons.Mauve: p.Mauve, icons.Teal: p.Teal, icons.Blue: p.Blue, icons.Plain: p.Subtext0}
	s.icon = func(role icons.Role, glyph string, dim bool) string {
		c := p.Overlay0
		if !dim {
			c = roles[role]
		}
		return lipgloss.NewStyle().Foreground(theme.Readable(c, p.Base, minReadable)).Render(glyph)
	}
	gitColors := map[string]color.Color{"M": p.Yellow, "U": p.Green, "A": p.Green, "D": p.Red, "R": p.Blue, "!": p.Red, "I": p.Overlay0}
	s.gitLetter = func(code string) string {
		c, ok := gitColors[code]
		if !ok {
			return code
		}
		return lipgloss.NewStyle().Foreground(theme.Readable(c, p.Base, minReadable)).Bold(code == "!").Render(code)
	}
	if l.ScreenBackground == "theme" {
		s.bg, s.text = p.Base, p.Text
	}
	return s
}
