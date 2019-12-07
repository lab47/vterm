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

	left, right := cl.Position.SplitEvenColumns()

	right.Start.Col++

	row1 := &LayoutRow{
		Term: cl.Term,
	}

	row1.Position = left

	term, err := o.spawnTerm()
	if err != nil {
		return err
	}

	row2 := &LayoutRow{
		Term: term,
	}

	row2.Position = right

	col1 := &LayoutColumn{}

	col1.Position = row1.Position

	col1.Data = append(col1.Data, row1)

	col2 := &LayoutColumn{}

	col2.Position = row2.Position

	col2.Data = append(col2.Data, row2)

	cl.Data = append(cl.Data, col1, col2)
	cl.Term = nil

	w, err := term.Start(right.Height(), right.Width(), right.Start.Row, right.Start.Col)
	if err != nil {
		return err
	}

	o.l.focusTerm = term
	o.l.focusInput = w

	o.l.currentRow = row2

	return o.l.m.Redraw()
}

func (o *Operations) SplitHoriz() error {
	cl := o.l.currentRow
	term := cl.Term

	top, bottom := cl.Position.SplitEvenRows()

	bottom.Start.Row++

	row1 := &LayoutRow{
		Term: cl.Term,
	}

	row1.Position = top

	term, err := o.spawnTerm()
	if err != nil {
		return err
	}

	row2 := &LayoutRow{
		Term: term,
	}

	row2.Position = bottom

	col1 := &LayoutColumn{}

	col1.Position = row1.Position

	col1.Data = append(col1.Data, row1)

	col2 := &LayoutColumn{}

	col2.Position = row2.Position

	col2.Data = append(col2.Data, row2)

	cl.Data = append(cl.Data, col1, col2)
	cl.Term = nil

	w, err := term.Start(bottom.Height(), bottom.Width(), bottom.Start.Row, bottom.Start.Col)
	if err != nil {
		return err
	}

	o.l.focusTerm = term
	o.l.focusInput = w

	o.l.currentRow = row2

	return o.l.m.Redraw()
}
