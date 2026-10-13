package preview

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// Probe reads facts about a media file (duration, size, codecs) with some tool. Tests inject a fake; SystemProbe
// is the macOS implementation. A Probe may be nil: then media files get the hex view.
type Probe func(ctx context.Context, path string) ([]string, error)

// Raster draws the first page of a document into a PNG file and returns its path. The caller removes it with
// Discard. Tests inject a fake; SystemRaster is the real one.
type Raster func(ctx context.Context, path string) (string, error)

const (
	toolTimeout = 5 * time.Second
	rasterDir   = "lazyfinder-page-"
)

var mediaExts = map[string]bool{
	"mp4": true, "m4v": true, "mov": true, "mkv": true, "avi": true, "webm": true,
	"mp3": true, "m4a": true, "wav": true, "flac": true, "aac": true, "ogg": true, "aiff": true,
}

func extOf(name string) string { return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) }

// SystemProbe asks Spotlight's metadata (`mdls`, macOS) about a file. It runs no shell and gives up after a few
// seconds.
func SystemProbe(ctx context.Context, path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "mdls",
		"-name", "kMDItemDurationSeconds", "-name", "kMDItemPixelWidth", "-name", "kMDItemPixelHeight",
		"-name", "kMDItemCodecs", path).Output()
	if err != nil {
		return nil, err
	}
	facts := ParseMdls(string(out))
	if len(facts) == 0 {
		return nil, errors.New("no media metadata")
	}
	return facts, nil
}

// ParseMdls turns the output of mdls into facts, dropping empty fields.
func ParseMdls(out string) []string {
	vals := map[string]string{}
	var codecs []string
	inList := false
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case inList && line == ")":
			inList = false
		case inList:
			codecs = append(codecs, strings.Trim(strings.TrimSuffix(line, ","), `"`))
		case strings.HasPrefix(line, "kMDItemCodecs") && strings.HasSuffix(line, "("):
			inList = true
		default:
			if k, v, ok := strings.Cut(line, "="); ok {
				if v = strings.TrimSpace(v); v != "(null)" {
					vals[strings.TrimSpace(k)] = v
				}
			}
		}
	}
	var facts []string
	if secs, err := strconv.ParseFloat(vals["kMDItemDurationSeconds"], 64); err == nil && secs > 0 {
		s := int(secs)
		d := fmt.Sprintf("%d:%02d", s/60, s%60)
		if s >= 3600 {
			d = fmt.Sprintf("%d:%02d:%02d", s/3600, s%3600/60, s%60)
		}
		facts = append(facts, fmt.Sprintf(i18n.T("Duración: %s", "Duration: %s"), d))
	}
	w, errW := strconv.Atoi(vals["kMDItemPixelWidth"])
	h, errH := strconv.Atoi(vals["kMDItemPixelHeight"])
	if errW == nil && errH == nil {
		facts = append(facts, fmt.Sprintf(i18n.T("Dimensiones: %d x %d", "Dimensions: %d x %d"), w, h))
	}
	if len(codecs) > 0 {
		facts = append(facts, fmt.Sprintf(i18n.T("Códecs: %s", "Codecs: %s"), strings.Join(codecs, ", ")))
	}
	return facts
}

// SystemRaster renders page 1 of a PDF with pdftoppm when it is installed, else with sips (macOS).
func SystemRaster(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout*2)
	defer cancel()
	dir, err := os.MkdirTemp("", rasterDir)
	if err != nil {
		return "", err
	}
	out := filepath.Join(dir, "page.png")
	var cmd *exec.Cmd
	if tool, err := exec.LookPath("pdftoppm"); err == nil {
		cmd = exec.CommandContext(ctx, tool, "-png", "-f", "1", "-l", "1", "-r", "110", "-singlefile", path, filepath.Join(dir, "page"))
	} else if tool, err := exec.LookPath("sips"); err == nil {
		cmd = exec.CommandContext(ctx, tool, "-s", "format", "png", path, "--out", out)
	} else {
		_ = os.RemoveAll(dir)
		return "", errors.New("neither pdftoppm nor sips is installed")
	}
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if _, err := os.Stat(out); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return out, nil
}

// Discard removes a page image made by SystemRaster together with its private folder. It refuses anything else.
func Discard(path string) {
	dir := filepath.Dir(path)
	if strings.HasPrefix(filepath.Base(dir), rasterDir) && filepath.Base(path) == "page.png" {
		_ = os.RemoveAll(dir)
	}
}
