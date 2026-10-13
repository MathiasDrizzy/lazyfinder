package ui

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const tagsFile = "tags.json"

// Tags are lazyfinder's own labels on paths, kept in tags.json next to config.json. They are not Finder's tags
// (those live in extended attributes); see ADR 0012.

// retag copies the stored tags onto the entries on screen.
func (m *Model) retag() {
	set := func(es []fsmodel.Entry) {
		for i := range es {
			es[i].Tags = m.tags[es[i].Path]
		}
	}
	set(m.listing.Entries)
	set(m.search.results)
}

func (m *Model) saveTags() { saveState(m.stateFile(tagsFile), m.tags) }

func (m *Model) startTag() tea.Cmd {
	if len(m.targets()) == 0 {
		return nil
	}
	m.prompt = promptState{kind: promptTag}
	return nil
}

// toggleTag adds the tag to every target, or removes it from all of them when they all have it already.
func (m *Model) toggleTag(name string) tea.Cmd {
	name = strings.TrimSpace(name)
	paths := m.targets()
	if name == "" || len(paths) == 0 {
		return nil
	}
	has := func(p string) bool { return slices.Contains(m.tags[p], name) }
	all := true
	for _, p := range paths {
		all = all && has(p)
	}
	for _, p := range paths {
		switch {
		case all:
			if rest := slices.DeleteFunc(slices.Clone(m.tags[p]), func(t string) bool { return t == name }); len(rest) > 0 {
				m.tags[p] = rest
			} else {
				delete(m.tags, p)
			}
		case !has(p):
			m.tags[p] = append(slices.Clone(m.tags[p]), name)
		}
	}
	m.saveTags()
	m.retag()
	if all {
		m.setStatus(fmt.Sprintf(i18n.T("Etiqueta quitada: %s", "Untagged: %s"), name))
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("Etiquetado: %s", "Tagged: %s"), name))
	}
	return nil
}

// moveTags makes the tags of from belong to to, after a rename.
func (m *Model) moveTags(from, to string) {
	ts, ok := m.tags[from]
	if !ok {
		return
	}
	delete(m.tags, from)
	m.tags[to] = ts
	m.saveTags()
	m.retag()
}

// showTags lists every tag in use with how many paths carry it; choosing one searches for it here.
func (m *Model) showTags() tea.Cmd {
	count := map[string]int{}
	for _, ts := range m.tags {
		for _, t := range ts {
			count[t]++
		}
	}
	names := slices.Sorted(maps.Keys(count))
	if len(names) == 0 {
		m.setStatus(i18n.T("Aún no hay etiquetas: pulsa t para etiquetar", "No tags yet: press t to tag"))
		return nil
	}
	items := make([]pickItem, len(names))
	for i, n := range names {
		n := n
		items[i] = pickItem{label: n, hint: fmt.Sprintf("%d", count[n]), run: func() tea.Cmd {
			return m.runSaved(savedSearch{Query: "tag:" + n, Recursive: true})
		}}
	}
	m.openPicker(i18n.T("Etiquetas", "Tags"), items)
	return nil
}
