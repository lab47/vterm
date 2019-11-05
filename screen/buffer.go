package screen

type ScreenCell struct {
	val   rune
	pen   *ScreenPen
	extra []rune
}

func (s *ScreenCell) Value() (rune, []rune) {
	return s.val, s.extra
}

func (s *ScreenCell) Pen() *ScreenPen {
	return s.pen
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
		rows:  rows,
		cols:  cols,
		lines: make([]*line, rows),
	}

	for i := 0; i < rows; i++ {
		buf.lines[i] = &line{
			cells: make([]ScreenCell, cols),
		}
	}

	return buf
}

type line struct {
	cells []ScreenCell

	continuation bool
}

func (l *line) Len() int {
	for i := len(l.cells) - 1; i >= 0; i-- {
		if l.cells[i].val > 0 {
			return i + 1
		}
	}

	return 0
}

func (l *line) resize(sz int) {
	if len(l.cells) >= sz {
		return
	}

	cells := make([]ScreenCell, sz)

	copy(cells, l.cells)

	l.cells = cells
}

type Buffer struct {
	rows, cols int
	lines      []*line
}

func (b *Buffer) getLine(row int) *line {
	l := b.lines[row]
	if l == nil {
		l = &line{
			cells: make([]ScreenCell, b.cols),
		}

		b.lines[row] = l
	}

	return l
}

func (b *Buffer) injectLine(row int, data []ScreenCell) {
	lines := make([]*line, len(b.lines))

	copy(lines, b.lines[1:row])
	lines[row] = &line{cells: data}
	copy(lines[row+1:], b.lines[row:])

	b.lines = lines
}

func (b *Buffer) getCell(row, col int) *ScreenCell {
	line := b.getLine(row)
	return &line.cells[col]
}

func (b *Buffer) setCell(row, col int, r rune) {
	b.lines[row].cells[col].reset(r, nil)
}

func (b *Buffer) moveInRow(row, start, dest, cols int) {
	line := b.getLine(row)

	if start < dest {
		end := start + cols - 1
		for i := dest + cols - 1; i >= dest; i-- {
			line.cells[i] = line.cells[end]
			end--
		}
	} else {
		for i, cell := range line.cells[start : start+cols] {
			line.cells[dest+i] = cell
		}
	}
}

func (b *Buffer) moveBetweenRows(row, rowDest, start, cols int) {
	src := b.getLine(row)
	dst := b.getLine(rowDest)

	for i := start; i < start+cols; i++ {
		dst.cells[i] = src.cells[i]
		src.cells[i].reset(0, nil)
	}
}

func (b *Buffer) eraseInRow(row, start, cols int) {
	line := b.getLine(row)

	for i := start; i < start+cols; i++ {
		line.cells[i].reset(0, nil)
	}
}

/*
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
*/
