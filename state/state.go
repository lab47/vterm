package screen

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/evanphx/vterm/parser"
)

type Pos struct {
	Row, Col int
}

type Rect struct {
	Start, End Pos
}

func (s Rect) ScrollUp(dist int) ScrollRect {
	x := ScrollRect{Rect: s}
	x.Vertical = ScrollUp
	x.VerticalDistance = dist
	return x
}

func (s Rect) ScrollDown(dist int) ScrollRect {
	x := ScrollRect{Rect: s}
	x.Vertical = ScrollDown
	x.VerticalDistance = dist
	return x
}

func (s Rect) ScrollLeft(dist int) ScrollRect {
	x := ScrollRect{Rect: s}
	x.Horizontal = ScrollLeft
	x.HorizontalDistance = dist
	return x
}

func (s Rect) ScrollRight(dist int) ScrollRect {
	x := ScrollRect{Rect: s}
	x.Horizontal = ScrollRight
	x.HorizontalDistance = dist
	return x
}

type ScrollDirection int

const (
	ScrollNone  ScrollDirection = iota // don't scroll
	ScrollUp                           // move the content at the top of the rect to the bottom
	ScrollDown                         // move the content at the bottom of the rect to the top
	ScrollRight                        // move the content on the right side of the rect to the left side
	ScrollLeft                         // move the content on the left side of the rect to the right side
)

type ScrollRect struct {
	Rect
	Vertical           ScrollDirection
	VerticalDistance   int
	Horizontal         ScrollDirection
	HorizontalDistance int
}

func (s ScrollRect) Up(dist int) ScrollRect {
	x := s
	x.Vertical = ScrollUp
	x.VerticalDistance = dist
	return x
}

func (s ScrollRect) Down(dist int) ScrollRect {
	x := s
	x.Vertical = ScrollDown
	x.VerticalDistance = dist
	return x
}

func (s ScrollRect) Left(dist int) ScrollRect {
	x := s
	x.Horizontal = ScrollLeft
	x.HorizontalDistance = dist
	return x
}

func (s ScrollRect) Right(dist int) ScrollRect {
	x := s
	x.Horizontal = ScrollRight
	x.HorizontalDistance = dist
	return x
}

type CellRune struct {
	Rune  rune
	Width int
}

type Output interface {
	SetCell(pos Pos, val CellRune) error
	AppendCell(pos Pos, r rune) error
	ClearRect(r Rect) error
	ScrollRect(s ScrollRect) error
	Output(data []byte) error
	SetTermProp(prop string, val interface{}) error
	SetPenProp(prop string, val interface{}) error
}

type modes struct {
	insert          bool
	newline         bool
	cursor          bool
	origin          bool
	autowrap        bool
	leftrightmargin bool
	report_focus    bool
	bracketpaste    bool
	altscreen       bool
}

const (
	MouseNone int = iota
	MouseClick
	MouseDrag
	MouseMove
)

const (
	MouseX10 int = iota
	MouseUTF8
	MouseSGR
	MouseRXVT
)

type State struct {
	rows, cols int
	cursor     Pos
	pen        PenState
	output     Output

	lastPos  Pos
	tabStops []bool

	modes         modes
	mouseProtocol int
	savedCursor   Pos

	scrollregion struct {
		top, bottom int
	}
}

func NewState(rows, cols int, output Output) (*State, error) {
	screen := &State{
		rows:     rows,
		cols:     cols,
		output:   output,
		tabStops: make([]bool, cols),
	}

	err := screen.Reset()
	if err != nil {
		return nil, err
	}

	return screen, nil
}

func (s *State) Reset() error {
	s.modes = modes{autowrap: true}
	for col := 0; col < s.cols; col++ {
		if col%8 == 0 {
			s.tabStops[col] = true
		} else {
			s.tabStops[col] = false
		}
	}

	s.scrollregion.top = 0
	s.scrollregion.bottom = -1

	return nil
}

