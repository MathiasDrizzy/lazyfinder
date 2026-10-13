// Command lazyfinder is a lazy file manager for the terminal.
package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/preview"
	"github.com/MathiasDrizzy/lazyfinder/internal/search"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// Exit codes (part of the public API, see docs/SPEC.md).
const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

// Options is everything the UI needs to start.
type Options struct {
	Start   string
	Home    string
	Cfg     config.Config
	Warning string
	NoColor bool
	Follow  bool // --follow: the narrow project tree
	Logger  *slog.Logger

	ConfigPath string
}

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr, launch))
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer, start func(Options) error) int {
	fs := flag.NewFlagSet("lazyfinder", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")
	printDefault := fs.Bool("print-default-config", false, "print the default config.json and exit")
	logPath := fs.String("log", "", "write a JSON log to `file` (off by default)")
	logPaths := fs.Bool("log-paths", false, "include file names in the log")
	langFlag := fs.String("lang", "", "interface language for this run: auto, en, es, pt, fr, de, it, ja or zh")
	themeFlag := fs.String("theme", "", "theme for this run: a built-in or user theme name, or terminal")
	followDir := fs.String("follow", "", "follow a project in a narrow tree with live updates and git state: `directory`")
	noMouse := fs.Bool("no-mouse", false, "ignore the mouse for this run")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: lazyfinder [flags] [directory]\n       lazyfinder --follow directory\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return exitOK
		}
		return exitUsage
	}
	home := getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	switch {
	case *showVersion:
		fmt.Fprintf(stdout, "lazyfinder %s\n", version)
		return exitOK
	case *printDefault:
		_, _ = stdout.Write(config.DefaultJSON(home))
		return exitOK
	}

	args = fs.Args()
	if *followDir != "" {
		if len(args) > 0 {
			fmt.Fprintln(stderr, "lazyfinder: --follow takes its folder as its own value; drop the extra argument")
			return exitUsage
		}
		args = []string{*followDir}
	}
	startDir, err := startDirectory(args)
	if err != nil {
		fmt.Fprintln(stderr, "lazyfinder:", err)
		return exitRuntime
	}
	opts := Options{Start: startDir, Home: home, NoColor: getenv("NO_COLOR") != "", Follow: *followDir != ""}
	opts.ConfigPath = config.ResolvePath(getenv, home)
	cfg, err := config.Load(opts.ConfigPath, home)
	if err != nil {
		opts.Warning = err.Error()
		fmt.Fprintln(stderr, "lazyfinder: using default settings:", err)
		cfg = config.Defaults(home)
	}
	if *langFlag != "" {
		if _, ok := i18n.Parse(*langFlag); !ok && *langFlag != "auto" {
			fmt.Fprintf(stderr, "lazyfinder: unknown language %q (use auto, en, es, pt, fr, de, it, ja or zh)\n", *langFlag)
			return exitUsage
		}
		cfg.Language = *langFlag
	}
	if *themeFlag != "" {
		if !theme.Valid(*themeFlag) {
			fmt.Fprintf(stderr, "lazyfinder: unknown theme %q (use %s)\n", *themeFlag, strings.Join(theme.Names(), ", "))
			return exitUsage
		}
		cfg.Theme = *themeFlag
	}
	if *noMouse {
		cfg.Mouse = false
	}
	if opts.Follow {
		cfg.Columns = []string{"name"}
	}
	i18n.SetLanguage(cfg.Language)
	opts.Cfg = cfg
	logger, closeLog, err := newLogger(*logPath, *logPaths, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "lazyfinder:", err)
		return exitRuntime
	}
	defer closeLog()
	opts.Logger = logger
	logger.Info("start", "version", version)
	if err := start(opts); err != nil {
		logger.Error("exit", "error", err.Error())
		fmt.Fprintln(stderr, "lazyfinder:", err)
		return exitRuntime
	}
	logger.Info("exit")
	return exitOK
}

func startDirectory(args []string) (string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cannot open %q: %w", dir, unwrap(err))
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", dir)
	}
	return abs, nil
}

func unwrap(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}

// newLogger returns a JSON logger writing to path, or a discarding one when path is empty. Names are not logged
// unless withPaths is set.
func newLogger(path string, withPaths bool, _ io.Writer) (*slog.Logger, func(), error) {
	if path == "" {
		return slog.New(slog.NewJSONHandler(io.Discard, nil)), func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open log file: %w", unwrap(err))
	}
	h := slog.NewJSONHandler(f, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if !withPaths && (a.Key == "path" || a.Key == "name") {
			return slog.Attr{}
		}
		return a
	}})
	return slog.New(h), func() { _ = f.Close() }, nil
}

func launch(o Options) error {
	opt := watch.DefaultOptions()
	opt.MaxDirs = o.Cfg.Watch.MaxDirs
	opt.Poll = time.Duration(o.Cfg.Watch.PollIntervalMS) * time.Millisecond
	var w watch.Watcher = watch.Noop{}
	if fw, err := watch.NewFS(opt); err == nil { // without a watcher the app still works, it just does not follow
		defer func() { _ = fw.Close() }()
		w = fw
	}
	d := ui.Deps{Home: o.Home, Start: o.Start, Cfg: o.Cfg, NoColor: o.NoColor, Warning: o.Warning, ConfigPath: o.ConfigPath, Watcher: w, Follow: o.Follow}
	if o.Cfg.Preview.ExternalTools != "off" {
		d.Probe, d.Raster = preview.SystemProbe, preview.SystemRaster
	}
	if o.Cfg.Search.ContentTool == "auto" {
		d.ContentFind = search.SystemRg
	}
	m := ui.New(d)
	_, err := tea.NewProgram(m).Run()
	return err
}
