// Package keymap names the actions of the file list and maps keys to them. The defaults are the ones in
// docs/keybindings.md; the config's "keys" object replaces the keys of an action (docs/adr/0008).
package keymap

import (
	"fmt"
	"slices"
	"strings"
)

// Action is something the user can do from the list.
type Action string

// The actions of the list, in the order the help shows them.
const (
	Up               Action = "up"
	Down             Action = "down"
	PageUp           Action = "page_up"
	PageDown         Action = "page_down"
	First            Action = "first"
	Last             Action = "last"
	Open             Action = "open"
	Parent           Action = "parent"
	Back             Action = "back"
	Forward          Action = "forward"
	Preview          Action = "preview"
	OpenWith         Action = "open_with"
	QuickLook        Action = "quick_look"
	Sort             Action = "sort"
	SortReverse      Action = "sort_reverse"
	Hidden           Action = "hidden"
	Reload           Action = "reload"
	Ignored          Action = "ignored"
	Search           Action = "search"
	Goto             Action = "goto"
	Expand           Action = "expand"
	Collapse         Action = "collapse"
	Mark             Action = "mark"
	MarkAll          Action = "mark_all"
	Copy             Action = "copy"
	Cut              Action = "cut"
	Paste            Action = "paste"
	Duplicate        Action = "duplicate"
	CopyPath         Action = "copy_path"
	Rename           Action = "rename"
	Chmod            Action = "chmod"
	NewFolder        Action = "new_folder"
	NewFile          Action = "new_file"
	Trash            Action = "trash"
	Undo             Action = "undo"
	Info             Action = "info"
	Layout           Action = "layout"
	TagKey           Action = "tag"
	Tags             Action = "tags"
	SavedSearches    Action = "saved_searches"
	Filter           Action = "filter"
	Commands         Action = "commands"
	Recents          Action = "recents"
	Bookmark         Action = "bookmark"
	NewTab           Action = "new_tab"
	CloseTab         Action = "close_tab"
	NextTab          Action = "next_tab"
	PrevTab          Action = "prev_tab"
	SidebarNarrow    Action = "sidebar_narrow"
	SidebarWiden     Action = "sidebar_widen"
	ColumnNarrow     Action = "column_narrow"
	ColumnWiden      Action = "column_widen"
	FavoritesShorter Action = "favorites_shorter"
	FavoritesTaller  Action = "favorites_taller"
	Help             Action = "help"
	Focus            Action = "focus"
	Settings         Action = "settings"
	Project          Action = "project"
	Branch           Action = "branch"
	Quit             Action = "quit"
)

// Actions lists every action.
var Actions = []Action{
	Up, Down, PageUp, PageDown, First, Last, Open, Parent, Back, Forward, Preview, OpenWith, QuickLook,
	Sort, SortReverse, Hidden, Ignored, Search, Goto, Expand, Collapse, Mark, MarkAll, Copy, Cut, Paste, Duplicate, CopyPath, Rename, Chmod, NewFolder, NewFile, Trash, Undo, Info, Reload, Layout, TagKey, Tags, SavedSearches, Filter, Commands, Recents, Bookmark, NewTab, CloseTab, NextTab, PrevTab, SidebarNarrow, SidebarWiden, ColumnNarrow, ColumnWiden, FavoritesShorter, FavoritesTaller, Help, Focus, Settings, Project, Branch, Quit,
}

// Defaults are the keys of each action when the config says nothing. Key names are Bubble Tea's.
var Defaults = map[Action][]string{
	Up: {"up", "k"}, Down: {"down", "j"}, PageUp: {"pgup", "ctrl+u"}, PageDown: {"pgdown", "ctrl+d"}, First: {"home", "g"},
	Last: {"end", "G"}, Open: {"enter", "right", "l"}, Parent: {"backspace", "left", "h"},
	Back: {"alt+left", "["}, Forward: {"alt+right", "]"}, Preview: {"space"}, OpenWith: {"O"}, QuickLook: {"Q"},
	Sort: {"s"}, SortReverse: {"S"}, Hidden: {"."}, Ignored: {"I"}, Search: {"/", "ctrl+f"}, Goto: {"ctrl+g"}, Expand: {"+", "="}, Collapse: {"-"},
	Mark: {"v"}, MarkAll: {"ctrl+a"}, Copy: {"c"}, Cut: {"x"}, Paste: {"p", "ctrl+v"}, Duplicate: {"D"}, CopyPath: {"y"},
	Rename: {"f2", "R"}, Chmod: {"P"}, NewFolder: {"N", "ctrl+n"}, NewFile: {"n"}, Trash: {"d", "delete"}, Undo: {"u", "ctrl+z"}, Info: {"i"},
	Reload: {"r"}, Layout: {"M"}, TagKey: {"t"}, Tags: {"T"}, SavedSearches: {"F"}, Filter: {"f"}, Commands: {"!"}, Recents: {"E"}, Bookmark: {"b"}, NewTab: {"ctrl+t"}, CloseTab: {"ctrl+w"}, NextTab: {"}"}, PrevTab: {"{"}, SidebarNarrow: {"<"}, SidebarWiden: {">"}, ColumnNarrow: {"ctrl+left"}, ColumnWiden: {"ctrl+right"}, FavoritesShorter: {"ctrl+up"}, FavoritesTaller: {"ctrl+down"}, Help: {"?", "f1"}, Focus: {"tab"}, Settings: {","}, Project: {"ctrl+b"}, Branch: {"B"}, Quit: {"q"},
}

