package multiplex

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/y0ssar1an/q"
)

type Term struct {
	id     int
	m      *Multiplexer
	cmdbuf *CommandBuffer

	mu sync.Mutex

	parser *parser.Parser
	screen *screen.Screen
	state  *state.State
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

var termId int32

func NewTerm(m *Multiplexer, cmd *exec.Cmd) (*Term, error) {
	widget := &Term{
		id:        int(atomic.AddInt32(&termId, 1)),
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

func (w *Term) ResizeMoved(rows, cols, rowsOffset, colsOffset int) {
	// Resize the parser without the lock because that goroutine might
	// call back into DamageDone and need the lock. If that happens, we'll
	// get a deadlock from inversion, so instead just perform the parse
	// stack resize, then take the lock and update the layout data.
	w.parser.Resize(context.TODO(), rows, cols)

	w.mu.Lock()
	defer w.mu.Unlock()

	w.cols, w.rows = rows, cols
	w.coffset = colsOffset
	w.roffset = rowsOffset

	q.Q(w.id, rows, cols, rowsOffset, colsOffset)

	// pty.Setsize(w.f, w.currentSize())

	w.updateUsed()

	w.applyDamage(state.Rect{Start: state.Pos{0, 0}, End: state.Pos{rows - 1, cols - 1}})
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

func (w *Term) Write(b []byte) (int, error) {
	return w.f.Write(b)
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
		// Setting them here prevents the race detector from worrying about
		// it when we use w.screen via the screen output handler
		w.screen = screen
		w.state = st
		w.parser = parser

		err := parser.Drive(context.TODO())
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
		}
	}()

	// go w.draw()

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
	w.mu.Lock()
	defer w.mu.Unlock()

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

var dam sync.Mutex

func (w *Term) applyDamage(r state.Rect) error {
	dam.Lock()
	defer dam.Unlock()

	defer w.moveCursor(w.cursorPos)
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

			abRow := row + w.roffset
			abCol := col + w.coffset

			if abRow == 0 && abCol > 40 && abCol < 46 {
				s := string(val)
				q.Q(w.id, val, s, abRow, abCol)
			}

			w.cmdbuf.SetCell(state.Pos{Row: abRow, Col: abCol}, val, cell.Pen())
		}

		w.used[row] = max + 1
	}

	return nil
}

func (w *Term) MoveCursor(p state.Pos) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.m.moveCursor(p)
}

func (w *Term) moveCursor(p state.Pos) error {
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
