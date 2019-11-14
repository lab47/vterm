package multiplex

import (
	"io"
	"os"
	"os/exec"
	"unicode/utf8"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/parser"
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

	focusTerm  *Term
	focusInput io.Writer

	curPos state.Pos
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
	m.ti.TPuts(m.out, m.ti.TParm(m.ti.MouseMode, 1))

	m.DrawHorizLine(state.Pos{Row: 1, Col: 0}, cols)
	m.DrawVerticalLine(state.Pos{Row: 0, Col: 2}, rows)

	return nil
}

func (m *Multiplexer) Run(cmd *exec.Cmd) error {
	term, err := NewTerm(m, cmd)
	if err != nil {
		return err
	}

	w, err := term.Start(m.rows-2, m.cols-3, 2, 3)
	if err != nil {
		return err
	}

	m.focusTerm = term
	m.focusInput = w

	return nil
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
	if err != nil {
		return err
	}

	if m.curPos.Col < m.cols {
		m.curPos.Col++
	}

	return err
}

func (m *Multiplexer) moveCursor(p state.Pos) error {
	if m.curPos != p {
		m.ti.TPuts(m.out, m.ti.TGoto(p.Col, p.Row))
		m.curPos = p
	}

	return nil
}

func (m *Multiplexer) Cleanup() {
	m.ti.TPuts(m.out, m.ti.TParm(m.ti.MouseMode, 0))
	m.ti.TPuts(m.out, m.ti.AttrOff)
	m.ti.TPuts(m.out, m.ti.Clear)
	m.ti.TPuts(m.out, m.ti.ExitCA)

	terminal.Restore(int(m.in.Fd()), m.st)
}

func (m *Multiplexer) HandleEvent(ev parser.Event) error {
	var err error

	switch ev := ev.(type) {
	case *parser.TextEvent:
		_, err = m.focusInput.Write(ev.Text)
	case *parser.ControlEvent:
		_, err = m.focusInput.Write([]byte{ev.Control})
	default:
		err = nil
	}

	return err
}

func (m *Multiplexer) InputData(r io.Reader) error {
	/*
			pr, pw := io.Pipe()

			go func() {
				buf := make([]byte, 8)

				for {
					n, err := pr.Read(buf)
					if err != nil {
						return
					}

					q.Q(buf[:n])
				}
			}()

		x := io.TeeReader(r, pw)
	*/
	p, err := parser.NewParser(r, m)
	if err != nil {
		return err
	}

	return p.Drive()
}
