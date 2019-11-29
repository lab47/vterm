package multiplex

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/pkg/terminfo"
	"github.com/evanphx/vterm/screen"
	"github.com/evanphx/vterm/state"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
	"github.com/y0ssar1an/q"

	_ "github.com/evanphx/vterm/pkg/terminfo/a/ansi"
)

type integrationOutput struct {
	screen *screen.Screen
}

func (i *integrationOutput) DamageDone(r state.Rect) error {
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

	n.It("can setup a split", func(t *testing.T) {
		var i integrationOutput
		scr, err := screen.NewScreen(rows, cols, &i)
		require.NoError(t, err)

		i.screen = scr

		st, err := state.NewState(rows, cols, scr)
		require.NoError(t, err)

		st.Debug = true

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

		q.Q("start")

		err = m.RunShell()
		require.NoError(t, err)

		time.Sleep(time.Second)

		m.HandleInput(TextEvent("echo 'hello'"))
		m.HandleInput(ControlEvent('\n'))

		time.Sleep(time.Second)

		q.Q("split")

		m.HandleInput(ControlEvent(0x1))

		/*
			time.Sleep(10 * time.Second)

			m.HandleInput(TextEvent("echo 'col2'"))
			m.HandleInput(ControlEvent('\n'))

			time.Sleep(5 * time.Second)

		*/

		var buf bytes.Buffer

		for i := 0; i < 8; i++ {
			cell, err := scr.GetCell(1, i)
			require.NoError(t, err)

			r, _ := cell.Value()
			buf.WriteRune(r)
		}

		t.Logf("output: %s", buf.String())

		buf.Reset()

		for i := 40; i < 48; i++ {
			cell, err := scr.GetCell(1, i)
			require.NoError(t, err)

			r, _ := cell.Value()
			buf.WriteRune(r)
		}

		t.Logf("output2: %s", buf.String())

		scr.WriteToFile("snap.txt")
	})

	n.Meow()
}
