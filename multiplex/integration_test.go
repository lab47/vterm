package multiplex

import (
	"context"
	"io"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/pkg/terminfo"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
	"github.com/y0ssar1an/q"

	_ "github.com/evanphx/vterm/pkg/terminfo/a/ansi"
)

type integrationOutput struct {
	screen *screen.Screen
}

func (i *integrationOutput) DamageDone(r state.Rect) error {
	return nil

	for row := r.Start.Row; row <= r.End.Row; row++ {
		for col := r.Start.Col; col <= r.End.Col; col++ {
			cell, err := i.screen.GetCell(row, col)
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

			if row == 2 && col == 0 {
				if val != 's' {
					q.Q(val)
					panic("huh")
				}
			}

			/*
				if val == 0 {
					if col < 10 && row < 4 {
						q.Q(0, row, col)
					}
					val = ' '
				} else {
					s := string(val)
					q.Q(val, s, row, col)
				}
			*/
		}
	}

	return nil
}

func (i *integrationOutput) MoveCursor(pos state.Pos) error {
	// q.Q(pos)
	return nil
}

func (i *integrationOutput) SetTermProp(attr state.TermAttr, val interface{}) error {
	return nil
}

func (i *integrationOutput) Output(data []byte) error {
	return nil
}

func (i *integrationOutput) StringEvent(kind string, data []byte) error {
	return nil
}

func TestIntegration(t *testing.T) {
	n := neko.Modern(t)

	var (
		rows = 25
		cols = 80
	)

	getRange := func(s *screen.Screen, row, col, sz int) (string, error) {
		var ret string

		for i := 0; i < sz; i++ {
			cell, err := s.GetCell(row, col+i)
			if err != nil {
				return "", err
			}

			val, _ := cell.Value()

			ret += string(val)
		}

		return ret, nil
	}

	assertRange := func(t *testing.T, s *screen.Screen, row int, col int, expected string) {
		l := utf8.RuneCountInString(expected)
		val, err := getRange(s, row, col, l)
		require.NoError(t, err)

		assert.Equal(t, expected, val)
	}

	n.It("can setup a split", func(t *testing.T) {
		var i integrationOutput
		scr, err := screen.NewScreen(rows, cols, &i)
		require.NoError(t, err)

		i.screen = scr

		st, err := state.NewState(rows, cols, scr)
		require.NoError(t, err)

		// st.Debug = true

		r, w := io.Pipe()

		par, err := parser.NewParser(r, st)
		require.NoError(t, err)

		go par.Drive(context.TODO())

		var m Multiplexer
		m.out = w
		m.ti, err = terminfo.LookupTerminfo("ansi")
		require.NoError(t, err)
		m.rows = rows
		m.cols = cols

		m.Config.Shell = []string{"sh"}

		err = m.RunShell()
		require.NoError(t, err)

		// This is to simulate what happens in the real world,
		// where the command is sent by the human after the shell
		// has changed the echo tcattrs. If we don't do this, the
		// text will be seen echod back by the tty because it will
		// be processed before the shell has disabled automatic-echo.
		time.Sleep(50 * time.Millisecond)

		m.HandleInput(TextEvent("echo 'hello'"))
		m.HandleInput(ControlEvent('\n'))

		m.HandleInput(ControlEvent(0x1))

		time.Sleep(50 * time.Millisecond)

		m.HandleInput(TextEvent("echo 'col2'"))
		m.HandleInput(ControlEvent('\n'))

		time.Sleep(time.Second)

		assertRange(t, scr, 0, 0, "sh-3.2$ echo 'hello'")
		assertRange(t, scr, 1, 0, "hello")
		assertRange(t, scr, 2, 0, "sh-3.2$")

		assertRange(t, scr, 0, 40, "│")
		assertRange(t, scr, 1, 40, "│")
		assertRange(t, scr, 2, 40, "│")

		assertRange(t, scr, 0, 41, "sh-3.2$ echo 'col2'")
		assertRange(t, scr, 1, 41, "col2")
		assertRange(t, scr, 2, 41, "sh-3.2$")

		scr.WriteToFile("snap.txt")
	})

	n.Meow()
}
