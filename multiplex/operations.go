package multiplex

import (
	"os"
	"os/exec"
)

type Operations struct {
	l *Layout
}

func (o *Operations) spawnTerm() (*Term, error) {
	shell := o.l.m.Config.Shell
	cmd := exec.Command(shell[0], shell[1:]...)
	cmd.Env = append(os.Environ(), o.l.m.Config.Env...)

	return NewTerm(o.l.m, cmd)
}

func (o *Operations) Split() error {
	cl := o.l.currentRow
	term := cl.Term

	width := o.l.Columns / 2

	row1 := &LayoutRow{
		Column: 0,
		Height: cl.Height,
		Term:   cl.Term,
	}

	term, err := o.spawnTerm()
	if err != nil {
		return err
	}

	row2 := &LayoutRow{
		Column: width,
		Height: cl.Height,
		Term:   term,
	}

	col1 := &LayoutColumn{
		Width: width,
	}

	col1.Data = append(col1.Data, row1)

	col2 := &LayoutColumn{
		Width: width,
	}

	col2.Data = append(col2.Data, row2)

	cl.Data = append(cl.Data, col1, col2)
	cl.Term = nil

	w, err := term.Start(o.l.Rows, width, 0, width)
	if err != nil {
		return err
	}

	o.l.focusTerm = term
	o.l.focusInput = w

	return o.l.m.Redraw()
}
