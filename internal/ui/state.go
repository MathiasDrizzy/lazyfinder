package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// stateFile is the path of a small JSON file of remembered things (recent folders, saved searches) kept next to
// config.json. It is empty when the app has no config path, as in tests, so nothing is written to the user's home.
func (m *Model) stateFile(name string) string {
	if m.deps.ConfigPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(m.deps.ConfigPath), name)
}

func loadState(path string, into any) {
	if path == "" {
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, into) // a damaged file simply starts the list again
	}
}

// saveState writes v as JSON, readable only by the user (it says which files were opened and where). A failure is
// not worth interrupting the user for: the value still works for this session.
func saveState(path string, v any) {
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, append(data, '\n'), 0o600)
	_ = os.Chmod(path, 0o600) // WriteFile keeps the mode of a file that already exists
}
