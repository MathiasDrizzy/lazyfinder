package ui

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/search"
	"github.com/charmbracelet/x/ansi"
)

type searchMode int

const (
	searchOff     searchMode = iota
	searchTyping             // the bar has the keyboard
	searchResults            // the bar stays, the list has the keyboard
)

// searchState is the search bar. While it is not off, m.listing holds the results and base the folder's own listing.
type searchState struct {
	mode      searchMode
	input     string
	recursive bool
	base      fsmodel.Listing
	results   []fsmodel.Entry
	running   bool
	truncated bool
	gen       int
	cancel    context.CancelFunc
	ch        chan searchEvent
}

// searchEvent is one step of a running walk.
type searchEvent struct {
	gen     int
	entries []fsmodel.Entry
	done    bool
	err     error
}

type searchMsg searchEvent

// A chip adds or removes a token of the query with one click.
type chip struct {
	label func() string
	token string // empty for the scope chip
}

var chips = []chip{
	{func() string { return i18n.T("Imágenes", "Images") }, "kind:image"},
	{func() string { return i18n.T("Texto", "Text") }, "kind:text"},
	{func() string { return i18n.T("Carpetas", "Folders") }, "kind:folder"},
	{func() string { return i18n.T("Hoy", "Today") }, "modified:today"},
	{func() string { return i18n.T("Semana", "Week") }, "modified:week"},
}

func (m *Model) searchOn() bool { return m.search.mode != searchOff }

func (m *Model) openSearch() {
	if m.search.mode == searchOff {
		m.search = searchState{base: m.listing}
	}
	m.search.mode = searchTyping
}

// endSearch closes the bar, stops any walk and brings the folder back.
func (m *Model) endSearch() tea.Cmd {
	if m.search.mode == searchOff {
		return nil
	}
	if m.search.cancel != nil {
		m.search.cancel()
	}
	m.search = searchState{}
	return m.reload()
}

func (m *Model) searchKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	switch key {
	case "esc":
		return m.endSearch()
	case "enter":
		m.search.mode = searchResults
		return nil
	case "ctrl+s":
		return m.saveSearch()
	case "tab":
		m.search.recursive = !m.search.recursive
		return m.runSearch()
	case "up", "down", "pgup", "pgdown":
		act, _ := m.keys.Lookup(key)
		return m.listKey(act)
	case "backspace":
		if r := []rune(m.search.input); len(r) > 0 {
			m.search.input = string(r[:len(r)-1])
		}
		return m.runSearch()
	case "ctrl+u":
		m.search.input = ""
		return m.runSearch()
	}
	if k.Text != "" && !strings.ContainsAny(key, "\x00") && (k.Mod&(tea.ModCtrl|tea.ModAlt)) == 0 {
		m.search.input += k.Text
		return m.runSearch()
	}
	return nil
}

// resultsKey handles the keys of the results mode that differ from the plain list: esc clears, / edits.
func (m *Model) resultsKey(key string) (tea.Cmd, bool) {
	switch key {
	case "esc":
		return m.endSearch(), true
	case "/", "ctrl+f":
		m.search.mode = searchTyping
		return nil, true
	case "ctrl+s":
		return m.saveSearch(), true
	}
	return nil, false
}

// runSearch applies the current query: at once inside the folder, in the background for subfolders.
func (m *Model) runSearch() tea.Cmd {
	if m.search.cancel != nil {
		m.search.cancel()
		m.search.cancel = nil
	}
	m.search.gen++
	m.search.running, m.search.truncated = false, false
	q, err := search.Parse(m.search.input, m.deps.Now())
	if err != nil {
		m.setError(err)
		return nil
	}
	m.setStatus("")
	if q.Empty() {
		return m.showResults(m.search.base.Entries)
	}
	if !m.search.recursive {
		var out []fsmodel.Entry
		for _, e := range m.search.base.Entries {
			if q.Match(e) {
				out = append(out, e)
			}
		}
		return m.showResults(out)
	}
	m.showResults(nil)
	return m.startWalk(q)
}

func (m *Model) showResults(es []fsmodel.Entry) tea.Cmd {
	m.search.results = es
	l := fsmodel.Listing{Dir: m.search.base.Dir, Entries: es}.Sorted(m.sort)
	m.listing = l
	m.sel, m.top = 0, 0
	return m.refreshPreview()
}

