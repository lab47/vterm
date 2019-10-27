package screen

import (
	"github.com/evanphx/vterm/state"
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

	idx := (s.cols * row) + col

	return s.buffer.getCell(idx)
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
		start := (s.cols * row) + r.Start.Col
		dest := (s.cols * row) + (r.Start.Col + dist)

		s.buffer.move(start, dest, dist)
		s.buffer.erase(start, dist)
	}

	return nil
}

func (s *Screen) slideRectLeft(r state.Rect, dist int) error {
	for row := r.Start.Row; row <= r.End.Row; row++ {
		start := (s.cols * row) + r.Start.Col
		dest := (s.cols * row) + (r.Start.Col - dist)

		s.buffer.move(start, dest, dist)
		s.buffer.erase(start, dist)
	}

	return nil
}

func (s *Screen) slideRectDown(r state.Rect, dist int) error {
	cols := r.End.Col - r.Start.Col + 1

	for row := r.End.Row; row >= r.Start.Row; row-- {
		start := (s.cols * row) + r.Start.Col
		dest := (s.cols * (row + dist)) + r.Start.Col

		s.buffer.move(start, dest, cols)
	}

	for row := r.Start.Row; row < r.Start.Row+dist; row++ {
		start := (s.cols * row) + r.Start.Col

		s.buffer.erase(start, cols)
	}

	return nil
}

func (s *Screen) slideRectUp(r state.Rect, dist int) error {
	cols := r.End.Col - r.Start.Col + 1

	for row := r.Start.Row; row <= r.End.Row; row++ {
		start := (s.cols * row) + r.Start.Col
		dest := (s.cols * (row - dist)) + r.Start.Col

		s.buffer.move(start, dest, cols)
	}

	for row := r.End.Row; row < r.End.Row+dist; row++ {
		start := (s.cols * row) + r.Start.Col

		s.buffer.erase(start, cols)
	}

	return nil
}

func (s *Screen) eraseRect(r state.Rect) {
	for row := r.Start.Row; row < s.rows && row <= r.End.Row; row++ {
		start := (s.cols * row) + r.Start.Col

		s.buffer.erase(start, r.End.Col-r.Start.Col+1)
	}
}

func (s *Screen) ScrollRect(r state.ScrollRect) error {
	switch r.Direction {
	case state.ScrollRight:
		sr := r.Rect
		sr.End.Col -= r.Distance

		return s.slideRectRight(sr, r.Distance)
	case state.ScrollLeft:
		sr := r.Rect
		sr.Start.Col += r.Distance

		return s.slideRectLeft(sr, r.Distance)
	case state.ScrollDown:
		sr := r.Rect
		sr.End.Row -= r.Distance

		return s.slideRectDown(sr, r.Distance)
	case state.ScrollUp:
		sr := r.Rect
		sr.Start.Row += r.Distance

		return s.slideRectUp(sr, r.Distance)
	default:
		return nil
	}
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
