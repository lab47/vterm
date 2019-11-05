package screen

import (
	"testing"

	"github.com/evanphx/vterm/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestScreenReflow(t *testing.T) {
	n := neko.Modern(t)

	n.It("can resize the cells by reflowing them to a wider view", func(t *testing.T) {
		var sink sinkOps

		screen, err := NewScreen(5, 80, &sink)
		require.NoError(t, err)

		screen.getCell(2, 0).reset('b', nil)
		screen.getCell(2, 20).reset('c', nil)
		screen.getCell(3, 0).reset('d', nil)

		lineInfo := make([]state.LineInfo, 25)

		lineInfo[2].Continuation = true

		err = screen.Resize(5, 100, lineInfo)
		require.NoError(t, err)

		assert.Equal(t, 'b', screen.getCell(1, 80).val)
		assert.Equal(t, 'c', screen.getCell(2, 0).val)
		assert.Equal(t, 'd', screen.getCell(3, 0).val)
	})

	n.It("can resize the cells by reflowing them to a more narrow view", func(t *testing.T) {
		var sink sinkOps

		screen, err := NewScreen(5, 80, &sink)
		require.NoError(t, err)

		screen.getCell(2, 0).reset('b', nil)
		screen.getCell(2, 70).reset('c', nil)
		screen.getCell(3, 0).reset('d', nil)

		lineInfo := make([]state.LineInfo, 25)

		lineInfo[3].Continuation = true

		err = screen.Resize(5, 60, lineInfo)
		require.NoError(t, err)

		assert.Equal(t, 'b', screen.getCell(2, 0).val)
		assert.Equal(t, 'c', screen.getCell(3, 10).val)
		assert.Equal(t, 'd', screen.getCell(3, 11).val)
	})

	n.It("insert rows on narrow resize", func(t *testing.T) {
		var sink sinkOps

		screen, err := NewScreen(5, 80, &sink)
		require.NoError(t, err)

		screen.getCell(2, 0).reset('b', nil)
		screen.getCell(2, 70).reset('c', nil)
		screen.getCell(3, 0).reset('d', nil)

		lineInfo := make([]state.LineInfo, 25)

		err = screen.Resize(5, 60, lineInfo)
		require.NoError(t, err)

		assert.Equal(t, 'b', screen.getCell(1, 0).val)
		assert.Equal(t, 'c', screen.getCell(3, 10).val)
		assert.Equal(t, 'd', screen.getCell(4, 0).val)

	})

	n.Meow()
}
