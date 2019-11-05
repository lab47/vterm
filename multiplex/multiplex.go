package multiplex

import (
	"io"
	"os"
	"os/exec"
	"unicode/utf8"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/gdamore/tcell/terminfo"
	"github.com/gdamore/tcell/terminfo/dynamic"
)

type Multiplexer struct {
	ti *terminfo.Terminfo
	st *terminal.State

	rows, cols int

	in, out *os.File

	buf []byte
}

func (m *Multiplexer) Init() error {
	m.in = os.Stdin
	m.out = os.Stdout

	ti, err := terminfo.LookupTerminfo(os.Getenv("TERM"))
	if err != nil {
		ti, _, err = dynamic.LoadTerminfo(os.Getenv("TERM"))
		if err != nil {
			return err
		}
		terminfo.AddTerminfo(ti)
	}

	m.ti = ti

	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		return err
	}

	m.rows = rows
	m.cols = cols

	m.buf = make([]byte, 32)

	st, err := terminal.MakeRaw(int(m.in.Fd()))
	if err != nil {
		return err
	}

	m.st = st

	m.ti.TPuts(m.out, m.ti.EnterCA)
	m.ti.TPuts(m.out, m.ti.EnableAcs)
	m.ti.TPuts(m.out, m.ti.Clear)

	m.DrawHorizLine(state.Pos{Row: 1, Col: 0}, cols)
	m.DrawVerticalLine(state.Pos{Row: 0, Col: 2}, rows)

	return nil
}

func (m *Multiplexer) Run(cmd *exec.Cmd) (io.Writer, error) {
	term, err := NewTerm(m, cmd)
	if err != nil {
		return nil, err
	}

	return term.Start(m.rows-2, m.cols-3, 2, 3)
}

func (m *Multiplexer) setCell(p state.Pos, val rune, pen *screen.ScreenPen) error {
	m.ti.TPuts(m.out, m.ti.TGoto(p.Col, p.Row))

	if pen != nil {
		switch c := pen.FGColor().(type) {
		case state.IndexColor:
			m.ti.TPuts(m.out, m.ti.TColor(c.Index, -1))
		case state.DefaultColor:
			m.ti.TPuts(m.out, m.ti.AttrOff)
		}
	}

	n := utf8.EncodeRune(m.buf, val)

	_, err := m.out.Write(m.buf[:n])
	return err
}

func (m *Multiplexer) moveCursor(p state.Pos) error {
	m.ti.TPuts(m.out, m.ti.TGoto(p.Col, p.Row))
	return nil
}

func (m *Multiplexer) Cleanup() {
	m.ti.TPuts(m.out, m.ti.AttrOff)
	m.ti.TPuts(m.out, m.ti.Clear)
	m.ti.TPuts(m.out, m.ti.ExitCA)

	terminal.Restore(int(m.in.Fd()), m.st)
}
