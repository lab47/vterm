package multiplex

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type eventSink struct {
	events []Event
}

func (e *eventSink) HandleInput(ev Event) error {
	e.events = append(e.events, ev)
	return nil
}

func TestInput(t *testing.T) {
	n := neko.Modern(t)

	n.It("emits runs for normal ascii", func(t *testing.T) {
		var sink eventSink

		ir, err := NewInputReader(strings.NewReader("hello"), &sink)
		require.NoError(t, err)

		err = ir.Drive()
		require.Error(t, err, io.EOF)

		te, ok := sink.events[0].(TextEvent)
		require.True(t, ok)

		assert.Equal(t, "hello", string(te))
	})

	n.It("emits runs for utf-8", func(t *testing.T) {
		var sink eventSink

		ir, err := NewInputReader(strings.NewReader("❯"), &sink)
		require.NoError(t, err)

		err = ir.Drive()
		require.Error(t, err, io.EOF)

		te, ok := sink.events[0].(TextEvent)
		require.True(t, ok)

		r, _ := utf8.DecodeRune(te)

		assert.Equal(t, rune(10095), r)
	})

	n.It("emits mouse events for mouse escape codes", func(t *testing.T) {
		var sink eventSink

		mev := "\x1b[<3;1;2m"

		ir, err := NewInputReader(strings.NewReader(mev), &sink)
		require.NoError(t, err)

		err = ir.Drive()
		require.Error(t, err, io.EOF)

		me, ok := sink.events[0].(MouseEvent)
		require.True(t, ok)

		assert.Equal(t, 1, me.Col)
		assert.Equal(t, 2, me.Row)
		assert.Equal(t, Up, me.Op)
		assert.Equal(t, byte(3), me.Button)
	})

	n.It("emits runs for control events", func(t *testing.T) {
		var sink eventSink

		ir, err := NewInputReader(strings.NewReader("\x01\x02\x03"), &sink)
		require.NoError(t, err)

		err = ir.Drive()
		require.Error(t, err, io.EOF)

		ce, ok := sink.events[0].(ControlEvent)
		require.True(t, ok)

		assert.Equal(t, ControlEvent(0x1), ce)
	})

	n.Meow()
}
