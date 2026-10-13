package theme

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
)

var sgrSeq = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// PaintBackground pinta de bg todas las celdas de s que no tengan un fondo propio: cada línea
// empieza con ese fondo y, tras cada secuencia que lo borra (un reset o 49) sin poner otro,
// se vuelve a poner. Así no queda ninguna celda con el fondo de la terminal: ni los espacios
// entre paneles, ni el relleno de las líneas, ni lo que dibujan los popups sin fondo.
//
// With a text color, text that carries none takes it as well, the same way.
func PaintBackground(s string, bg color.Color, text ...color.Color) string {
	set, setFg := BackgroundSeq(bg), ""
	if len(text) > 0 {
		setFg = ForegroundSeq(text[0])
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = sgrSeq.ReplaceAllStringFunc(line, func(seq string) string {
			params := sgrSeq.FindStringSubmatch(seq)[1]
			if clearsBackground(params) {
				seq += set
			}
			if setFg != "" && clearsForeground(params) {
				seq += setFg
			}
			return seq
		})
		lines[i] = set + setFg + line + "\x1b[m"
	}
	return strings.Join(lines, "\n")
}

// clearsBackground indica si los parámetros de un SGR dejan el fondo sin color: un reset (vacío,
// 0 al principio) o 49, y sin fijar a la vez otro fondo (40-47, 48, 100-107).
func clearsBackground(params string) bool {
	parts := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(parts) == 0 {
		return true
	}
	clears := false
	for i := 0; i < len(parts); i++ {
		switch p := parts[i]; {
		case p == "38" || p == "58": // color de primer plano o subrayado: sus parámetros no son SGR sueltos
			if i+1 < len(parts) && parts[i+1] == "5" {
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			}
		case p == "48":
			return false
		case len(p) == 2 && (p[0] == '4' && p[1] >= '0' && p[1] <= '7'), len(p) == 3 && p[:2] == "10" && p[2] >= '0' && p[2] <= '7':
			return false
		case p == "0" || p == "00" || p == "49":
			clears = true
		}
	}
	return clears
}

// BackgroundSeq is the escape sequence that sets bg as the background color.
func BackgroundSeq(bg color.Color) string {
	r, g, b, _ := bg.RGBA()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}

// ForegroundSeq is the escape sequence that sets c as the text color.
func ForegroundSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}

// clearsForeground says whether the parameters of an SGR leave the text without a color: a reset or 39, without
// setting another one (30-37, 38, 90-97) in the same sequence.
func clearsForeground(params string) bool {
	parts := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	if len(parts) == 0 {
		return true
	}
	clears := false
	for i := 0; i < len(parts); i++ {
		switch p := parts[i]; {
		case p == "48" || p == "58":
			if i+1 < len(parts) && parts[i+1] == "5" {
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			}
		case p == "38":
			return false
		case len(p) == 2 && p[0] == '3' && p[1] >= '0' && p[1] <= '7', len(p) == 2 && p[0] == '9' && p[1] >= '0' && p[1] <= '7':
			return false
		case p == "0" || p == "00" || p == "39":
			clears = true
		}
	}
	return clears
}