func (s *State) HandleEvent(gev parser.Event) error {
	switch ev := gev.(type) {
	case *parser.TextEvent:
		return s.writeData(ev.Text)
	case *parser.ControlEvent:
		return s.handleControl(ev.Control)
	case *parser.CSIEvent:
		return s.handleCSI(ev)
	default:
		return fmt.Errorf("unhandled event type: %T", ev)
	}
}

func (s *State) scrollBounds() (int, int) {
	bottom := s.scrollregion.bottom
	if bottom <= -1 {
		bottom = s.rows - 1
	}

	return s.scrollregion.top, bottom
}

func (s *State) setCursor(p Pos) {
	if s.modes.origin {
		switch {
		case p.Row < s.scrollregion.top:
			p.Row = s.scrollregion.top
		case p.Row >= s.scrollregion.bottom:
			p.Row = s.scrollregion.bottom
		}
	} else {
		switch {
		case p.Row < 0:
			p.Row = 0
		case p.Row >= s.rows:
			p.Row = s.rows - 1
		}

	}

	switch {
	case p.Col < 0:
		p.Col = 0
	case p.Col >= s.cols:
		p.Col = s.cols - 1
	}

	s.cursor = p
}

func (s *State) advancePos() Pos {
	pos := s.cursor

	newCur := s.cursor
	newCur.Col++

	if newCur.Col >= s.cols {
		newCur.Row++
		newCur.Col = 0
	}

	s.setCursor(newCur)

	return pos
}

