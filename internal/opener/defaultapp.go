package opener

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultAppTimeout = 2 * time.Second

// launchServices asks macOS for the application that opens the file in argv[0]; the path is an argument, never part
// of the script.
const launchServices = `function run(argv) {
  ObjC.import('AppKit');
  var u = $.NSWorkspace.sharedWorkspace.URLForApplicationToOpenURL($.NSURL.fileURLWithPath(argv[0]));
  return u.isNil() ? '' : ObjC.unwrap(u.path);
}`

// DefaultApp names the application the system opens path with ("TextEdit"), or "" when it cannot tell. It is a
// variable so tests do not depend on the machine.
var DefaultApp = func(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAppTimeout)
	defer cancel()
	if goos == "linux" {
		mime, err := exec.CommandContext(ctx, "xdg-mime", "query", "filetype", path).Output()
		if err != nil {
			return ""
		}
		app, err := exec.CommandContext(ctx, "xdg-mime", "query", "default", strings.TrimSpace(string(mime))).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSuffix(strings.TrimSpace(string(app)), ".desktop")
	}
	out, err := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", launchServices, path).Output()
	if err != nil {
		return ""
	}
	return appName(string(out))
}

// appName is the name of the application at the path osascript printed, "" for none.
func appName(out string) string {
	if p := strings.TrimSpace(out); p != "" {
		return strings.TrimSuffix(filepath.Base(p), ".app")
	}
	return ""
}
