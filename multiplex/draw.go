package multiplex

import (
	"unicode/utf8"

	"github.com/evanphx/vterm/state"
)

const (
	vbar = '│'
	hbar = '─'
	ltee = '┤'
)

var (
	vbarBytes []byte
	hbarBytes []byte
	lteeBytes []byte
)

func init() {
	bytes := make([]byte, 8)
	n := utf8.EncodeRune(bytes, vbar)

	vbarBytes = bytes[:n]

	bytes = make([]byte, 8)
	n = utf8.EncodeRune(bytes, hbar)

	hbarBytes = bytes[:n]

	bytes = make([]byte, 8)
	n = utf8.EncodeRune(bytes, ltee)

	lteeBytes = bytes[:n]
}

func (m *Multiplexer) DrawVerticalLine(p state.Pos, dist int) error {
	for i := 0; i < dist; i++ {
		m.moveCursor(p)
		m.out.Write(vbarBytes)
		p.Row++
	}
	return nil
}

func (m *Multiplexer) DrawHorizLine(p state.Pos, dist int) error {
	for i := 0; i < dist; i++ {
		m.moveCursor(p)
		m.out.Write(hbarBytes)
		p.Col++
	}
	return nil
}
