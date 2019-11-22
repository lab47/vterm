package multiplex

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/y0ssar1an/q"
)

type Term struct {
	m      *Multiplexer
	cmdbuf *CommandBuffer

	screen *screen.Screen
	cmd    *exec.Cmd

	rows, cols       int
	roffset, coffset int

	f *os.File

	damageLock    sync.Mutex
	pendingDamage []state.Rect

	cursorPos state.Pos

	used []int

	newDamage chan state.Rect
}

func NewTerm(m *Multiplexer, cmd *exec.Cmd) (*Term, error) {
	widget := &Term{
		m:         m,
		cmdbuf:    m.NewCommandBuffer(),
		cmd:       cmd,
		newDamage: make(chan state.Rect),
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

	w.updateUsed()
}

func (w *Term) updateUsed() {
	if len(w.used) == 0 {
		w.used = make([]int, w.cols)
	} else if w.cols > len(w.used) {
		for i := 0; i < w.cols-len(w.used); i++ {
			w.used = append(w.used, 0)
		}
	} else if len(w.used) > w.cols {
		w.used = w.used[:w.cols]
	}
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

	w.updateUsed()

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

	go w.draw()

	return nil
}

func (w *Term) draw() {
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()

	var damage []state.Rect

	for {
		select {
		case r := <-w.newDamage:
			if len(damage) == 0 && w.smallRect(r) {
				if !w.m.inputData.IsZero() {
					q.Q(time.Since(w.m.inputData).String())
					w.m.inputData = time.Time{}
				}
				w.applyDamage(r)
			} else {
				damage = append(damage, r)
			}
		case <-tick.C:
			for _, r := range damage {
				w.applyDamage(r)
			}

			damage = damage[:0]
		}
	}
}

const cellThreshold = 100

func (w *Term) smallRect(r state.Rect) bool {
	area := (r.End.Col - r.Start.Col) * (r.End.Row - r.Start.Row)

	return area < cellThreshold
}

func (w *Term) DamageDone(r state.Rect) error {
	// w.newDamage <- r

	// return nil

	// w.damageLock.Lock()
	// defer w.damageLock.Unlock()

	/*
		if !w.m.inputData.IsZero() {
			q.Q(time.Since(w.m.inputData).String())
			w.m.inputData = time.Time{}
		}
	*/

	return w.applyDamage(r)

	// w.pendingDamage = append(w.pendingDamage, r)

	return nil
}

func (w *Term) applyDamage(r state.Rect) error {
	defer w.MoveCursor(w.cursorPos)
	defer w.cmdbuf.Flush()

	for row := r.Start.Row; row <= r.End.Row; row++ {
		// used := w.used[row]
		max := -1

		for col := r.Start.Col; col <= r.End.Col; col++ {
			cell, err := w.screen.GetCell(row, col)
			if err != nil {
				return err
			}

			val, _ := cell.Value()

			/*
				if val == 0 {
					if col < used {
						w.cmdbuf.SetCell(state.Pos{Row: row + w.roffset, Col: col + w.coffset}, ' ', cell.Pen())
					}
				} else {
					max = col
				}
			*/

			if val == 0 {
				val = ' '
			}

			w.cmdbuf.SetCell(state.Pos{Row: row + w.roffset, Col: col + w.coffset}, val, cell.Pen())
		}

		w.used[row] = max + 1
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
