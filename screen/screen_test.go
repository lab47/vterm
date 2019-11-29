package screen

import (
	"testing"

	"github.com/evanphx/vterm/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type sinkOps struct {
	damaged []state.Rect
}

func (s *sinkOps) DamageDone(r state.Rect) error {
	s.damaged = append(s.damaged, r)
	return nil
}

func (s *sinkOps) MoveCursor(p state.Pos) error {
	panic("not implemented")
}

func (s *sinkOps) Output(b []byte) error {
	panic("not implemented")
}

func (s *sinkOps) SetTermProp(prop state.TermAttr, val interface{}) error {
	panic("not implemented")
}

func (s *sinkOps) StringEvent(kind string, b []byte) error {
	panic("not implemented")
}

func TestScreen(t *testing.T) {
	n := neko.Modern(t)

	n.It("can scroll a region to the right", func(t *testing.T) {
		var sink sinkOps
		screen, err := NewScreen(25, 80, &sink)
		require.NoError(t, err)

		rect := state.Rect{
			Start: state.Pos{Row: 1, Col: 1},
			End:   state.Pos{Row: 2, Col: 2},
		}

		screen.getCell(1, 1).reset('a', nil)
		screen.getCell(2, 1).reset('b', nil)

		sr := state.ScrollRect{
			Rect:      rect,
			Direction: state.ScrollRight,
			Distance:  1,
		}

		err = screen.ScrollRect(sr)
		require.NoError(t, err)

		// Check for the move
		assert.Equal(t, 'a', screen.getCell(1, 2).val)
		assert.Equal(t, 'b', screen.getCell(2, 2).val)

		// Check erasure
		assert.Equal(t, rune(0), screen.getCell(1, 1).val, "old value not erased")
		assert.Equal(t, rune(0), screen.getCell(2, 1).val, "old value not erased")

	})

	n.It("can scroll a region to the left", func(t *testing.T) {
		var sink sinkOps
		screen, err := NewScreen(25, 80, &sink)
		require.NoError(t, err)

		rect := state.Rect{
			Start: state.Pos{Row: 1, Col: 0},
			End:   state.Pos{Row: 2, Col: 1},
		}

		screen.getCell(1, 1).reset('a', nil)
		screen.getCell(2, 1).reset('b', nil)

		sr := state.ScrollRect{
			Rect:      rect,
			Direction: state.ScrollLeft,
			Distance:  1,
		}

		err = screen.ScrollRect(sr)
		require.NoError(t, err)

		// Check erasure
		assert.Equal(t, rune(0), screen.getCell(1, 1).val)
		assert.Equal(t, rune(0), screen.getCell(2, 1).val)

		// Check for the move
		assert.Equal(t, 'a', screen.getCell(1, 0).val)
		assert.Equal(t, 'b', screen.getCell(2, 0).val)
	})

	n.It("can scroll a region down", func(t *testing.T) {
		var sink sinkOps
		screen, err := NewScreen(25, 80, &sink)
		require.NoError(t, err)

		rect := state.Rect{
			Start: state.Pos{Row: 1, Col: 1},
			End:   state.Pos{Row: 3, Col: 1},
		}

		screen.buffer.setCell(1, 1, ScreenCell{val: 'a'})
		screen.buffer.setCell(2, 1, ScreenCell{val: 'b'})

		sr := state.ScrollRect{
			Rect:      rect,
			Direction: state.ScrollDown,
			Distance:  1,
		}

		err = screen.ScrollRect(sr)
		require.NoError(t, err)

		// Check erasure
		assert.Equal(t, rune(0), screen.getCell(1, 1).val, "old value not erased")

		// Check for the move
		assert.Equal(t, 'a', screen.getCell(2, 1).val)
		assert.Equal(t, 'b', screen.getCell(3, 1).val)
	})

	n.It("can scroll a region up", func(t *testing.T) {
		var sink sinkOps
		screen, err := NewScreen(25, 80, &sink)
		require.NoError(t, err)

		rect := state.Rect{
			Start: state.Pos{Row: 0, Col: 1},
			End:   state.Pos{Row: 2, Col: 1},
		}

		screen.getCell(1, 1).reset('a', nil)
		screen.getCell(2, 1).reset('b', nil)

		sr := state.ScrollRect{
			Rect:      rect,
			Direction: state.ScrollUp,
			Distance:  1,
		}

		err = screen.ScrollRect(sr)
		require.NoError(t, err)

		// Check erasure
		assert.Equal(t, rune(0), screen.getCell(2, 1).val, "old value not erased")

		// Check for the move
		assert.Equal(t, 'a', screen.getCell(0, 1).val)
		assert.Equal(t, 'b', screen.getCell(1, 1).val)
	})

	n.It("can scroll a region up and the old content is erased on a line", func(t *testing.T) {
		var sink sinkOps
		screen, err := NewScreen(25, 80, &sink)
		require.NoError(t, err)

		rect := state.Rect{
			Start: state.Pos{Row: 0, Col: 1},
			End:   state.Pos{Row: 2, Col: 3},
		}

		screen.getCell(1, 1).reset('a', nil)
		screen.getCell(1, 2).reset('b', nil)
		screen.getCell(1, 3).reset('c', nil)
		screen.getCell(2, 1).reset('d', nil)

		sr := state.ScrollRect{
			Rect:      rect,
			Direction: state.ScrollUp,
			Distance:  1,
		}

		err = screen.ScrollRect(sr)
		require.NoError(t, err)

		// Check erasure
		assert.Equal(t, rune(0), screen.getCell(2, 1).val, "old value not erased")

		// Check for the move
		assert.Equal(t, 'a', screen.getCell(0, 1).val)
		assert.Equal(t, 'b', screen.getCell(0, 2).val)
		assert.Equal(t, 'c', screen.getCell(0, 3).val)
		assert.Equal(t, 'd', screen.getCell(1, 1).val)
		assert.Equal(t, rune(0), screen.getCell(1, 2).val)
		assert.Equal(t, rune(0), screen.getCell(1, 3).val)
	})

	n.Meow()
}
