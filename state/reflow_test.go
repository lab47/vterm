package state

import (
	"strings"
	"testing"

	"github.com/evanphx/vterm/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestStateReflow(t *testing.T) {
	n := neko.Modern(t)

	n.It("generates SetCell for normal output", func(t *testing.T) {
		var sink opSink

		state, err := NewState(20, 80, &sink)
		require.NoError(t, err)

		long := strings.Repeat("X", 100)

		err = state.HandleEvent(&parser.TextEvent{
			Text: []byte(long),
		})

		require.NoError(t, err)

		assert.True(t, state.lineInfo[1].Continuation)

		err = state.Resize(20, 110)
		require.NoError(t, err)

		assert.Equal(t, 110, sink.resize.cols)
		assert.Equal(t, 20, sink.resize.rows)
		assert.True(t, sink.resize.lines[1].Continuation)
	})

	n.Meow()
}
