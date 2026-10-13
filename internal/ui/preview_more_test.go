package ui

import (
	"archive/zip"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/preview"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func (e *env) addFile(name, body string) {
	e.t.Helper()
	if err := os.WriteFile(filepath.Join(e.root, name), []byte(body), 0o644); err != nil {
		e.t.Fatal(err)
	}
	e.key("r")
}

// withTools rebuilds the model with fake external tools.
func (e *env) withTools(p preview.Probe, r preview.Raster) {
	d := e.m.deps
	d.Probe, d.Raster = p, r
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
}

func TestSpaceOnABinaryFileShowsAHexDump(t *testing.T) {
	e := newEnv(t)
	e.addFile("blob.bin", "abc\x00\x01\x02def")
	e.clickText("blob.bin")
	e.key("space")
	e.mustHave("00000000  61 62 63 00 01 02 64 65  66")
	e.mustHave("|abc...def|")
}

func TestSpaceOnAnArchiveListsItsContents(t *testing.T) {
	e := newEnv(t)
	zp := filepath.Join(e.root, "pack.zip")
	f, err := os.Create(zp)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("docs/guide.txt")
	_, _ = w.Write([]byte("guide"))
	_ = zw.Close()
	_ = f.Close()
	e.key("r")
	e.clickText("pack.zip")
	e.key("space")
	e.mustHave("docs/guide.txt")
	e.mustHave("1 entry")
}

func TestSpaceOnMarkdownShowsItRendered(t *testing.T) {
	e := newEnv(t)
	e.addFile("readme.md", "# Big Title\n\n```go\nx := 1\n```\n")
	e.clickText("readme.md")
	e.key("space")
	e.mustHave("Big Title")
	e.mustNotHave("```")
}

func TestMarkdownRawFromTheConfigShowsTheSource(t *testing.T) {
	e := newEnv(t)
	d := e.m.deps
	d.Cfg.Preview.Markdown = "raw"
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.addFile("readme.md", "# Big Title\n\n```go\nx := 1\n```\n")
	e.clickText("readme.md")
	e.key("space")
	e.mustHave("```go")
}

func TestSpaceOnAVideoShowsTheProbedFacts(t *testing.T) {
	e := newEnv(t)
	e.withTools(func(context.Context, string) ([]string, error) { return []string{"Duration: 2:05"}, nil }, nil)
	e.addFile("clip.mp4", "\x00\x00\x00\x18ftyp")
	e.clickText("clip.mp4")
	e.key("space")
	e.mustHave("Duration: 2:05")
}

func TestExternalToolsOffNeverCallsTheProbe(t *testing.T) {
	e := newEnv(t)
	called := false
	e.withTools(func(context.Context, string) ([]string, error) { called = true; return []string{"x"}, nil }, nil)
	d := e.m.deps
	d.Cfg.Preview.ExternalTools = "off"
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.addFile("clip.mp4", "\x00\x00\x00\x18ftyp")
	e.clickText("clip.mp4")
	e.key("space")
	if called {
		t.Fatal("preview.external_tools off must not run any tool")
	}
	e.mustHave("00000000")
}

func TestPDFPageIsDrawnAsAnImageAndTheTempPageIsKept(t *testing.T) {
	e := newEnv(t)
	page := filepath.Join(t.TempDir(), "page.png")
	f, err := os.Create(page)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 60, 80))); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	e.withTools(nil, func(context.Context, string) (string, error) { return page, nil })
	e.addFile("doc.pdf", "%PDF-1.4\n\x00")
	e.send(uv.KittyGraphicsEvent{Options: kitty.Options{ID: 31}, Payload: []byte("OK")})
	e.clickText("doc.pdf")
	e.key("space")
	if !strings.ContainsRune(ansi.Strip(e.m.View().Content), 0x10EEEE) {
		t.Fatalf("the PDF page must be drawn with Kitty placeholders:\n%s", e.text())
	}
	if _, err := os.Stat(page); err != nil {
		t.Fatal("a file that is not one of our page images must never be removed")
	}
}
