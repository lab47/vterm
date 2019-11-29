package multiplex

import (
	"io"

	"github.com/y0ssar1an/q"
)

type LayoutColumn struct {
	Row   int
	Width int
	Data  []*LayoutRow
}

type LayoutRow struct {
	Column int
	Height int

	// This is an or. A row has a term OR a set of columns
	Data []*LayoutColumn
	Term *Term
}

type Layout struct {
	m *Multiplexer

	Operations Operations

	Rows, Columns int

	Data []*LayoutRow

	focusTerm  *Term
	focusInput io.Writer

	currentRow *LayoutRow
}

func NewLayout(m *Multiplexer, t *Term, rows, cols int) (*Layout, error) {
	var l Layout
	l.m = m
	l.Rows = rows
	l.Columns = cols
	l.Operations.l = &l

	row := &LayoutRow{
		Height: rows,
		Column: 0,
		Term:   t,
	}

	l.Data = append(l.Data, row)

	l.currentRow = row
	l.focusTerm = t

	return &l, nil
}

func (l *Layout) Start() error {
	w, err := l.focusTerm.Start(l.Rows, l.Columns, 0, 0)
	if err != nil {
		return err
	}

	l.focusInput = w

	return nil
}

func (l *Layout) Write(b []byte) (int, error) {
	return l.focusInput.Write(b)
}

func (l *Layout) Draw(w io.Writer) error {
	c1 := l.currentRow.Data[0]
	c2 := l.currentRow.Data[1]

	r1 := c1.Data[0]
	r2 := c2.Data[0]

	if r1 != nil && r2 != nil {
		q.Q("setup")
	} else {
		q.Q("error")
	}

	r1.Term.ResizeMoved(r1.Height, c1.Width, c1.Row, r1.Column)
	r2.Term.ResizeMoved(r2.Height, c2.Width, c2.Row, r2.Column)

	return nil
}
