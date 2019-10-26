package screen

type ScreenPen struct {
}

type ScreenCell struct {
	val   rune
	pen   *ScreenPen
	extra []rune
}

func (s *ScreenCell) reset(r rune, pen *ScreenPen) error {
	s.val = r
	s.pen = pen
	s.extra = nil
	return nil
}

func (s *ScreenCell) resetTo(x *ScreenCell) {
	s.val = x.val
	s.pen = x.pen
	s.extra = nil

	for _, a := range x.extra {
		s.extra = append(s.extra, a)
	}
}

func (s *ScreenCell) addExtra(r rune) error {
	s.extra = append(s.extra, r)
	return nil
}

func NewBuffer(rows, cols int) *Buffer {
	buf := &Buffer{
		cells: make([]ScreenCell, rows*cols),
	}

	return buf
}

type Buffer struct {
	cells []ScreenCell
}

func (b *Buffer) getCell(idx int) *ScreenCell {
	return &b.cells[idx]
}

func (b *Buffer) move(start, dest, cols int) {
	if start < dest {
		end := start + cols - 1
		for i := dest + cols - 1; i >= dest; i-- {
			b.cells[i] = b.cells[end]
			end--
		}
	} else {
		for i, cell := range b.cells[start : start+cols] {
			b.cells[dest+i] = cell
		}
	}
}

func (b *Buffer) erase(start, cols int) {
	for i := start; i < start+cols; i++ {
		b.cells[i].reset(0, nil)
	}
}
