// Command lazyfinder is a lazy file manager for the terminal.
package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
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
	Logger  *slog.Logger
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
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: lazyfinder [flags] [directory]\n\nFlags:\n")
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

	startDir, err := startDirectory(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, "lazyfinder:", err)
		return exitRuntime
	}
	opts := Options{Start: startDir, Home: home, NoColor: getenv("NO_COLOR") != ""}
	cfg, err := config.Load(config.ResolvePath(getenv, home), home)
	if err != nil {
		opts.Warning = err.Error()
		fmt.Fprintln(stderr, "lazyfinder: using default settings:", err)
		cfg = config.Defaults(home)
	}
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
	m := ui.New(ui.Deps{Home: o.Home, Start: o.Start, Cfg: o.Cfg, NoColor: o.NoColor, Warning: o.Warning})
	_, err := tea.NewProgram(m).Run()
	return err
}
