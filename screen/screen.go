package screen

import (
	"errors"

	"github.com/evanphx/vterm/state"
	"github.com/y0ssar1an/q"
)

type Updates interface {
	DamageDone(r state.Rect) error
	MoveCursor(p state.Pos) error
	SetTermProp(attr state.TermAttr, val interface{}) error
	Output(data []byte) error
	StringEvent(kind string, data []byte) error
}

type Screen struct {
	rows, cols int

	pen *ScreenPen

	buffers []*Buffer
	buffer  *Buffer

	updates Updates
}

var _ state.Output = &Screen{}

func NewScreen(rows, cols int, updates Updates) (*Screen, error) {
	screen := &Screen{
		rows:    rows,
		cols:    cols,
		updates: updates,

		buffer: NewBuffer(rows, cols),
		pen:    &ScreenPen{},
	}

	return screen, nil
}

func (s *Screen) getCell(row, col int) *ScreenCell {
	if row < 0 {
		panic("huh")
	}

	return s.buffer.getCell(row, col)
}

var ErrOutOfBounds = errors.New("position of out bounds")

func (s *Screen) GetCell(row, col int) (*ScreenCell, error) {
	if row < 0 || row >= s.rows || col < 0 || col >= s.cols {
		return nil, ErrOutOfBounds
	}

	return s.getCell(row, col), nil
}

func (s *Screen) damagePos(p state.Pos) error {
	return s.damageRect(state.Rect{Start: p, End: p})
}

func (s *Screen) damageRect(r state.Rect) error {
	return s.updates.DamageDone(r)
}

func (s *Screen) MoveCursor(pos state.Pos) error {
	return s.updates.MoveCursor(pos)
}

func (s *Screen) SetCell(pos state.Pos, val state.CellRune) error {
	cell := s.getCell(pos.Row, pos.Col)
	if cell == nil {
		return nil
	}

	err := cell.reset(val.Rune, s.pen)
	if err != nil {
		return err
	}

	return s.damagePos(pos)

	// todo use width
}

func (s *Screen) AppendCell(pos state.Pos, r rune) error {
	cell := s.getCell(pos.Row, pos.Col)
	if cell == nil {
		return nil
	}

	err := cell.addExtra(r)
	if err != nil {
		return err
	}

	return s.damagePos(pos)
}

func (s *Screen) ClearRect(r state.Rect) error {
	for row := r.Start.Row; row < s.rows && row <= r.End.Row; row++ {
		for col := r.Start.Col; col <= r.End.Col; col++ {
			cell := s.getCell(row, col)
			cell.reset(0, s.pen)
		}
	}

	return s.damageRect(r)
}

func (s *Screen) slideRectRight(r state.Rect, dist int) error {
	for row := r.Start.Row; row <= r.End.Row; row++ {
		start := r.Start.Col
		dest := r.Start.Col + dist

		s.buffer.moveInRow(row, start, dest, dist)
		s.buffer.eraseInRow(row, start, dist)
	}

	return nil
}

func (s *Screen) slideRectLeft(r state.Rect, dist int) error {
	for row := r.Start.Row; row <= r.End.Row; row++ {
		start := r.Start.Col
		dest := r.Start.Col - dist

		s.buffer.moveInRow(row, start, dest, dist)
		s.buffer.eraseInRow(row, start, dist)
	}

	return nil
}

func (s *Screen) slideRectDown(r state.Rect, dist int) error {
	cols := r.End.Col - r.Start.Col + 1

	for row := r.End.Row; row >= r.Start.Row; row-- {
		s.buffer.moveBetweenRows(row, row+dist, r.Start.Col, cols)
	}

	for row := r.Start.Row; row < r.Start.Row+dist; row++ {
		s.buffer.eraseInRow(row, r.Start.Col, cols)
	}

	return nil
}

func (s *Screen) slideRectUp(r state.Rect, dist int) error {
	cols := r.End.Col - r.Start.Col + 1

	for row := r.Start.Row; row <= r.End.Row; row++ {
		s.buffer.moveBetweenRows(row, row-dist, r.Start.Col, cols)
	}

	for row := r.End.Row; row < r.End.Row+dist; row++ {
		s.buffer.eraseInRow(row, r.Start.Col, cols)
	}

	return nil
}

func (s *Screen) ScrollRect(r state.ScrollRect) error {
	q.Q(r, r.Direction.String())
	switch r.Direction {
	case state.ScrollRight:
		sr := r.Rect
		sr.End.Col -= r.Distance

		err := s.slideRectRight(sr, r.Distance)
		if err != nil {
			return err
		}
	case state.ScrollLeft:
		sr := r.Rect
		sr.Start.Col += r.Distance

		err := s.slideRectLeft(sr, r.Distance)
		if err != nil {
			return err
		}
	case state.ScrollDown:
		sr := r.Rect
		sr.End.Row -= r.Distance

		err := s.slideRectDown(sr, r.Distance)
		if err != nil {
			return err
		}
	case state.ScrollUp:
		sr := r.Rect
		sr.Start.Row += r.Distance

		err := s.slideRectUp(sr, r.Distance)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	return s.damageRect(r.Rect)
}

func (s *Screen) Output(data []byte) error {
	return s.updates.Output(data)
}

func (s *Screen) SetTermProp(prop state.TermAttr, val interface{}) error {
	return s.updates.SetTermProp(prop, val)
}

func (s *Screen) SetPenProp(prop state.PenAttr, val interface{}, ps state.PenState) error {
	s.pen = &ScreenPen{PenState: ps}
	return nil
}

func (s *Screen) StringEvent(kind string, data []byte) error {
	return s.updates.StringEvent(kind, data)
}
