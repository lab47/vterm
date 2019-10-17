package screen

import (
	"testing"

	"github.com/evanphx/vterm/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type opSink struct {
	cellOps    map[Pos]CellRune
	appendOps  map[Pos][]rune
	clearRects []Rect
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

	n.It("can request a rectangle is cleared", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		screen.cursor = Pos{1, 3}

		err = screen.HandleEvent(&parser.CSIEvent{Command: '@'})
		require.NoError(t, err)

		require.Equal(t, 1, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 4}}, sink.clearRects[0])

		err = screen.HandleEvent(&parser.CSIEvent{Command: '@', Args: []int{10}})
		require.NoError(t, err)

		require.Equal(t, 2, len(sink.clearRects))

		assert.Equal(t, Rect{Start: Pos{1, 3}, End: Pos{1, 13}}, sink.clearRects[1])
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
	})

	n.Meow()
}
