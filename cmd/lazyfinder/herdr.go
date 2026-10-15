package main

import (
	"fmt"
	"io"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/herdr"
)

// runHerdr runs the step of the herdr plugin that herdr-plugin.toml asks for: ensure from the event hooks, toggle
// from the action.
func runHerdr(action string, getenv func(string) string, home string, stderr io.Writer) int {
	if action != "ensure" && action != "toggle" {
		fmt.Fprintf(stderr, "lazyfinder: unknown --herdr action %q (use ensure or toggle)\n", action)
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
	s := herdr.Settings{Dock: cfg.Herdr.Dock, AutoOpen: cfg.Herdr.AutoOpen, Width: cfg.Herdr.Width}
	e := herdr.Env{
		PluginID: getenv("HERDR_PLUGIN_ID"), TabID: getenv("HERDR_TAB_ID"),
		Event: getenv("HERDR_PLUGIN_EVENT_JSON"), StateDir: getenv("HERDR_PLUGIN_STATE_DIR"),
	}
	c := herdr.Client{Socket: sock}
	if action == "toggle" {
		err = herdr.Toggle(c, e, s)
	} else {
		err = herdr.Ensure(c, e, s)
	}
	if err != nil {
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
