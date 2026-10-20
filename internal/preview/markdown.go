package preview

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"charm.land/glamour/v2"
)

// Markdown is rendered by glamour, which takes no context and has inputs a few hundred KB long that run for minutes,
// fill gigabytes or overflow the stack, which kills a Go program however it recovers: nested quotes, nested emphasis,
// unclosed links, tables of thousands of columns. Guessing them from the input missed some and refused good files, so
// the render runs in a child process, the same program started again with markdownChild set, with a time, memory and
// stack limit. Whatever goes wrong there, the preview shows the source.
const (
	markdownChild   = "LAZYFINDER_RENDER_MARKDOWN" // "<width>:<0 or 1 for no color>"
	markdownTimeout = 2 * time.Second
	maxChildHeap    = 512 << 20 // bytes
	maxChildStack   = 64 << 20  // bytes, instead of Go's 1 GB
	maxMarkdownOut  = 8 << 20   // bytes of rendered text
	maxMarkdownIn   = 16 << 20  // bytes of Markdown, above any preview.max_text_bytes worth rendering
)

// init serves the render when this program was started as the child. It runs before main and before TestMain, so a
// test binary started this way renders instead of running its tests again.
func init() {
	if spec, ok := os.LookupEnv(markdownChild); ok {
		os.Exit(serveMarkdown(spec, os.Stdin, os.Stdout))
	}
}

// serveMarkdown is the child: it renders the Markdown on in to out, and stops itself past the limits.
func serveMarkdown(spec string, in io.Reader, out io.Writer) int {
	debug.SetMaxStack(maxChildStack)
	go watchChild()
	w, nc, _ := strings.Cut(spec, ":")
	width, err := strconv.Atoi(w)
	if err != nil {
		return 2
	}
	src, err := io.ReadAll(io.LimitReader(in, maxMarkdownIn+1))
	if err != nil || len(src) > maxMarkdownIn {
		return 2
	}
	style := "dark"
	if nc == "1" {
		style = "notty"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(max(20, width)))
	if err != nil {
		return 2
	}
	rendered, err := r.Render(string(src))
	if err != nil {
		return 2
	}
	if _, err := io.WriteString(out, rendered); err != nil {
		return 2
	}
	return 0
}

// watchChild ends the child once its heap passes maxChildHeap, or once it outlives the time its parent waits, in case
// the parent is gone.
func watchChild() {
	deadline := time.Now().Add(2 * markdownTimeout)
	var ms runtime.MemStats
	for range time.Tick(20 * time.Millisecond) {
		runtime.ReadMemStats(&ms)
		if ms.HeapAlloc > maxChildHeap || time.Now().After(deadline) {
			os.Exit(3)
		}
	}
}

// renderMarkdown turns Markdown into styled lines wrapped at width, in a child process. Any failure, a limit passed
// or ctx cancelled returns ok=false and the caller shows the source instead.
func renderMarkdown(ctx context.Context, src string, width int, noColor bool) (lines []string, ok bool) {
	exe, err := os.Executable()
	if err != nil || len(src) > maxMarkdownIn {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, markdownTimeout)
	defer cancel()
	nc := "0"
	if noColor {
		nc = "1"
	}
	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d:%s", markdownChild, width, nc))
	cmd.Stdin = strings.NewReader(src)
	out := &capped{max: maxMarkdownOut}
	cmd.Stdout = out
	if err := cmd.Run(); err != nil || out.over {
		return nil, false
	}
	return strings.Split(strings.Trim(out.buf.String(), "\n"), "\n"), true
}

// capped keeps up to max bytes written to it and notes whether more came.
type capped struct {
	buf  strings.Builder
	max  int
	over bool
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); len(p) > room {
		c.buf.Write(p[:max(room, 0)])
		c.over = true
		return len(p), nil
	}
	c.buf.Write(p)
	return len(p), nil
}
