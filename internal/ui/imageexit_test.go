package ui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
)

// flatten runs cmd and returns the messages it produces, in order; a sequence or a batch is run command by command.
func flatten(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var out []tea.Msg
		for i := range v.Len() {
			out = append(out, flatten(v.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// An image of the preview lives in the terminal, not in the frame: quitting with it on screen left it drawn over
// the shell, since the terminal keeps a Kitty image until it is told to delete it.
func TestQuittingWithAnImageDeletesItFirst(t *testing.T) {
	for _, key := range []string{"ctrl+c", "q"} {
		t.Run(key, func(t *testing.T) {
			e := newEnv(t)
			e.m.pv = previewState{open: true, id: 1, img: &imgState{id: 1, cols: 4, rows: 2}}
			_, cmd := e.m.Update(keyMsg(key))
			deleted := false
			for _, msg := range flatten(cmd) {
				switch msg := msg.(type) {
				case tea.RawMsg:
					deleted = deleted || msg.Msg == termimg.DeleteAllSequence()
				case tea.QuitMsg:
					if !deleted {
						t.Fatal("quit before deleting the image")
					}
					return
				}
			}
			t.Fatal("the key did not quit")
		})
	}
}
