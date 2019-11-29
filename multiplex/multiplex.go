package multiplex

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/pkg/terminfo"
	"github.com/evanphx/vterm/pkg/terminfo/dynamic"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/y0ssar1an/q"
)

const mouseMode = "%?%p1%{1}%=%t%'h'%Pa%e%'l'%Pa%;\x1b[?1000%ga%c\x1b[?1002%ga%c\x1b[?1006%ga%c"

type Multiplexer struct {
	Config Config

	ti *terminfo.Terminfo
	st *terminal.State

	rows, cols int

	outMu sync.Mutex

	out io.Writer

	resetFd int

	buf []byte

	layout *Layout

	curPos state.Pos

	inputData time.Time
}

func (m *Multiplexer) Init() error {
	var (
		in  = os.Stdin
		out = os.Stdout
	)

	if len(m.Config.Shell) == 0 {
		m.Config.Shell = []string{os.Getenv("SHELL")}
	}

	m.out = out
	m.resetFd = int(in.Fd())

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

	st, err := terminal.MakeRaw(m.resetFd)
	if err != nil {
		return err
	}

	m.st = st

	m.ti.TPuts(m.out, m.ti.EnterCA)
	m.ti.TPuts(m.out, m.ti.EnableAcs)
	// m.ti.TPuts(m.out, m.ti.Clear)
	m.ti.TPuts(m.out, m.ti.TParm(mouseMode, 1))

	// m.DrawHorizLine(state.Pos{Row: 1, Col: 0}, cols)
	// m.DrawVerticalLine(state.Pos{Row: 0, Col: 2}, rows)

	return nil
}

func (m *Multiplexer) RunShell() error {
	shell := m.Config.Shell
	cmd := exec.Command(shell[0], shell[1:]...)
	cmd.Env = append(os.Environ(), m.Config.Env...)

	return m.Run(cmd)
}

func (m *Multiplexer) Run(cmd *exec.Cmd) error {
	term, err := NewTerm(m, cmd)
	if err != nil {
		return err
	}

	layout, err := NewLayout(m, term, m.rows, m.cols)
	if err != nil {
		return err
	}

	m.layout = layout

	return m.layout.Start()
}

func (m *Multiplexer) Redraw() error {
	// m.ti.TPuts(m.out, m.ti.Clear)

	return m.layout.Draw(m.out)
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
	m.outMu.Lock()
	defer m.outMu.Unlock()

	if m.curPos != p {
		m.ti.TParmf(m.out, m.ti.SetCursor, p.Row, p.Col)
		// m.ti.TPuts(m.out, m.ti.TGoto(p.Col, p.Row))
		m.curPos = p
	}

	return nil
}

func (m *Multiplexer) Cleanup() {
	m.ti.TPuts(m.out, m.ti.TParm(mouseMode, 0))
	m.ti.TPuts(m.out, m.ti.AttrOff)
	// m.ti.TPuts(m.out, m.ti.Clear)
	m.ti.TPuts(m.out, m.ti.ExitCA)

	if m.resetFd != -1 {
		terminal.Restore(m.resetFd, m.st)
	}
}

func (m *Multiplexer) HandleInput(ev Event) error {
	var err error

	switch ev := ev.(type) {
	case TextEvent:
		_, err = m.layout.Write([]byte(ev))
	case ControlEvent:
		q.Q(ev)

		if ev == 0x1 {
			m.layout.Operations.Split()
		} else {
			_, err = m.layout.Write([]byte{byte(ev)})
		}

	default:
		err = nil
	}

	return err
}

type timerReader struct {
	io.Reader
	m *Multiplexer
}

func (t *timerReader) Read(b []byte) (int, error) {
	n, err := t.Reader.Read(b)

	q.Q(b[:n])

	t.m.inputData = time.Now()

	return n, err
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
	ip, err := NewInputReader(r, m)
	// p, err := parser.NewParser(&timerReader{r, m}, m)
	if err != nil {
		return err
	}

	return ip.Drive()
}
