package multiplex

import (
	"io"

	"github.com/evanphx/vterm/state"
)

type LayoutElement struct {
	Position state.Rect
}

type LayoutColumn struct {
	LayoutElement
	Data []*LayoutRow
}

type LayoutRow struct {
	LayoutElement

	// This is an or. A row has a term OR a set of columns
	Data []*LayoutColumn
	Term *Term
}

type Layout struct {
	m *Multiplexer

	Operations Operations

	Rows, Columns int

	top *LayoutRow

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
		Term: t,
	}

	row.Position = state.Rect{
		Start: state.Pos{Row: 0, Col: 0},
		End:   state.Pos{Row: rows - 1, Col: cols - 1},
	}

	l.top = row
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
	return l.drawRow(l.top)

	/*
		c1 := l.currentRow.Data[0]
		c2 := l.currentRow.Data[1]

		r1 := c1.Data[0]
		r2 := c2.Data[0]

		l.m.DrawVerticalLine(state.Pos{Row: 0, Col: r2.Column - 1}, l.Rows)

		r1.Term.ResizeMoved(r1.Height, c1.Width, c1.Row, r1.Column)
		r2.Term.ResizeMoved(r2.Height, c2.Width, c2.Row, r2.Column)

		return nil
	*/
}

func (l *Layout) drawRow(r *LayoutRow) error {
	pos := r.Position.Start

	if pos.Row > 0 {
		pos.Row--
		l.m.layout.m.DrawHorizLine(pos, r.Position.Width())
	}

	if r.Term != nil {
		r.Term.ResizeMoved(r.Position.Height(), r.Position.Width(), r.Position.Start.Row, r.Position.Start.Col)
		return nil
	}

	for _, col := range r.Data {
		err := l.drawColumn(col)
		if err != nil {
			return err
		}
	}

	/*
		c1 := l.currentRow.Data[0]
		c2 := l.currentRow.Data[1]

		r1 := c1.Data[0]
		r2 := c2.Data[0]

		r1.Term.ResizeMoved(r1.Height, c1.Width, c1.Row, r1.Column)
		r2.Term.ResizeMoved(r2.Height, c2.Width, c2.Row, r2.Column)
	*/

	return nil
}

func (l *Layout) drawColumn(c *LayoutColumn) error {
	pos := c.Position.Start

	if pos.Col > 0 {
		pos.Col--
		l.m.DrawVerticalLine(pos, c.Position.Height())
	}

	for _, row := range c.Data {
		err := l.drawRow(row)
		if err != nil {
			return err
		}
	}

	return nil
}
