package screen

import (
	"testing"

	"github.com/evanphx/vterm/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type prop struct {
	prop string
	val  interface{}
}

type opSink struct {
	cellOps    map[Pos]CellRune
	appendOps  map[Pos][]rune
	clearRects []Rect
	scrollRect []ScrollRect
	outputs    [][]byte
	termProps  []prop
	penProps   []prop
}

func (o *opSink) SetCell(pos Pos, val CellRune) error {
	if o.cellOps == nil {
		o.cellOps = make(map[Pos]CellRune)
	}

	o.cellOps[pos] = val
	return nil
}

func (o *opSink) AppendCell(pos Pos, r rune) error {
	if o.appendOps == nil {
		o.appendOps = make(map[Pos][]rune)
	}

	o.appendOps[pos] = append(o.appendOps[pos], r)

	return nil
}

func (o *opSink) ClearRect(rect Rect) error {
	o.clearRects = append(o.clearRects, rect)
	return nil
}

func (o *opSink) ScrollRect(rect ScrollRect) error {
	o.scrollRect = append(o.scrollRect, rect)
	return nil
}

func (o *opSink) Output(b []byte) error {
	cp := make([]byte, len(b))
	copy(cp, b)

	o.outputs = append(o.outputs, cp)

	return nil
}

func (o *opSink) SetTermProp(p string, val interface{}) error {
	o.termProps = append(o.termProps, prop{p, val})
	return nil
}

func (o *opSink) SetPenProp(p string, val interface{}) error {
	o.penProps = append(o.penProps, prop{p, val})
	return nil
}

