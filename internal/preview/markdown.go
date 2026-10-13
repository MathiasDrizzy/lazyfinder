package preview

import (
	"strings"

	"charm.land/glamour/v2"
)

// renderMarkdown turns Markdown into styled lines wrapped at width. Any failure returns ok=false and the caller
// shows the source instead.
func renderMarkdown(src string, width int, noColor bool) (lines []string, ok bool) {
	style := "dark"
	if noColor {
		style = "notty"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(max(20, width)))
	if err != nil {
		return nil, false
	}
	out, err := r.Render(src)
	if err != nil {
		return nil, false
	}
	return strings.Split(strings.Trim(out, "\n"), "\n"), true
}
