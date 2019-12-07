package multiplex

import (
	"unicode/utf8"

	"github.com/evanphx/vterm/state"
)

const (
	vbar  = '│'
	hbar  = '─'
	ltee  = '┤'
	rtee  = '├'
	cross = '┼'
)

var (
	vbarBytes  []byte
	hbarBytes  []byte
	lteeBytes  []byte
	rteeBytes  []byte
	crossBytes []byte
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

	bytes = make([]byte, 8)
	n = utf8.EncodeRune(bytes, rtee)

	rteeBytes = bytes[:n]

	bytes = make([]byte, 8)
	n = utf8.EncodeRune(bytes, cross)

	crossBytes = bytes[:n]
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

func (m *Multiplexer) DrawRightTee(p state.Pos) error {
	m.moveCursor(p)
	m.out.Write(rteeBytes)
	return nil
}