func TestState(t *testing.T) {
	n := neko.Modern(t)

	n.It("generates SetCell for normal output", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(80, 20, &sink)
		require.NoError(t, err)

		err = screen.HandleEvent(&parser.TextEvent{
			Text: []byte("ABC"),
		})

		require.NoError(t, err)

		assert.Equal(t, CellRune{rune('A'), 1}, sink.cellOps[Pos{0, 0}])
		assert.Equal(t, CellRune{rune('B'), 1}, sink.cellOps[Pos{0, 1}])
		assert.Equal(t, CellRune{rune('C'), 1}, sink.cellOps[Pos{0, 2}])
	})

	n.It("outputs runes for utf-8 input", func(t *testing.T) {
		tests := []struct {
			input  []byte
			output []CellRune
		}{
			{
				[]byte("\xc3\x81\xc3\xa9"),
				[]CellRune{{0xc1, 1}, {0xe9, 1}},
			},
			{
				[]byte("\xef\xbc\x90 "),
				[]CellRune{{0xff10, 1}, {0x20, 1}}, // we don't support full width yet
			},
			{
				[]byte("\xF0\x9F\x98\x80 "),
				[]CellRune{{0x1f600, 1}, {0x20, 1}}, // we don't support full width yet
			},
		}

		for _, i := range tests {
			var sink opSink

			screen, err := NewState(80, 20, &sink)
			require.NoError(t, err)

			err = screen.HandleEvent(&parser.TextEvent{
				Text: i.input,
			})

			require.NoError(t, err)

			col := 0

			for _, r := range i.output {
				assert.Equal(t, r, sink.cellOps[Pos{0, col}])
				col += r.Width
			}
		}
	})

	n.It("sends an update when detecting a combining char", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(20, 80, &sink)
		require.NoError(t, err)

		err = screen.HandleEvent(&parser.TextEvent{
			Text: []byte("e\xcc\x81Z"),
		})

		require.NoError(t, err)

		assert.Equal(t, rune(0x65), sink.cellOps[Pos{0, 0}].Rune)
		assert.Equal(t, rune(0x301), sink.appendOps[Pos{0, 0}][0])
		assert.Equal(t, rune(0x5a), sink.cellOps[Pos{0, 1}].Rune)

		assert.Equal(t, Pos{0, 2}, screen.cursor)
	})

	n.It("moves the cursor on a control characters", func(t *testing.T) {
		tests := []struct {
			control byte
			pos     Pos
		}{
			{'\b', Pos{0, 2}},
			{'\t', Pos{0, 8}},
			{'\r', Pos{0, 0}},
			{'\n', Pos{1, 0}},
			{'\b', Pos{1, 0}},
			{'\b', Pos{1, 0}},
			{'\t', Pos{1, 8}},
			{'\t', Pos{1, 8 * 2}},
			{'\t', Pos{1, 8 * 3}},
			{'\t', Pos{1, 8 * 4}},
			{'\t', Pos{1, 8 * 5}},
			{'\t', Pos{1, 8 * 6}},
			{'\t', Pos{1, 8 * 7}},
			{'\t', Pos{1, 8 * 8}},
			{'\t', Pos{1, 8 * 9}},
			{'\t', Pos{1, 79}},
			{'\t', Pos{1, 79}},
		}

		var sink opSink

		screen, err := NewState(20, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{0, 3}

		for _, test := range tests {
			err = screen.HandleEvent(&parser.ControlEvent{
				Control: test.control,
			})

			require.NoError(t, err)

			assert.Equal(t, test.pos, screen.cursor)
		}
	})

	n.It("moves the cursor on a responding to CSI events", func(t *testing.T) {
		tests := []struct {
			event *parser.CSIEvent
			pos   Pos
		}{
			{&parser.CSIEvent{Command: 'H', Args: []int{4, 2}}, Pos{3, 1}},
			{&parser.CSIEvent{Command: 'C'}, Pos{3, 2}},
			{&parser.CSIEvent{Command: 'C', Args: []int{3}}, Pos{3, 5}},
			{&parser.CSIEvent{Command: 'C', Args: []int{0}}, Pos{3, 6}},
			{&parser.CSIEvent{Command: 'C', Args: []int{1}}, Pos{3, 7}},
			{&parser.CSIEvent{Command: 'B'}, Pos{4, 7}},
			{&parser.CSIEvent{Command: 'B', Args: []int{3}}, Pos{7, 7}},
			{&parser.CSIEvent{Command: 'B', Args: []int{0}}, Pos{8, 7}},
			{&parser.CSIEvent{Command: 'B', Args: []int{1}}, Pos{9, 7}},
			{&parser.CSIEvent{Command: 'A'}, Pos{8, 7}},
			{&parser.CSIEvent{Command: 'A', Args: []int{3}}, Pos{5, 7}},
			{&parser.CSIEvent{Command: 'A', Args: []int{0}}, Pos{4, 7}},
			{&parser.CSIEvent{Command: 'A', Args: []int{1}}, Pos{3, 7}},
			{&parser.CSIEvent{Command: 'D'}, Pos{3, 6}},
			{&parser.CSIEvent{Command: 'D', Args: []int{3}}, Pos{3, 3}},
			{&parser.CSIEvent{Command: 'D', Args: []int{0}}, Pos{3, 2}},
			{&parser.CSIEvent{Command: 'D', Args: []int{1}}, Pos{3, 1}},
			{&parser.CSIEvent{Command: 'E'}, Pos{4, 0}},
			{&parser.CSIEvent{Command: 'E', Args: []int{3}}, Pos{7, 0}},
			{&parser.CSIEvent{Command: 'E', Args: []int{0}}, Pos{8, 0}},
			{&parser.CSIEvent{Command: 'E', Args: []int{1}}, Pos{9, 0}},
			{&parser.CSIEvent{Command: 'H', Args: []int{10, 2}}, Pos{9, 1}},
			{&parser.CSIEvent{Command: 'F'}, Pos{8, 0}},
			{&parser.CSIEvent{Command: 'F', Args: []int{3}}, Pos{5, 0}},
			{&parser.CSIEvent{Command: 'F', Args: []int{0}}, Pos{4, 0}},
			{&parser.CSIEvent{Command: 'F', Args: []int{1}}, Pos{3, 0}},
			{&parser.CSIEvent{Command: 'H', Args: []int{10, 2}}, Pos{9, 1}},
			{&parser.CSIEvent{Command: 'G'}, Pos{9, 0}},
			{&parser.CSIEvent{Command: 'G', Args: []int{3}}, Pos{9, 2}},
			{&parser.CSIEvent{Command: 'H', Args: []int{8}}, Pos{7, 0}},
			{&parser.CSIEvent{Command: 'H'}, Pos{0, 0}},
			{&parser.CSIEvent{Command: 'A'}, Pos{0, 0}},
			{&parser.CSIEvent{Command: 'D'}, Pos{0, 0}},
			{&parser.CSIEvent{Command: 'H', Args: []int{25, 80}}, Pos{24, 79}},
			{&parser.CSIEvent{Command: 'B'}, Pos{24, 79}},
			{&parser.CSIEvent{Command: 'C'}, Pos{24, 79}},
			{&parser.CSIEvent{Command: 'E'}, Pos{24, 0}},
			{&parser.CSIEvent{Command: 'H'}, Pos{0, 0}},
			{&parser.CSIEvent{Command: 'F'}, Pos{0, 0}},
			{&parser.CSIEvent{Command: 'G', Args: []int{999}}, Pos{0, 79}},
			{&parser.CSIEvent{Command: 'H', Args: []int{99, 99}}, Pos{24, 79}},
			{&parser.CSIEvent{Command: 'd', Args: []int{5}}, Pos{4, 79}},
			{&parser.CSIEvent{Command: 'H', Args: []int{1, 41}}, Pos{0, 40}},
			{&parser.CSIEvent{Command: 'I'}, Pos{0, 48}},
			{&parser.CSIEvent{Command: 'I', Args: []int{2}}, Pos{0, 64}},
			{&parser.CSIEvent{Command: 'Z'}, Pos{0, 56}},
			{&parser.CSIEvent{Command: 'Z', Args: []int{2}}, Pos{0, 40}},
		}

		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{0, 3}

		for i, test := range tests {
			err = screen.HandleEvent(test.event)
			require.NoError(t, err)

			require.Equal(t, test.pos, screen.cursor, "%dth : On event: %#v", i, test.event)
		}
	})

	n.It("can scroll a line to insert characters", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: '@'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 79}}, sr.Rect)

		assert.Equal(t, ScrollRight, sr.Horizontal)
		assert.Equal(t, 1, sr.HorizontalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: '@', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, ScrollRight, sr.Horizontal)
		assert.Equal(t, 10, sr.HorizontalDistance)
	})

	n.It("can clear the display", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'J'})
		require.NoError(t, err)

		require.Equal(t, 2, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 79}}, sink.clearRects[0])
		assert.Equal(t, Rect{Start: Pos{2, 0}, End: Pos{24, 79}}, sink.clearRects[1])

		sink.clearRects = nil
		screen.cursor = Pos{1, 0}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'J'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{24, 79}}, sink.clearRects[0])

		sink.clearRects = nil
		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'J', Args: []int{1}})
		require.NoError(t, err)

		require.Equal(t, 2, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{0, 79}}, sink.clearRects[0])
		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{1, 3}}, sink.clearRects[1])

		sink.clearRects = nil
		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'J', Args: []int{2}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{24, 79}}, sink.clearRects[0])
	})

	n.It("can erase lines", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'K'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 79}}, sink.clearRects[0])

		sink.clearRects = nil
		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'K', Args: []int{1}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{1, 3}}, sink.clearRects[0])

		sink.clearRects = nil
		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'K', Args: []int{2}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{1, 79}}, sink.clearRects[0])
	})

	n.It("can insert lines by scrolling a region", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'L'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollDown, sr.Vertical)
		assert.Equal(t, 1, sr.VerticalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'L', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollDown, sr.Vertical)
		assert.Equal(t, 10, sr.VerticalDistance)
	})

	n.It("can delete lines by scrolling a region", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'M'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollUp, sr.Vertical)
		assert.Equal(t, 1, sr.VerticalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'M', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollUp, sr.Vertical)
		assert.Equal(t, 10, sr.VerticalDistance)
	})

	n.It("can scroll a line to delete characters", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'P'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 79}}, sr.Rect)

		assert.Equal(t, ScrollLeft, sr.Horizontal)
		assert.Equal(t, 1, sr.HorizontalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'P', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, ScrollLeft, sr.Horizontal)
		assert.Equal(t, 10, sr.HorizontalDistance)
	})

	n.It("can scroll everything upward", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'S'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollUp, sr.Vertical)
		assert.Equal(t, 1, sr.VerticalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'S', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollUp, sr.Vertical)
		assert.Equal(t, 10, sr.VerticalDistance)
	})

	n.It("can scroll everything downward", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'T'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr := sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollDown, sr.Vertical)
		assert.Equal(t, 1, sr.VerticalDistance)

		sink.scrollRect = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'T', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.scrollRect))

		sr = sink.scrollRect[0]

		assert.Equal(t, Rect{Start: Pos{0, 0}, End: Pos{24, 79}}, sr.Rect)

		assert.Equal(t, ScrollDown, sr.Vertical)
		assert.Equal(t, 10, sr.VerticalDistance)
	})

	n.It("can erase characters", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'X'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 3}}, sink.clearRects[0])

		sink.clearRects = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'X', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 12}}, sink.clearRects[0])
	})

	n.It("can emit a sequence for device attributes", func(t *testing.T) {

		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'c'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.outputs))

		assert.Equal(t, []byte("\x9b?1;2c"), sink.outputs[0])
	})

	n.It("can emit a sequence for device attributes, dec style", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'c', Leader: []byte{'>'}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.outputs))

		assert.Equal(t, []byte("\x9b>0;100;0c"), sink.outputs[0])
	})

	n.It("can position the cursor to an absolute row", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'd'})
		require.NoError(t, err)

		assert.Equal(t, Pos{0, 3}, screen.cursor)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'd', Args: []int{8}})
		require.NoError(t, err)

		assert.Equal(t, Pos{7, 3}, screen.cursor)
	})

	n.It("can clear tabstops", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 8}

		assert.True(t, screen.tabStops[8])

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'g'})
		require.NoError(t, err)

		assert.False(t, screen.tabStops[8])

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'g', Args: []int{3}})
		require.NoError(t, err)

		for _, b := range screen.tabStops {
			require.False(t, b)
		}
	})

	n.It("can activate modes", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 8}

		assert.True(t, screen.tabStops[8])

		assert.False(t, screen.modes.insert)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Args: []int{4}})
		require.NoError(t, err)

		assert.True(t, screen.modes.insert)

		assert.False(t, screen.modes.newline)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Args: []int{20}})
		require.NoError(t, err)

		assert.True(t, screen.modes.newline)
	})

	n.It("can activate dec modes", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 8}

		assert.True(t, screen.tabStops[8])

		assert.False(t, screen.modes.cursor)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1}})
		require.NoError(t, err)

		assert.True(t, screen.modes.cursor)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{5}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "reverse", sink.termProps[0].prop)
		assert.Equal(t, true, sink.termProps[0].val)

		sink.termProps = nil

		screen.cursor = Pos{4, 10}

		assert.False(t, screen.modes.origin)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{6}})
		require.NoError(t, err)

		assert.True(t, screen.modes.origin)

		assert.Equal(t, Pos{0, 0}, screen.cursor)

		assert.False(t, screen.modes.autowrap)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{7}})
		require.NoError(t, err)

		assert.True(t, screen.modes.autowrap)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{12}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "blink", sink.termProps[0].prop)
		assert.Equal(t, true, sink.termProps[0].val)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{25}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "visible", sink.termProps[0].prop)
		assert.Equal(t, true, sink.termProps[0].val)

		assert.False(t, screen.modes.leftrightmargin)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{69}})
		require.NoError(t, err)

		assert.True(t, screen.modes.leftrightmargin)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1000}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "mouse", sink.termProps[0].prop)
		assert.Equal(t, MouseClick, sink.termProps[0].val)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1002}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "mouse", sink.termProps[0].prop)
		assert.Equal(t, MouseDrag, sink.termProps[0].val)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1003}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "mouse", sink.termProps[0].prop)
		assert.Equal(t, MouseMove, sink.termProps[0].val)

		assert.False(t, screen.modes.report_focus)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1004}})
		require.NoError(t, err)

		assert.True(t, screen.modes.report_focus)

		assert.Equal(t, MouseX10, screen.mouseProtocol)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1005}})
		require.NoError(t, err)

		assert.Equal(t, MouseUTF8, screen.mouseProtocol)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1006}})
		require.NoError(t, err)

		assert.Equal(t, MouseSGR, screen.mouseProtocol)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1015}})
		require.NoError(t, err)

		assert.Equal(t, MouseRXVT, screen.mouseProtocol)

		sink.termProps = nil

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1047}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "altscreen", sink.termProps[0].prop)
		assert.Equal(t, true, sink.termProps[0].val)

		screen.cursor = Pos{12, 33}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1048}})
		require.NoError(t, err)

		assert.Equal(t, Pos{12, 33}, screen.savedCursor)

		sink.termProps = nil

		screen.cursor = Pos{13, 32}

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{1049}})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.termProps))

		assert.Equal(t, "altscreen", sink.termProps[0].prop)
		assert.Equal(t, true, sink.termProps[0].val)

		assert.Equal(t, Pos{13, 32}, screen.savedCursor)

		assert.False(t, screen.modes.bracketpaste)

		err = screen.HandleEvent(&parser.CSIEvent{Command: 'h', Leader: []byte{'?'}, Args: []int{2004}})
		require.NoError(t, err)

		assert.True(t, screen.modes.bracketpaste)
	})

	n.Meow()
}