func (s *State) writeData(data []byte) error {
	for len(data) > 0 {
		r, sz := utf8.DecodeRune(data)

		data = data[sz:]

		if unicode.In(r, unicode.Diacritic) {
			err := s.output.AppendCell(s.lastPos, r)
			if err != nil {
				return err
			}

			continue
		}

		pos := s.advancePos()

		s.lastPos = pos

		err := s.output.SetCell(pos, CellRune{r, 1})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *State) handleControl(control byte) error {
	pos := s.cursor

	switch control {
	case '\b':
		if pos.Col > 0 {
			pos.Col--
		}

	case '\t':
		pos.Col++
		for pos.Col < s.cols {
			if s.tabStops[pos.Col] {
				break
			}

			pos.Col++
		}
	case '\r':
		pos.Col = 0
	case '\n':
		pos.Row++
	}

	s.setCursor(pos)
	return nil
}

var csiHandlers = map[parser.CSICommand]func(*State, *parser.CSIEvent) error{
	parser.CUU: (*State).cursorUp,
	parser.VPB: (*State).cursorUp,
	parser.CUD: (*State).cursorDown,
	parser.VPR: (*State).cursorDown,
	parser.CUF: (*State).cursorForward,
	parser.HPR: (*State).cursorForward,
	parser.CUB: (*State).cursorBackward,
	parser.HPB: (*State).cursorBackward,
	parser.CNL: (*State).cursorNextLine,
	parser.CPL: (*State).cursorPrevLine,
	parser.CHA: (*State).cursorMoveCol,
	parser.HPA: (*State).cursorMoveCol,
	parser.CUP: (*State).cursorMove,
	parser.HVP: (*State).cursorMove,
	parser.VPA: (*State).cursorMoveRow,
	parser.CHT: (*State).cursorTabForward,
	parser.CBT: (*State).cursorTabBackward,

	parser.ICH: (*State).insertBlankChars,
	parser.ED:  (*State).eraseDisplay,
	parser.EL:  (*State).eraseLine,
	parser.IL:  (*State).insertLines,
	parser.DL:  (*State).deleteLines,
	parser.DCH: (*State).deleteChars,
	parser.SU:  (*State).scrollUp,
	parser.SD:  (*State).scrollDown,
	parser.ECH: (*State).eraseChars,

	parser.DA:    (*State).emitDeviceAttributes,
	parser.DA_LT: (*State).emitDeviceAttributes2,

	parser.TBC: (*State).clearTabStop,

	parser.SM:   (*State).setMode,
	parser.SM_Q: (*State).setDecMode,

	parser.SGR: (*State).selectGraphics,

	parser.DSR:   (*State).statusReport,
	parser.DSR_Q: (*State).statusReportDec,

	parser.DECSTR: (*State).softReset,

	parser.DECSTBM: (*State).setTopBottomMargin,
}

func (s *State) handleCSI(ev *parser.CSIEvent) error {
	cmd := ev.CSICommand()
	f, ok := csiHandlers[cmd]
	if !ok {
		return fmt.Errorf("unhandled CSI command: (%s) %x", cmd, ev.Command)
	}

	return f(s, ev)
}

func (s *State) cursorMove(ev *parser.CSIEvent) error {
	var pos Pos

	if len(ev.Args) > 0 && ev.Args[0] > 0 {
		pos.Row = ev.Args[0] - 1
	}

	if len(ev.Args) > 1 && ev.Args[1] > 0 {
		pos.Col = ev.Args[1] - 1
	}

	if pos.Row < 0 {
		pos.Row = 0
	} else if pos.Row >= s.rows {
		pos.Row = s.rows - 1
	}

	if pos.Col < 0 {
		pos.Col = 0
	} else if pos.Col >= s.cols {
		pos.Col = s.cols - 1
	}

	if s.modes.origin {
		pos.Row += s.scrollregion.top
	}

	s.setCursor(pos)

	return nil
}

func (s *State) cursorMoveCol(ev *parser.CSIEvent) error {
	pos := s.cursor

	if len(ev.Args) > 0 && ev.Args[0] > 0 {
		pos.Col = ev.Args[0] - 1
	} else {
		pos.Col = 0
	}

	if pos.Col < 0 {
		pos.Col = 0
	} else if pos.Col >= s.cols {
		pos.Col = s.cols - 1
	}

	s.setCursor(pos)

	return nil
}

func (s *State) cursorMoveRow(ev *parser.CSIEvent) error {
	pos := s.cursor

	row := 1

	if len(ev.Args) > 0 {
		row = ev.Args[0]
	}

	if row > 0 {
		pos.Row = row - 1
	}

	if pos.Row < 0 {
		pos.Row = 0
	} else if pos.Row >= s.rows {
		pos.Row = s.rows - 1
	}

	s.setCursor(pos)

	return nil
}

func (s *State) cursorForward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Col += inc

	if pos.Col >= s.cols {
		pos.Col = s.cols - 1
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorBackward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Col -= inc

	if pos.Col < 0 {
		pos.Col = 0
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorTabForward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	for i := 0; i < inc; i++ {
		for pos.Col < s.cols {
			pos.Col++

			if s.tabStops[pos.Col] {
				break
			}
		}
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorTabBackward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	for i := 0; i < inc; i++ {
		for pos.Col > 0 {
			pos.Col--

			if s.tabStops[pos.Col] {
				break
			}
		}
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorUp(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Row -= inc

	if pos.Row < 0 {
		pos.Row = 0
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorDown(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Row += inc

	if pos.Row >= s.rows {
		pos.Row = s.rows - 1
	}

	s.setCursor(pos)
	return nil
}

func (s *State) cursorNextLine(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Row += inc

	if pos.Row >= s.rows {
		pos.Row = s.rows - 1
	}

	pos.Col = 0

	s.setCursor(pos)
	return nil
}

func (s *State) cursorPrevLine(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	pos.Row -= inc

	if pos.Row < 0 {
		pos.Row = 0
	}

	pos.Col = 0

	s.setCursor(pos)
	return nil
}

func (s *State) insertBlankChars(ev *parser.CSIEvent) error {
	start := s.cursor

	end := start

	end.Col = s.cols - 1

	dist := 1
	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	return s.output.ScrollRect(Rect{start, end}.ScrollRight(dist))
}

func (s *State) eraseDisplay(ev *parser.CSIEvent) error {
	mode := 0

	if len(ev.Args) > 0 {
		mode = ev.Args[0]
	}

	// TODO support the ? leader to indicate the DEC selective erase, which
	// only erases characters that were previously defined by DECSCA.

	switch mode {
	case 0: // from cursor to end of display
		start := s.cursor
		end := start

		end.Col = s.cols - 1

		if start.Col > 0 {
			err := s.output.ClearRect(Rect{start, end})
			if err != nil {
				return err
			}
			start.Row++
		}

		start.Col = 0

		end.Row = s.rows - 1

		return s.output.ClearRect(Rect{start, end})
	case 1: // from start to cursor
		start := Pos{0, 0}

		end := s.cursor

		end.Row--
		end.Col = s.cols - 1

		err := s.output.ClearRect(Rect{start, end})
		if err != nil {
			return err
		}

		start = s.cursor
		end = start

		start.Col = 0

		return s.output.ClearRect(Rect{start, end})
	case 2: // the whole display
		start := Pos{0, 0}
		end := Pos{s.rows - 1, s.cols - 1}
		return s.output.ClearRect(Rect{start, end})
	}

	return nil
}

func (s *State) eraseLine(ev *parser.CSIEvent) error {
	mode := 0

	if len(ev.Args) > 0 {
		mode = ev.Args[0]
	}

	// TODO support the ? leader to indicate the DEC selective erase, which
	// only erases characters that were previously defined by DECSCA.

	start := s.cursor
	end := start

	switch mode {
	case 0: // from cursor to end of line
		end.Col = s.cols - 1
	case 1: // from start to cursor
		start.Col = 0
	case 2: // the whole display
		start.Col = 0
		end.Col = s.cols - 1
	default:
		return nil
	}

	return s.output.ClearRect(Rect{start, end})
}

func (s *State) insertLines(ev *parser.CSIEvent) error {
	var dist int

	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	if dist == 0 {
		dist = 1
	}

	start := s.cursor
	start.Col = 0

	_, bottom := s.scrollBounds()

	end := Pos{bottom, s.cols - 1}

	return s.output.ScrollRect(Rect{start, end}.ScrollDown(dist))
}

func (s *State) deleteLines(ev *parser.CSIEvent) error {
	var dist int

	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	if dist == 0 {
		dist = 1
	}

	start := s.cursor
	start.Col = 0

	_, bottom := s.scrollBounds()

	end := Pos{bottom, s.cols - 1}

	return s.output.ScrollRect(Rect{start, end}.ScrollUp(dist))
}

func (s *State) deleteChars(ev *parser.CSIEvent) error {
	start := s.cursor

	end := start

	end.Col = s.cols - 1

	dist := 1
	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	return s.output.ScrollRect(Rect{start, end}.ScrollLeft(dist))
}

func (s *State) scrollUp(ev *parser.CSIEvent) error {
	top, bottom := s.scrollBounds()

	start := Pos{top, 0}
	end := Pos{bottom, s.cols - 1}

	var dist int
	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	if dist == 0 {
		dist = 1
	}

	return s.output.ScrollRect(Rect{start, end}.ScrollUp(dist))
}

func (s *State) scrollDown(ev *parser.CSIEvent) error {
	top, bottom := s.scrollBounds()

	start := Pos{top, 0}
	end := Pos{bottom, s.cols - 1}

	var dist int
	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	if dist == 0 {
		dist = 1
	}

	return s.output.ScrollRect(Rect{start, end}.ScrollDown(dist))
}

func (s *State) eraseChars(ev *parser.CSIEvent) error {
	start := s.cursor

	var dist int

	if len(ev.Args) > 0 {
		dist = ev.Args[0]
	}

	if dist == 0 {
		dist = 1
	}

	end := start
	end.Col += (dist - 1)

	return s.output.ClearRect(Rect{start, end})
}

func (s *State) emitDeviceAttributes(ev *parser.CSIEvent) error {
	return s.output.Output([]byte("\x9b?1;2c"))
}

func (s *State) emitDeviceAttributes2(ev *parser.CSIEvent) error {
	return s.output.Output([]byte("\x9b>0;100;0c"))
}

func (s *State) clearTabStop(ev *parser.CSIEvent) error {
	var mode int

	if len(ev.Args) > 0 {
		mode = ev.Args[0]
	}

	switch mode {
	case 0:
		s.tabStops[s.cursor.Col] = false
	case 3:
		s.tabStops = make([]bool, s.cols)
	}

	return nil
}

func (s *State) setMode(ev *parser.CSIEvent) error {
	if len(ev.Args) == 0 {
		return nil
	}

	mode := ev.Args[0]

	switch mode {
	case 4:
		s.modes.insert = true
	case 20:
		s.modes.newline = true
	}

	return nil
}

func (s *State) setDecMode(ev *parser.CSIEvent) error {
	if len(ev.Args) == 0 {
		return nil
	}

	mode := ev.Args[0]

	switch mode {
	case 1:
		s.modes.cursor = true
	case 5:
		return s.output.SetTermProp("reverse", true)
	case 6:
		s.modes.origin = true
		s.cursor = Pos{0, 0}
	case 7:
		s.modes.autowrap = true
	case 12:
		return s.output.SetTermProp("blink", true)
	case 25:
		return s.output.SetTermProp("visible", true)
	case 69:
		s.modes.leftrightmargin = true
	case 1000:
		return s.output.SetTermProp("mouse", MouseClick)
	case 1002:
		return s.output.SetTermProp("mouse", MouseDrag)
	case 1003:
		return s.output.SetTermProp("mouse", MouseMove)
	case 1004:
		s.modes.report_focus = true
	case 1005:
		s.mouseProtocol = MouseUTF8
	case 1006:
		s.mouseProtocol = MouseSGR
	case 1015:
		s.mouseProtocol = MouseRXVT
	case 1047:
		return s.output.SetTermProp("altscreen", true)
	case 1048:
		s.savedCursor = s.cursor
	case 1049:
		s.savedCursor = s.cursor
		return s.output.SetTermProp("altscreen", true)
	case 2004:
		s.modes.bracketpaste = true
	}

	return nil
}

func (s *State) statusReport(ev *parser.CSIEvent) error {
	var which int

	if len(ev.Args) > 0 {
		which = ev.Args[0]
	}

	switch which {
	case 5:
		return s.output.Output([]byte("\x9b0n"))
	case 6:
		return s.output.Output([]byte(fmt.Sprintf("\x9b%d;%dR", s.cursor.Row+1, s.cursor.Col+1)))
	}

	return nil
}

func (s *State) statusReportDec(ev *parser.CSIEvent) error {
	var which int

	if len(ev.Args) > 0 {
		which = ev.Args[0]
	}

	switch which {
	case 5:
		return s.output.Output([]byte("\x9b?0n"))
	case 6:
		return s.output.Output([]byte(fmt.Sprintf("\x9b?%d;%dR", s.cursor.Row+1, s.cursor.Col+1)))
	}

	return nil
}

func (s *State) softReset(ev *parser.CSIEvent) error {
	return s.Reset()
}

func (s *State) setTopBottomMargin(ev *parser.CSIEvent) error {
	var (
		top    = 1
		bottom = -1
	)

	switch len(ev.Args) {
	case 2:
		bottom = ev.Args[1] - 1
		fallthrough
	case 1:
		top = ev.Args[0]
	}

	if top < 1 {
		top = 1
	}

	if top > s.rows {
		top = s.rows
	}

	if bottom > s.rows {
		bottom = s.rows
	}

	s.scrollregion.top = top - 1
	s.scrollregion.bottom = bottom

	return nil
}