func (m *Model) startWalk(q search.Query) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan searchEvent, 4)
	m.search.cancel, m.search.ch, m.search.running = cancel, ch, true
	gen, root, max := m.search.gen, m.search.base.Dir, m.deps.Cfg.Search.MaxResults
	tags := maps.Clone(m.tags) // the walk runs in its own goroutine
	opts := search.Options{Recursive: true, ShowHidden: m.hidden, Max: max, Tags: func(p string) []string { return tags[p] }}
	go func() {
		err := search.Walk(ctx, root, q, opts, func(batch []fsmodel.Entry) {
			select {
			case ch <- searchEvent{gen: gen, entries: batch}:
			case <-ctx.Done():
			}
		})
		select {
		case ch <- searchEvent{gen: gen, done: true, err: err}:
		case <-ctx.Done():
		}
	}()
	return waitSearch(ch)
}

func waitSearch(ch chan searchEvent) tea.Cmd {
	return func() tea.Msg { return searchMsg(<-ch) }
}

func (m *Model) onSearch(msg searchMsg) tea.Cmd {
	if msg.gen != m.search.gen || m.search.mode == searchOff {
		return nil
	}
	if len(msg.entries) > 0 {
		keep := ""
		if e, ok := m.selected(); ok {
			keep = e.Name
		}
		m.search.results = append(m.search.results, msg.entries...)
		m.retag()
		m.listing = fsmodel.Listing{Dir: m.search.base.Dir, Entries: m.search.results}.Sorted(m.sort)
		for i, e := range m.listing.Entries {
			if e.Name == keep {
				m.sel = i
			}
		}
		m.ensureVisible()
	}
	if msg.done {
		m.search.running = false
		m.search.truncated = errors.Is(msg.err, search.ErrTruncated)
		if msg.err != nil && !m.search.truncated && !errors.Is(msg.err, context.Canceled) {
			m.setError(msg.err)
		}
		return m.refreshPreview()
	}
	return waitSearch(m.search.ch)
}

// where is the folder of a result relative to the folder being searched, empty for direct children.
func (m *Model) where(e fsmodel.Entry) string {
	rel, err := filepath.Rel(m.search.base.Dir, filepath.Dir(e.Path))
	if err != nil || rel == "." {
		return ""
	}
	return rel
}

// chipSpan is where a chip of the bar sits on the bottom row.
type chipSpan struct {
	x, w  int
	scope bool
	token string
}

func (m *Model) scopeLabel() string {
	if m.search.recursive {
		return i18n.T("Subcarpetas", "Subfolders")
	}
	return i18n.T("Esta carpeta", "This folder")
}

// searchBar renders the bottom row while searching and returns where its chips are.
func (m *Model) searchBar() (string, []chipSpan) {
	prompt := " / "
	text := m.search.input
	cursor := ""
	if m.search.mode == searchTyping {
		cursor = "▏"
	}
	if text == "" && m.search.mode == searchTyping {
		text = m.styles.dim(i18n.T("escribe para buscar (kind: size: modified:)", "type to search (kind: size: modified:)"))
	}
	left := m.styles.accent(prompt) + text + cursor
	var spans []chipSpan
	x := ansi.StringWidth(left) + 2
	var parts []string
	add := func(label string, active bool, scope bool, token string) {
		s := " " + label + " "
		w := ansi.StringWidth(s)
		switch {
		case active:
			s = m.styles.selected(s)
		default:
			s = m.styles.footerKey(s)
		}
		parts = append(parts, s)
		spans = append(spans, chipSpan{x: x, w: w, scope: scope, token: token})
		x += w + 1
	}
	add("["+m.scopeLabel()+"]", false, true, "")
	for _, c := range chips {
		add(c.label(), strings.Contains(" "+m.search.input+" ", " "+c.token+" "), false, c.token)
	}
	count := fmt.Sprintf(i18n.T("%d encontrados", "%d found"), len(m.entries()))
	switch {
	case m.search.running:
		count = i18n.T("buscando…", "searching…")
	case m.search.truncated:
		count += " (" + i18n.T("límite alcanzado", "limit reached") + ")"
	}
	line := left + "  " + strings.Join(parts, " ")
	pad := m.w - ansi.StringWidth(line) - ansi.StringWidth(count) - 1
	if pad > 0 {
		line += strings.Repeat(" ", pad) + m.styles.dim(count)
	}
	return fit(line, m.w), spans
}

// toggleToken adds the token to the query, or removes it when it is already there.
func (m *Model) toggleToken(token string) tea.Cmd {
	words := strings.Fields(m.search.input)
	out := words[:0:0]
	found := false
	for _, w := range words {
		if w == token {
			found = true
			continue
		}
		out = append(out, w)
	}
	if !found {
		out = append(out, token)
	}
	m.search.input = strings.Join(out, " ")
	return m.runSearch()
}

func (m *Model) clickChip(idx int) tea.Cmd {
	_, spans := m.searchBar()
	if idx < 0 || idx >= len(spans) {
		return nil
	}
	if spans[idx].scope {
		m.search.recursive = !m.search.recursive
		return m.runSearch()
	}
	return m.toggleToken(spans[idx].token)
}
