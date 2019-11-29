package parser

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type collectEvents struct {
	Events []Event
}

func (c *collectEvents) HandleEvent(ev Event) error {
	c.Events = append(c.Events, ev)
	return nil
}

func TestParser(t *testing.T) {
	n := neko.Modern(t)

	n.It("can parse normal text", func(t *testing.T) {
		input := "hello"
		output := []byte("hello")

		var c collectEvents

		pr, err := NewParser(strings.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		ev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, output, ev.Text)
	})

	n.It("can parse C0 bytes", func(t *testing.T) {
		inputs := []byte{0x3, 0x1f}

		for _, i := range inputs {
			var c collectEvents

			pr, err := NewParser(bytes.NewReader([]byte{i}), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(ControlEvent)
			require.True(t, ok)

			assert.Equal(t, i, byte(ev))
		}
	})

	bs := func(str ...string) [][]byte {
		var out [][]byte
		for _, s := range str {
			out = append(out, []byte(s))
		}
		return out
	}

	n.It("can parse C1 7-bit", func(t *testing.T) {
		inputs := bs("\x1b\x43", "\x1b\x5f")
		ctl := []byte{0x83, 0x9f}

		for idx, b := range inputs {
			var c collectEvents

			pr, err := NewParser(bytes.NewReader(b), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(ControlEvent)
			require.True(t, ok)

			assert.Equal(t, ctl[idx], byte(ev))
		}
	})

	n.It("can parse utf-8", func(t *testing.T) {
		inputs := bs("\xf0\x9f\x98\x80", "\xc2\xa2")

		for _, b := range inputs {
			var c collectEvents

			pr, err := NewParser(bytes.NewReader(b), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(*TextEvent)
			require.True(t, ok)

			assert.Equal(t, b, ev.Text)
		}
	})

	n.It("can parse multiple events", func(t *testing.T) {
		input := []byte("1\n2")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 3, len(c.Events))

		ev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, "1", string(ev.Text))

		cev, ok := c.Events[1].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, byte('\n'), byte(cev))

		ev, ok = c.Events[2].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, "2", string(ev.Text))
	})

	n.It("handles escapes", func(t *testing.T) {
		inputs := bs("\x1b=", "\x1b(X")

		for _, i := range inputs {
			var c collectEvents

			pr, err := NewParser(bytes.NewReader(i), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(*EscapeEvent)
			require.True(t, ok)

			assert.Equal(t, i[1:], ev.Data)
		}
	})

	n.It("handles escape canceling escape", func(t *testing.T) {
		i := bs("\x1b(\x1b)Z")[0]

		var c collectEvents

		pr, err := NewParser(bytes.NewReader(i), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		ev, ok := c.Events[0].(*EscapeEvent)
		require.True(t, ok)

		assert.Equal(t, []byte(")Z"), ev.Data)
	})

	n.It("handles cancel cancaling escape", func(t *testing.T) {
		i := bs("\x1b(\x18AB")[0]

		var c collectEvents

		pr, err := NewParser(bytes.NewReader(i), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		ev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("AB"), ev.Text)
	})

	n.It("handles control in escape", func(t *testing.T) {
		input := []byte("\x1b(\nX")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 2, len(c.Events))

		ev, ok := c.Events[0].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, byte('\n'), byte(ev))

		cev, ok := c.Events[1].(*EscapeEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("(X"), cev.Data)
	})

	csi := func(command byte, args ...int) *CSIEvent {
		if args == nil {
			args = make([]int, 0)
		}

		return &CSIEvent{
			Command:  command,
			Args:     args,
			Leader:   make([]byte, 0),
			Intermed: make([]byte, 0),
		}
	}

	csiL := func(command byte, leader []byte, args ...int) *CSIEvent {
		if args == nil {
			args = make([]int, 0)
		}

		return &CSIEvent{
			Command:  command,
			Leader:   leader,
			Args:     args,
			Intermed: make([]byte, 0),
		}
	}

	csiI := func(command byte, intermed []byte, args ...int) *CSIEvent {
		if args == nil {
			args = make([]int, 0)
		}

		return &CSIEvent{
			Command:  command,
			Args:     args,
			Intermed: intermed,
			Leader:   make([]byte, 0),
		}
	}

	n.It("handles CSI sequences", func(t *testing.T) {
		tests := []struct {
			input string
			event *CSIEvent
		}{
			//!CSI 0 args
			{"\x1b[a", csi(0x61)},
			// !CSI 1 arg
			{"\x1b[9b", csi(0x62, 9)},
			// !CSI 2 args
			{"\x1b[3;4c", csi(0x63, 3, 4)},
			// !CSI 1 arg 1 sub
			// PUSH "\x1b[1:2c"
			// csi 0x63 1+,2

			// !CSI many digits
			{"\x1b[678d", csi(0x64, 678)},
			// !CSI leading zero
			{"\x1b[007e", csi(0x65, 7)},
			// !CSI qmark
			{"\x1b[?2;7f", csiL(0x66, []byte{0x3f}, 2, 7)},
			// !CSI greater
			{"\x1b[>c", csiL(0x63, []byte{0x3e})},
			// !CSI SP
			{"\x1b[12 q", csiI(0x71, []byte{0x20}, 12)},
		}

		for _, test := range tests {
			var c collectEvents

			pr, err := NewParser(strings.NewReader(test.input), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(*CSIEvent)
			require.True(t, ok)

			assert.Equal(t, test.event, ev)
		}
	})

	n.It("handles mixed CSI", func(t *testing.T) {
		input := []byte("A\x1b[8mB")

		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 3, len(c.Events))

		ev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("A"), ev.Text)

		cev, ok := c.Events[1].(*CSIEvent)
		require.True(t, ok)

		assert.Equal(t, byte(0x6d), cev.Command)
		assert.Equal(t, []int{8}, cev.Args)

		ev, ok = c.Events[2].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("B"), ev.Text)
	})

	n.It("handles escape cancels csi, starts escape", func(t *testing.T) {
		input := []byte("\x1b[123\x1b9")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*EscapeEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("9"), cev.Data)
	})

	n.It("handles CAN cancels csi, returns to normal", func(t *testing.T) {
		input := []byte("\x1b[12\x18AB")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("AB"), cev.Text)
	})

	n.It("handles C0 in Escape", func(t *testing.T) {
		input := []byte("\x1b[12\n;3X")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 2, len(c.Events))

		cev, ok := c.Events[0].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, byte(10), byte(cev))

		ev, ok := c.Events[1].(*CSIEvent)
		require.True(t, ok)

		assert.Equal(t, byte(0x58), ev.Command)
		assert.Equal(t, []int{12, 3}, ev.Args)
	})

	n.It("handles OSC sequences", func(t *testing.T) {
		tests := []struct {
			input string
			event *OSCEvent
		}{
			// !OSC BEL
			{"\x1b]1;Hello\x07", &OSCEvent{Command: 1, Data: "Hello"}},

			// !OSC ST (7bit)
			{"\x1b]1;Hello\x1b\\", &OSCEvent{Command: 1, Data: "Hello"}},
		}

		for _, test := range tests {
			var c collectEvents

			pr, err := NewParser(strings.NewReader(test.input), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(*OSCEvent)
			require.True(t, ok)

			assert.Equal(t, test.event, ev)
		}
	})

	n.It("handles escape cancels OSC, starts escape", func(t *testing.T) {
		input := []byte("\x1b]Something\x1b9")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*EscapeEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("9"), cev.Data)
	})

	n.It("handles CAN cancels OSC, returns to normal", func(t *testing.T) {
		input := []byte("\x1b]12\x18AB")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("AB"), cev.Text)
	})

	n.It("handles C0 in OSC", func(t *testing.T) {
		input := []byte("\x1b]2;\nBye\x07")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 2, len(c.Events))

		cev, ok := c.Events[0].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, byte(10), byte(cev))

		ev, ok := c.Events[1].(*OSCEvent)
		require.True(t, ok)

		assert.Equal(t, 2, ev.Command)
		assert.Equal(t, "Bye", ev.Data)
	})

	n.It("handles DSC sequences", func(t *testing.T) {
		tests := []struct {
			input string
			event *StringEvent
		}{
			// !OSC BEL
			{"\x1bP1;Hello\x07", &StringEvent{Kind: "DCS", Data: []byte("1;Hello")}},

			// !OSC ST (7bit)
			{"\x1bP1;Hello\x1b\\", &StringEvent{Kind: "DCS", Data: []byte("1;Hello")}},
		}

		for _, test := range tests {
			var c collectEvents

			pr, err := NewParser(strings.NewReader(test.input), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			ev, ok := c.Events[0].(*StringEvent)
			require.True(t, ok)

			assert.Equal(t, test.event, ev)
		}
	})

	n.It("handles escape cancels DCS, starts escape", func(t *testing.T) {
		input := []byte("\x1bPSomething\x1b9")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*EscapeEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("9"), cev.Data)
	})

	n.It("handles CAN cancels DCS, returns to normal", func(t *testing.T) {
		input := []byte("\x1bP12\x18AB")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		cev, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte("AB"), cev.Text)
	})

	n.It("handles C0 in DCS", func(t *testing.T) {
		input := []byte("\x1bPB\nye\x07")
		var c collectEvents

		pr, err := NewParser(bytes.NewReader(input), &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 2, len(c.Events))

		cev, ok := c.Events[0].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, byte(10), byte(cev))

		ev, ok := c.Events[1].(*StringEvent)
		require.True(t, ok)

		assert.Equal(t, "DCS", ev.Kind)
		assert.Equal(t, []byte("Bye"), ev.Data)
	})

	n.It("ignores NUL and DEL", func(t *testing.T) {
		tests := []struct {
			input string
			event Event
		}{
			{"a\x00b", &TextEvent{Text: []byte("ab")}},
			{"a\x7fb", &TextEvent{Text: []byte("ab")}},
			{"\x1b[12\x003m", csi(0x6d, 123)},
			{"\x1b[12\x7f3m", csi(0x6d, 123)},
		}

		for _, test := range tests {
			var c collectEvents

			pr, err := NewParser(strings.NewReader(test.input), &c)
			require.NoError(t, err)

			err = pr.Drive(context.TODO())
			require.Error(t, err, io.EOF)

			require.Equal(t, 1, len(c.Events))

			assert.Equal(t, test.event, c.Events[0])
		}
	})

	n.It("handles utf-8 runes properly", func(t *testing.T) {
		str := "\xe2\x9d\xaf"
		input := strings.NewReader(str)

		var c collectEvents
		pr, err := NewParser(input, &c)
		require.NoError(t, err)

		err = pr.Drive(context.TODO())
		require.Error(t, err, io.EOF)

		require.Equal(t, 1, len(c.Events))

		te, ok := c.Events[0].(*TextEvent)
		require.True(t, ok)

		assert.Equal(t, []byte(str), te.Text)
	})

	n.Meow()
}
