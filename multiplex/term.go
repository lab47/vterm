package multiplex

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
)

type Term struct {
	m *Multiplexer

	screen *screen.Screen
	cmd    *exec.Cmd

	rows, cols       int
	roffset, coffset int

	f *os.File

	damageLock    sync.Mutex
	pendingDamage []state.Rect

	cursorPos state.Pos
}

func NewTerm(m *Multiplexer, cmd *exec.Cmd) (*Term, error) {
	widget := &Term{
		m:   m,
		cmd: cmd,
	}

	return widget, nil
}

// Draw is called to inform the widget to draw itself.  A containing
// Term will generally call this during the application draw loop.
func (w *Term) Draw() {
	w.damageLock.Lock()
	defer w.damageLock.Unlock()

	for _, r := range w.pendingDamage {
		w.applyDamage(r)
	}

	w.pendingDamage = w.pendingDamage[:0]
}

func (w *Term) Resize(rows, cols int) {
	w.cols, w.rows = rows, cols
	pty.Setsize(w.f, w.currentSize())
}

func (w *Term) currentSize() *pty.Winsize {
	var ws pty.Winsize
	ws.Cols = uint16(w.cols)
	ws.Rows = uint16(w.rows)

	return &ws
}

func (w *Term) Start(rows, cols, roffset, coffset int) (io.Writer, error) {
	w.rows = rows
	w.cols = cols
	w.roffset = roffset
	w.coffset = coffset

	out, err := pty.StartWithSize(w.cmd, w.currentSize())
	if err != nil {
		return nil, err
	}

	w.f = out

	return w.f, w.begin()
}

// Size returns the size of the widget (content size) as width, height
// in columns.  Layout managers should attempt to ensure that at least
// this much space is made available to the View for this Term.  Extra
// space may be allocated on as an needed basis.
func (w *Term) Size() (int, int) {
	return w.cols, w.rows
}

func (w *Term) begin() error {
	screen, err := screen.NewScreen(w.rows, w.cols, w)
	if err != nil {
		return err
	}

	st, err := state.NewState(w.rows, w.cols, screen)
	if err != nil {
		return err
	}

	parser, err := parser.NewParser(w.f, st)
	if err != nil {
		return err
	}

	go func() {
		err := parser.Drive()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
		}
	}()

	w.screen = screen

	return nil
}

func (w *Term) DamageDone(r state.Rect) error {
	w.damageLock.Lock()
	defer w.damageLock.Unlock()

	return w.applyDamage(r)

	// w.pendingDamage = append(w.pendingDamage, r)

	return nil
}

func (w *Term) applyDamage(r state.Rect) error {
	defer w.MoveCursor(w.cursorPos)

	for row := r.Start.Row; row <= r.End.Row; row++ {
		for col := r.Start.Col; col <= r.End.Col; col++ {
			cell, err := w.screen.GetCell(row, col)
			if err != nil {
				return err
			}

			val, _ := cell.Value()

			if val == 0 {
				val = ' '
			}

			w.m.setCell(state.Pos{Row: row + w.roffset, Col: col + w.coffset}, val, cell.Pen())
		}
	}

	return nil
}

func (w *Term) MoveCursor(p state.Pos) error {
	w.cursorPos = p
	p.Row += w.roffset
	p.Col += w.coffset
	return w.m.moveCursor(p)
}

func (w *Term) SetTermProp(attr state.TermAttr, val interface{}) error {
	return nil
}

func (w *Term) Output(data []byte) error {
	_, err := w.f.Write(data)
	return err
}

func (w *Term) StringEvent(kind string, data []byte) error {
	return nil
}