// The two keybinding modes: dual keeps the Vim letters next to the arrows, lazy leaves only the arrows and the
// other keys.
const (
	ModeDual = "dual"
	ModeLazy = "lazy"
)

var vimLetters = map[Action][]string{Up: {"k"}, Down: {"j"}, Open: {"l"}, Parent: {"h"}, First: {"g"}, Last: {"G"}}

// WithMode returns the user's overrides plus, in lazy mode, the default keys of the Vim actions without their
// letters. An action the user already rebound keeps the user's keys.
func WithMode(overrides map[string][]string, mode string) map[string][]string {
	if mode != ModeLazy {
		return overrides
	}
	out := make(map[string][]string, len(overrides)+len(vimLetters))
	for a, letters := range vimLetters {
		out[string(a)] = slices.DeleteFunc(slices.Clone(Defaults[a]), func(k string) bool { return slices.Contains(letters, k) })
	}
	for a, ks := range overrides {
		out[a] = ks
	}
	return out
}

// Map is the effective binding. Build it with New; it is read-only afterwards.
type Map struct {
	byKey    map[string]Action
	byAction map[Action][]string
}

// New merges the user's overrides into the defaults. A problem (unknown action, empty key, a key that two actions
// would share) is returned as a warning and the offending override is dropped, so the app always starts.
func New(overrides map[string][]string) (Map, []string) {
	var warns []string
	eff := map[Action][]string{}
	for a, ks := range Defaults {
		eff[a] = slices.Clone(ks)
	}
	names := make([]string, 0, len(overrides))
	for n := range overrides {
		names = append(names, n)
	}
	slices.Sort(names)
	var applied []Action
	for _, n := range names {
		a := Action(n)
		if _, ok := Defaults[a]; !ok {
			warns = append(warns, fmt.Sprintf("keys: unknown action %q (ignored)", n))
			continue
		}
		ks := overrides[n]
		if len(ks) == 0 || slices.ContainsFunc(ks, func(k string) bool { return strings.TrimSpace(k) == "" }) {
			warns = append(warns, fmt.Sprintf("keys: action %q needs a non-empty list of keys (default kept)", n))
			continue
		}
		eff[a] = slices.Clone(ks)
		applied = append(applied, a)
	}
	// A key may belong to one action only. Overrides that clash are undone one at a time, in a fixed order.
	for {
		owner := map[string]Action{}
		clash := ""
		var culprit Action
		for _, a := range Actions {
			for _, k := range eff[a] {
				if prev, dup := owner[k]; dup && clash == "" {
					clash = k
					culprit = a
					if !slices.Contains(applied, a) {
						culprit = prev
					}
				}
				owner[k] = a
			}
		}
		if clash == "" {
			break
		}
		if !slices.Contains(applied, culprit) { // two defaults cannot clash (tested), so this is unreachable
			break
		}
		warns = append(warns, fmt.Sprintf("keys: %q is bound to two actions; the default keys of %q stay", clash, culprit))
		eff[culprit] = slices.Clone(Defaults[culprit])
		applied = slices.DeleteFunc(applied, func(x Action) bool { return x == culprit })
	}
	m := Map{byKey: map[string]Action{}, byAction: eff}
	for _, a := range Actions {
		for _, k := range eff[a] {
			m.byKey[k] = a
		}
	}
	return m, warns
}

// Lookup returns the action bound to key.
func (m Map) Lookup(key string) (Action, bool) {
	a, ok := m.byKey[key]
	return a, ok
}

// Keys returns the keys of an action, the first being the one to show in hints.
func (m Map) Keys(a Action) []string { return m.byAction[a] }
