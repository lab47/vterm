package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektra/neko"
)

func TestRect(t *testing.T) {
	n := neko.Modern(t)

	n.It("can split a rectangle into even columns", func(t *testing.T) {
		r := Rect{
			Start: Pos{0, 0},
			End:   Pos{24, 79},
		}

		left, right := r.SplitEvenColumns()

		assert.Equal(t, Pos{0, 0}, left.Start)
		assert.Equal(t, Pos{Row: 24, Col: 39}, left.End)
		assert.Equal(t, Pos{Row: 0, Col: 40}, right.Start)
		assert.Equal(t, Pos{Row: 24, Col: 79}, right.End)
	})

	n.It("can split a rectangle into even rows", func(t *testing.T) {
		r := Rect{
			Start: Pos{0, 0},
			End:   Pos{24, 79},
		}

		left, right := r.SplitEvenRows()

		assert.Equal(t, Pos{0, 0}, left.Start)
		assert.Equal(t, Pos{Row: 11, Col: 79}, left.End)
		assert.Equal(t, Pos{Row: 12, Col: 0}, right.Start)
		assert.Equal(t, Pos{Row: 24, Col: 79}, right.End)
	})

	n.It("can split a rectangle into columns by percentage", func(t *testing.T) {
		r := Rect{
			Start: Pos{0, 0},
			End:   Pos{24, 79},
		}

		left, right := r.SplitColumns(25.0)

		assert.Equal(t, Pos{0, 0}, left.Start)
		assert.Equal(t, Pos{Row: 24, Col: 59}, left.End)
		assert.Equal(t, Pos{Row: 0, Col: 60}, right.Start)
		assert.Equal(t, Pos{Row: 24, Col: 79}, right.End)
	})

	n.It("can split a rectangle into rows by percentage", func(t *testing.T) {
		r := Rect{
			Start: Pos{0, 0},
			End:   Pos{24, 79},
		}

		left, right := r.SplitRows(25.0)

		assert.Equal(t, Pos{0, 0}, left.Start)
		assert.Equal(t, Pos{Row: 17, Col: 79}, left.End)
		assert.Equal(t, Pos{Row: 18, Col: 0}, right.Start)
		assert.Equal(t, Pos{Row: 24, Col: 79}, right.End)
	})

	n.Meow()
}
