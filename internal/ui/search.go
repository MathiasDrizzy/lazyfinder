package ui

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
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
	scope     string // folder, subfolders or system
	kinds     string // both, files or folders
	by        string // name or content
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
	note    string // a message for the status bar
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

// The search choices, as written in config.json (search.scope, search.kinds, search.by).
var (
	searchScopes = []string{"folder", "subfolders", "system"}
	searchKinds  = []string{"both", "files", "folders"}
	searchBys    = []string{"name", "content"}
)

func orDefault(v string, list []string) string {
	if slices.Contains(list, v) {
		return v
	}
	return list[0]
}

// deep reports whether the search looks beyond the folder itself.
func (s searchState) deep() bool { return s.scope != "folder" }

func (m *Model) openSearch() {
	if m.search.mode == searchOff {
		c := m.deps.Cfg.Search
		m.search = searchState{base: m.listing, scope: orDefault(c.Scope, searchScopes), kinds: orDefault(c.Kinds, searchKinds), by: orDefault(c.By, searchBys)}
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
		return m.cycleSearch("scope", 1)
	case "ctrl+k":
		return m.cycleSearch("kinds", 1)
	case "ctrl+n":
		return m.cycleSearch("by", 1)
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

// cycleSearch moves one of the three search choices to its next value, remembers it in config.json and searches again.
func (m *Model) cycleSearch(what string, dir int) tea.Cmd {
	c := &m.deps.Cfg.Search
	switch what {
	case "scope":
		m.search.scope = cycle(searchScopes, m.search.scope, dir)
		c.Scope = m.search.scope
	case "kinds":
		m.search.kinds = cycle(searchKinds, m.search.kinds, dir)
		c.Kinds = m.search.kinds
	default:
		m.search.by = cycle(searchBys, m.search.by, dir)
		c.By = m.search.by
	}
	m.rememberSearch(what)
	return m.runSearch()
}

func (m *Model) rememberSearch(what string) {
	if m.deps.ConfigPath == "" {
		return
	}
	c := m.deps.Cfg.Search
	val := map[string]string{"scope": c.Scope, "kinds": c.Kinds, "by": c.By}[what]
	if err := config.Update(m.deps.ConfigPath, map[string]any{"search." + what: val}); err != nil {
		m.setError(err)
	}
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
	q, err := search.ParseBy(m.search.input, m.deps.Now(), m.search.by)
	if err != nil {
		m.setError(err)
		return nil
	}
	m.setStatus("")
	if q.Empty() && m.search.kinds == "both" {
		return m.showResults(m.search.base.Entries)
	}
	q = q.Only(m.search.kinds)
	if !m.search.deep() {
		var out []fsmodel.Entry
		want := []byte(strings.ToLower(q.Content()))
		for _, e := range m.search.base.Entries {
			if q.Match(e) && (q.Content() == "" || (!e.IsDir && search.FileContains(e.Path, want))) {
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
	system, index, home := m.search.scope == "system", m.deps.Index, m.deps.Home
	tags := maps.Clone(m.tags) // the walk runs in its own goroutine
	opts := search.Options{Recursive: true, ShowHidden: m.hidden, Max: max, Tags: func(p string) []string { return tags[p] }}
	find, hidden, text := m.deps.ContentFind, m.hidden, q.Content()
	go func() {
		if system {
			m.indexWalk(ctx, ch, gen, q, opts, index, home)
			return
		}
		if text != "" {
			set, err := search.ContentPaths(ctx, root, text, search.ContentOptions{Hidden: hidden, Find: find})
			if err != nil {
				select {
				case ch <- searchEvent{gen: gen, done: true, err: err}:
				case <-ctx.Done():
				}
				return
			}
			opts.Allowed = func(p string) bool { return set[p] }
		}
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

// indexWalk is the whole-system search: it asks the system index and, when there is none, walks the home folder
// and says so. It runs in its own goroutine and only talks through ch.
func (m *Model) indexWalk(ctx context.Context, ch chan searchEvent, gen int, q search.Query, opts search.Options, index search.Indexer, home string) {
	send := func(ev searchEvent) {
		ev.gen = gen
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	}
	err := search.Index(ctx, index, q, opts, func(b []fsmodel.Entry) { send(searchEvent{entries: b}) })
	if errors.Is(err, search.ErrNoIndex) {
		send(searchEvent{note: i18n.T("El índice del sistema no está disponible: se busca en tu carpeta personal", "The system index is not available: searching your home folder")})
		if q.Content() != "" {
			send(searchEvent{done: true, err: err})
			return
		}
		err = search.Walk(ctx, home, q, opts, func(b []fsmodel.Entry) { send(searchEvent{entries: b}) })
	}
	send(searchEvent{done: true, err: err})
}

func waitSearch(ch chan searchEvent) tea.Cmd {
	return func() tea.Msg { return searchMsg(<-ch) }
}

func (m *Model) onSearch(msg searchMsg) tea.Cmd {
	if msg.gen != m.search.gen || m.search.mode == searchOff {
		return nil
	}
	if msg.note != "" {
		m.setStatus(msg.note)
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
	if m.search.scope == "system" {
		return m.homeShort(filepath.Dir(e.Path))
	}
	rel, err := filepath.Rel(m.search.base.Dir, filepath.Dir(e.Path))
	if err != nil || rel == "." {
		return ""
	}
	return rel
}

// chipSpan is where a chip of the bar sits on the bottom row.
type chipSpan struct {
	x, w   int
	action string // scope, kinds or by for the three choices; empty for a filter chip
	token  string
}

func (m *Model) scopeLabel() string { return scopeName(m.search.scope) }

func scopeName(scope string) string {
	switch scope {
	case "subfolders":
		return i18n.T("Subcarpetas", "Subfolders")
	case "system":
		return i18n.T("Sistema", "System")
	}
	return i18n.T("Esta carpeta", "This folder")
}

func kindsName(kinds string) string {
	switch kinds {
	case "files":
		return i18n.T("Archivos", "Files")
	case "folders":
		return i18n.T("Carpetas", "Folders")
	}
	return i18n.T("Archivos y carpetas", "Files and folders")
}

func byName(by string) string {
	if by == "content" {
		return i18n.T("Contenido", "Content")
	}
	return i18n.T("Nombre", "Name")
}

// searchBar renders the prompt row while searching: the field and how many things were found.
func (m *Model) searchBar() string {
	prompt := " / "
	text := m.search.input
	cursor := ""
	if m.search.mode == searchTyping {
		cursor = "▏"
	}
	if text == "" && m.search.mode == searchTyping {
		text = m.styles.dim(i18n.T("escribe para buscar · Enter va a los resultados · Esc cierra", "type to search · Enter goes to the results · Esc closes"))
	}
	line := m.styles.accent(prompt) + text + cursor
	count := fmt.Sprintf(i18n.T("%d encontrados", "%d found"), len(m.entries()))
	switch {
	case m.search.running:
		count = i18n.T("buscando…", "searching…")
	case m.search.truncated:
		count += " (" + i18n.T("límite alcanzado", "limit reached") + ")"
	}
	pad := m.w - ansi.StringWidth(line) - ansi.StringWidth(count) - 1
	if pad > 0 {
		line += strings.Repeat(" ", pad) + m.styles.dim(count)
	}
	return fit(line, m.w)
}

// filterBar renders the row above the prompt: where, what and how to search as labelled buttons, then the quick
// filters. What does not fit is dropped whole (first the quick filters, then the key hints), never cut.
func (m *Model) filterBar() (string, []chipSpan) {
	for _, v := range []struct {
		hints, labels bool
		chips         int
	}{{true, true, len(chips)}, {true, true, 3}, {true, true, 0}, {false, true, 0}, {false, false, 0}} {
		if line, spans := m.buildFilterBar(v.hints, v.labels, v.chips); ansi.StringWidth(line) <= m.w {
			return line, spans
		}
	}
	line, spans := m.buildFilterBar(false, false, 0)
	return fit(line, m.w), spans
}

func (m *Model) buildFilterBar(hints, labels bool, nChips int) (string, []chipSpan) {
	var b strings.Builder
	var spans []chipSpan
	x := 0
	put := func(styled, plain string) {
		b.WriteString(styled)
		x += ansi.StringWidth(plain)
	}
	put(" ", " ")
	choice := func(label, hint, value, action string) {
		if hints {
			label += " (" + hint + ")"
		}
		if labels {
			put(m.styles.dim(label+": "), label+": ")
		}
		button := " " + value + " ▾ "
		spans = append(spans, chipSpan{x: x, w: ansi.StringWidth(button), action: action})
		put(m.styles.selected(button), button)
		put("  ", "  ")
	}
	choice(i18n.T("Dónde", "Look in"), "Tab", m.scopeLabel(), "scope")
	choice(i18n.T("Qué", "What"), "^K", kindsName(m.search.kinds), "kinds")
	choice(i18n.T("Por", "By"), "^N", byName(m.search.by), "by")
	if nChips > 0 {
		put(m.styles.dim("│ "), "│ ")
	}
	for _, c := range chips[:nChips] {
		label := " " + c.label() + " "
		active := strings.Contains(" "+m.search.input+" ", " "+c.token+" ")
		spans = append(spans, chipSpan{x: x, w: ansi.StringWidth(label), token: c.token})
		if active {
			put(m.styles.selected(label), label)
		} else {
			put(m.styles.footerKey(label), label)
		}
		put(" ", " ")
	}
	return b.String(), spans
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
	_, spans := m.filterBar()
	if idx < 0 || idx >= len(spans) {
		return nil
	}
	if a := spans[idx].action; a != "" {
		return m.cycleSearch(a, 1)
	}
	return m.toggleToken(spans[idx].token)
}
