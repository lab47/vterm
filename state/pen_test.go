package state

import (
	"testing"

	"github.com/evanphx/vterm/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TeststatePen(t *testing.T) {
	n := neko.Modern(t)

	n.It("can set the current pen attributes", func(t *testing.T) {
		var sink opSink

		state, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		wrap := func(g, m PenGraphic, f func()) {
			sink.penProps = nil
			state.pen.fgColor = nil
			state.pen.attrs = 0

			f()

			assert.Equal(t, g, state.pen.attrs&m, "%s masked as %s (pen: %s)", g, m, state.pen.attrs)
		}

		checkProp := func(name string, val interface{}) {
			require.Equal(t, 1, len(sink.penProps))
			assert.Equal(t, name, sink.penProps[0].prop)
			assert.Equal(t, val, sink.penProps[0].val, "prop: %s", name)
		}

		wrap(PenBold, PenIntensity, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{1}})
			require.NoError(t, err)

			checkProp("intensity", PenBold)
		})

		wrap(PenFaint, PenIntensity, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{2}})
			require.NoError(t, err)

			checkProp("intensity", PenFaint)
		})

		wrap(PenItalic, PenStyle, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{3}})
			require.NoError(t, err)

			checkProp("style", PenItalic)
		})

		wrap(PenUnderlineSingle, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineSingle)
		})

		wrap(0, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 0}})
			require.NoError(t, err)

			checkProp("underline", PenNormal)
		})

		wrap(PenUnderlineSingle, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 1}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineSingle)
		})

		wrap(PenUnderlineDouble, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 2}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineDouble)
		})

		wrap(PenUnderlineCurly, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 3}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineCurly)
		})

		wrap(PenBlink, PenBlink, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{5}})
			require.NoError(t, err)

			checkProp("blink", true)
		})

		wrap(PenReverse, PenReverse, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{7}})
			require.NoError(t, err)

			checkProp("reverse", true)
		})

		wrap(PenConceal, PenConceal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{8}})
			require.NoError(t, err)

			checkProp("conceal", true)
		})

		wrap(PenStrikeThrough, PenStrikeThrough, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{9}})
			require.NoError(t, err)

			checkProp("strikethrough", true)
		})

		for i := uint8(10); i < 20; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{int(i)}})
				require.NoError(t, err)

				assert.Equal(t, i-10, state.pen.font)

				checkProp("font", int(i-10))
			})
		}

		wrap(PenFraktur, PenFraktur, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{20}})
			require.NoError(t, err)

			checkProp("style", PenFraktur)
		})

		wrap(PenUnderlineDouble, PenUnderline, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{21}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineDouble)
		})

		wrap(PenNormal, PenBold, func() {
			state.pen.attrs |= PenBold
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{22}})
			require.NoError(t, err)

			checkProp("intensity", PenNormal)
		})

		wrap(PenNormal, PenItalic, func() {
			state.pen.attrs |= PenItalic
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{23}})
			require.NoError(t, err)

			checkProp("style", PenNormal)
		})

		wrap(PenNormal, PenUnderline, func() {
			state.pen.attrs |= PenUnderline
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{24}})
			require.NoError(t, err)

			checkProp("underline", PenNormal)
		})

		wrap(PenNormal, PenBlink, func() {
			state.pen.attrs |= PenBlink
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{25}})
			require.NoError(t, err)

			checkProp("blink", false)
		})

		wrap(PenNormal, PenReverse, func() {
			state.pen.attrs |= PenReverse
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{27}})
			require.NoError(t, err)

			checkProp("reverse", false)
		})

		wrap(PenNormal, PenConceal, func() {
			state.pen.attrs |= PenConceal
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{28}})
			require.NoError(t, err)

			checkProp("conceal", false)
		})

		wrap(PenNormal, PenStrikeThrough, func() {
			state.pen.attrs |= PenStrikeThrough
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{29}})
			require.NoError(t, err)

			checkProp("strikethrough", false)
		})

		for i := 30; i < 38; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: i - 30}, state.pen.fgColor)

				checkProp("fg_color", IndexColor{Index: i - 30})
			})
		}

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{38, 5, 132}})
			require.NoError(t, err)

			assert.Equal(t, IndexColor{Index: 132}, state.pen.fgColor)

			checkProp("fg_color", IndexColor{Index: 132})
		})

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{38, 2, 55, 77, 99}})
			require.NoError(t, err)

			assert.Equal(t, RGBColor{Red: 55, Green: 77, Blue: 99}, state.pen.fgColor)

			checkProp("fg_color", RGBColor{Red: 55, Green: 77, Blue: 99})
		})

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{39}})
			require.NoError(t, err)

			assert.Equal(t, DefaultColor{}, state.pen.fgColor)

			checkProp("fg_color", DefaultColor{})
		})

		for i := 40; i < 48; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: i - 40}, state.pen.bgColor)

				checkProp("bg_color", IndexColor{Index: i - 40})
			})
		}

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{48, 5, 132}})
			require.NoError(t, err)

			assert.Equal(t, IndexColor{Index: 132}, state.pen.bgColor)

			checkProp("bg_color", IndexColor{Index: 132})
		})

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{48, 2, 55, 77, 99}})
			require.NoError(t, err)

			assert.Equal(t, RGBColor{Red: 55, Green: 77, Blue: 99}, state.pen.bgColor)

			checkProp("bg_color", RGBColor{Red: 55, Green: 77, Blue: 99})
		})

		wrap(PenNormal, PenNormal, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{49}})
			require.NoError(t, err)

			assert.Equal(t, DefaultColor{}, state.pen.bgColor)

			checkProp("bg_color", DefaultColor{})
		})

		wrap(PenFramed, PenWrapper, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{51}})
			require.NoError(t, err)

			checkProp("wrapper", PenFramed)
		})

		wrap(PenEncircled, PenWrapper, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{52}})
			require.NoError(t, err)

			checkProp("wrapper", PenEncircled)
		})

		wrap(PenOverlined, PenOverlined, func() {
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{53}})
			require.NoError(t, err)

			checkProp("overlined", true)
		})

		wrap(PenNormal, PenWrapper, func() {
			state.pen.attrs |= PenEncircled
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{54}})
			require.NoError(t, err)

			checkProp("wrapper", PenNormal)
		})

		wrap(PenNormal, PenOverlined, func() {
			state.pen.attrs |= PenOverlined
			err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{55}})
			require.NoError(t, err)

			checkProp("overlined", false)
		})

		for i := 90; i < 98; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: (i - 90) + 8}, state.pen.fgColor)

				checkProp("fg_color", IndexColor{Index: (i - 90) + 8})
			})
		}

		for i := 100; i < 108; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = state.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: (i - 100) + 8}, state.pen.bgColor)

				checkProp("bg_color", IndexColor{Index: (i - 100) + 8})
			})
		}

	})

	n.Meow()

}
