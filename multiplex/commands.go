package multiplex

import (
	"bytes"
	"io"

	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
)

func (m *Multiplexer) NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{
		m: m,
	}
}

type CommandBuffer struct {
	m   *Multiplexer
	buf bytes.Buffer

	setPos bool
	cursor state.Pos
	pen    *screen.ScreenPen
}

func (cb *CommandBuffer) samePen(pen *screen.ScreenPen) bool {
	if cb.pen == nil {
		if pen == nil {
			return true
		}

		return false
	}

	if pen == nil {
		return false
	}

	return cb.pen.Attrs() == pen.Attrs() &&
		cb.pen.FGColor() == pen.FGColor() &&
		cb.pen.BGColor() == pen.BGColor() &&
		cb.pen.Font() == pen.Font()
}

func (cb *CommandBuffer) SetCell(p state.Pos, val rune, pen *screen.ScreenPen) error {
	if !cb.setPos || cb.cursor != p {
		cb.m.ti.TPuts(&cb.buf, cb.m.ti.TGoto(p.Col, p.Row))
		cb.cursor = p
		cb.setPos = true
	}

	if pen != nil {
		if !cb.samePen(pen) {
			switch c := pen.FGColor().(type) {
			case state.IndexColor:
				cb.m.ti.TPuts(&cb.buf, cb.m.ti.TColor(c.Index, -1))
			case state.DefaultColor:
				cb.m.ti.TPuts(&cb.buf, cb.m.ti.AttrOff)
			}
		}

		cb.pen = pen
	}

	cb.buf.WriteRune(val)

	cb.cursor.Col++

	return nil
}

func (cb *CommandBuffer) Flush() error {
	_, err := io.Copy(cb.m.out, &cb.buf)
	if err != nil {
		return err
	}

	cb.buf.Reset()

	cb.setPos = false
	cb.pen = nil
	cb.m.curPos = cb.cursor

	return nil
}
