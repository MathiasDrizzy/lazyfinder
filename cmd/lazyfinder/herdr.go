package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/herdr"
)

// runHerdr runs the step of the herdr plugin that herdr-plugin.toml asks for: ensure from the event hooks, startup
// once herdr restored its session, toggle from the action.
func runHerdr(action string, getenv func(string) string, home string, stderr io.Writer) int {
	steps := map[string]func(herdr.Client, herdr.Env, herdr.Settings) error{
		"ensure": herdr.Ensure, "startup": herdr.Startup, "toggle": herdr.Toggle,
	}
	step, ok := steps[action]
	if !ok {
		fmt.Fprintf(stderr, "lazyfinder: unknown --herdr action %q (use ensure, startup or toggle)\n", action)
		return exitUsage
	}
	sock := getenv("HERDR_SOCKET_PATH")
	if sock == "" {
		fmt.Fprintln(stderr, "lazyfinder: --herdr runs inside herdr, which sets HERDR_SOCKET_PATH; it is not set")
		return exitRuntime
	}
	cfg, err := config.Load(config.ResolvePath(getenv, home), home)
	if err != nil {
		fmt.Fprintln(stderr, "lazyfinder: using default settings:", err)
		cfg = config.Defaults(home)
	}
	s := herdrSettings(cfg.Herdr)
	e := herdr.Env{
		PluginID: getenv("HERDR_PLUGIN_ID"), TabID: getenv("HERDR_TAB_ID"),
		Event: getenv("HERDR_PLUGIN_EVENT_JSON"), StateDir: getenv("HERDR_PLUGIN_STATE_DIR"),
	}
	if err := step(herdr.Client{Socket: sock}, e, s); err != nil {
		fmt.Fprintln(stderr, "lazyfinder:", err)
		return exitRuntime
	}
	return exitOK
}

// herdrFollower is set when lazyfinder runs as the pane of its herdr plugin: it returns the folder of the focused
// pane next to it, which the sidebar follows.
func herdrFollower(getenv func(string) string) func() string {
	sock, pane := getenv("HERDR_SOCKET_PATH"), getenv("HERDR_PANE_ID")
	if getenv("HERDR_PLUGIN_ID") != herdr.PluginID || sock == "" || pane == "" {
		return nil
	}
	c := herdr.Client{Socket: sock}
	return func() string {
		panes, err := c.Panes()
		if err != nil {
			return "" // herdr is busy or closing: keep the folder shown
		}
		return herdr.FollowDir(panes, pane)
	}
}

// herdrHider is set with herdrFollower: it keeps the tab of the pane without a sidebar once the user quits it, so the
// hooks do not open it again when the user comes back to the tab.
func herdrHider(getenv func(string) string, stderr io.Writer) func() {
	c, pane := herdr.Client{Socket: getenv("HERDR_SOCKET_PATH")}, getenv("HERDR_PANE_ID")
	e := herdr.Env{PluginID: getenv("HERDR_PLUGIN_ID"), StateDir: getenv("HERDR_PLUGIN_STATE_DIR")}
	return func() {
		if err := herdr.Hide(c, e, pane); err != nil {
			fmt.Fprintln(stderr, "lazyfinder:", err)
		}
	}
}

// herdrFitter keeps the pane of the plugin at the width of the settings it is given, one change at a time.
func herdrFitter(getenv func(string) string) func(config.Herdr, bool, bool) error {
	var mu sync.Mutex
	z := &herdr.Sizer{C: herdr.Client{Socket: getenv("HERDR_SOCKET_PATH")}, Pane: getenv("HERDR_PANE_ID")}
	return func(h config.Herdr, normal, force bool) error {
		mu.Lock()
		defer mu.Unlock()
		z.S = herdrSettings(h)
		return z.Fit(normal, force)
	}
}

// herdrOpener runs a terminal program in a new tab of herdr.
func herdrOpener(getenv func(string) string) func(string, []string, string) error {
	c := herdr.Client{Socket: getenv("HERDR_SOCKET_PATH")}
	e := herdr.Env{PluginID: getenv("HERDR_PLUGIN_ID"), StateDir: getenv("HERDR_PLUGIN_STATE_DIR"), WorkspaceID: getenv("HERDR_WORKSPACE_ID")}
	return func(dir string, argv []string, label string) error { return herdr.Open(c, e, dir, argv, label) }
}

// herdrSettings are the herdr settings of config.json as the plugin takes them.
func herdrSettings(h config.Herdr) herdr.Settings {
	return herdr.Settings{
		Dock: h.Dock, AutoOpen: h.AutoOpen, Columns: h.Columns, ExternalColumns: h.ExternalColumns,
		ExternalFrom: h.ExternalFrom, Adapt: h.Adapt, MinColumns: h.MinColumns, FitNormal: h.FitNormal,
	}
}
