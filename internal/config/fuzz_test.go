package config

import "testing"

// config.json is the user's, written by hand: any bytes must give a config or an error, never a panic.
func FuzzParse(f *testing.F) {
	for _, s := range []string{`{"version":1}`, `{}`, `[]`, `null`, `{"version":1,"openers":{"x":{"argv":["{file}"],"mode":"gui"}}}`, `{"version":1,"keys":{"quit":[""]}}`, `{"version":1,"sidebar":{"favorites":[{"name":"","path":"~"}]}}`, `{"version":1,"preview":{"max_text_bytes":-1}}`, `{"version":99}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data, "/home/u")
		_, _, _ = parseTolerant(data, "/home/u")
	})
}
