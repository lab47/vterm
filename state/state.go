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

type CellRune struct {
	Rune  rune
	Width int
}

type Output interface {
	SetCell(pos Pos, val CellRune) error
	AppendCell(pos Pos, r rune) error
	ClearRect(r Rect) error
}

type State struct {
	rows, cols int
	cursor     Pos
	output     Output

	lastPos Pos
}

func NewState(rows, cols int, output Output) (*State, error) {
	screen := &State{
		rows:   rows,
		cols:   cols,
		output: output,
	}

	return screen, nil
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

func (s *State) advancePos() Pos {
	pos := s.cursor
	s.cursor.Col++

	if s.cursor.Col >= s.cols {
		s.cursor.Row++
		s.cursor.Col = 0
	}

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
	switch control {
	case '\b':
		pos := s.cursor

		if pos.Col > 0 {
			pos.Col--
		}

		s.cursor = pos
	case '\t':
		diff := s.cursor.Col % 8
		s.cursor.Col += (8 - diff)

		if s.cursor.Col >= s.cols {
			s.cursor.Col = s.cols - 1
		}
	case '\r':
		s.cursor.Col = 0
	case '\n':
		s.cursor.Row++
	}

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

	parser.ICH: (*State).insertBlackChars,
	parser.ED:  (*State).eraseDisplay,
}

func (s *State) handleCSI(ev *parser.CSIEvent) error {
	f, ok := csiHandlers[ev.CSICommand()]
	if !ok {
		return fmt.Errorf("unhandled CSI command: %x", ev.Command)
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

	s.cursor = pos

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

	s.cursor = pos

	return nil
}

func (s *State) cursorMoveRow(ev *parser.CSIEvent) error {
	pos := s.cursor

	if len(ev.Args) > 0 && ev.Args[0] > 0 {
		pos.Row = ev.Args[0] - 1
	}

	if pos.Row < 0 {
		pos.Row = 0
	} else if pos.Row >= s.rows {
		pos.Row = s.rows - 1
	}

	s.cursor = pos

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

	s.cursor = pos
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

	s.cursor = pos
	return nil
}

func (s *State) cursorTabForward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	diff := pos.Col % 8
	pos.Col += (8 - diff)

	pos.Col += ((inc - 1) * 8)

	if pos.Col >= s.cols {
		pos.Col = s.cols - 1
	}

	s.cursor = pos
	return nil
}

func (s *State) cursorTabBackward(ev *parser.CSIEvent) error {
	pos := s.cursor

	inc := 1

	if len(ev.Args) > 0 && ev.Args[0] != 0 {
		inc = ev.Args[0]
	}

	diff := pos.Col % 8
	if diff > 0 {
		pos.Col -= diff
		inc--
	}

	pos.Col -= (inc * 8)

	if pos.Col < 0 {
		pos.Col = 0
	}

	s.cursor = pos
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

	s.cursor = pos
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

	s.cursor = pos
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

	s.cursor = pos
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

	s.cursor = pos
	return nil
}

func (s *State) insertBlackChars(ev *parser.CSIEvent) error {
	start := s.cursor

	end := start
	if len(ev.Args) > 0 {
		end.Col += ev.Args[0]
	} else {
		end.Col++
	}

	if end.Col > s.cols {
		end.Col = s.cols
	}

	return s.output.ClearRect(Rect{start, end})
}

func (s *State) eraseDisplay(ev *parser.CSIEvent) error {
	mode := 0

	if len(ev.Args) > 0 {
		mode = ev.Args[0]
	}

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
	}

	return nil
}
