package screen

import (
	"testing"

	"github.com/evanphx/vterm/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestScreenPen(t *testing.T) {
	n := neko.Modern(t)

	n.It("can set the current pen attributes", func(t *testing.T) {
		var sink opSink

		screen, err := NewState(25, 80, &sink)
		require.NoError(t, err)

		wrap := func(g, m PenGraphic, f func()) {
			sink.penProps = nil
			screen.pen.fgColor = nil
			screen.pen.attrs = 0

			f()

			assert.Equal(t, g, screen.pen.attrs&m, "%s masked as %s (pen: %s)", g, m, screen.pen.attrs)
		}

		checkProp := func(name string, val interface{}) {
			require.Equal(t, 1, len(sink.penProps))
			assert.Equal(t, name, sink.penProps[0].prop)
			assert.Equal(t, val, sink.penProps[0].val, "prop: %s", name)
		}

		wrap(PenBold, PenIntensity, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{1}})
			require.NoError(t, err)

			checkProp("intensity", PenBold)
		})

		wrap(PenFaint, PenIntensity, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{2}})
			require.NoError(t, err)

			checkProp("intensity", PenFaint)
		})

		wrap(PenItalic, PenStyle, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{3}})
			require.NoError(t, err)

			checkProp("style", PenItalic)
		})

		wrap(PenUnderlineSingle, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineSingle)
		})

		wrap(0, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 0}})
			require.NoError(t, err)

			checkProp("underline", PenNormal)
		})

		wrap(PenUnderlineSingle, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 1}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineSingle)
		})

		wrap(PenUnderlineDouble, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 2}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineDouble)
		})

		wrap(PenUnderlineCurly, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{4, 3}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineCurly)
		})

		wrap(PenBlink, PenBlink, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{5}})
			require.NoError(t, err)

			checkProp("blink", true)
		})

		wrap(PenReverse, PenReverse, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{7}})
			require.NoError(t, err)

			checkProp("reverse", true)
		})

		wrap(PenConceal, PenConceal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{8}})
			require.NoError(t, err)

			checkProp("conceal", true)
		})

		wrap(PenStrikeThrough, PenStrikeThrough, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{9}})
			require.NoError(t, err)

			checkProp("strikethrough", true)
		})

		for i := uint8(10); i < 20; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{int(i)}})
				require.NoError(t, err)

				assert.Equal(t, i-10, screen.pen.font)

				checkProp("font", int(i-10))
			})
		}

		wrap(PenFraktur, PenFraktur, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{20}})
			require.NoError(t, err)

			checkProp("style", PenFraktur)
		})

		wrap(PenUnderlineDouble, PenUnderline, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{21}})
			require.NoError(t, err)

			checkProp("underline", PenUnderlineDouble)
		})

		wrap(PenNormal, PenBold, func() {
			screen.pen.attrs |= PenBold
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{22}})
			require.NoError(t, err)

			checkProp("intensity", PenNormal)
		})

		wrap(PenNormal, PenItalic, func() {
			screen.pen.attrs |= PenItalic
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{23}})
			require.NoError(t, err)

			checkProp("style", PenNormal)
		})

		wrap(PenNormal, PenUnderline, func() {
			screen.pen.attrs |= PenUnderline
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{24}})
			require.NoError(t, err)

			checkProp("underline", PenNormal)
		})

		wrap(PenNormal, PenBlink, func() {
			screen.pen.attrs |= PenBlink
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{25}})
			require.NoError(t, err)

			checkProp("blink", false)
		})

		wrap(PenNormal, PenReverse, func() {
			screen.pen.attrs |= PenReverse
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{27}})
			require.NoError(t, err)

			checkProp("reverse", false)
		})

		wrap(PenNormal, PenConceal, func() {
			screen.pen.attrs |= PenConceal
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{28}})
			require.NoError(t, err)

			checkProp("conceal", false)
		})

		wrap(PenNormal, PenStrikeThrough, func() {
			screen.pen.attrs |= PenStrikeThrough
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{29}})
			require.NoError(t, err)

			checkProp("strikethrough", false)
		})

		for i := 30; i < 38; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: i - 30}, screen.pen.fgColor)

				checkProp("fg_color", IndexColor{Index: i - 30})
			})
		}

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{38, 5, 132}})
			require.NoError(t, err)

			assert.Equal(t, IndexColor{Index: 132}, screen.pen.fgColor)

			checkProp("fg_color", IndexColor{Index: 132})
		})

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{38, 2, 55, 77, 99}})
			require.NoError(t, err)

			assert.Equal(t, RGBColor{Red: 55, Green: 77, Blue: 99}, screen.pen.fgColor)

			checkProp("fg_color", RGBColor{Red: 55, Green: 77, Blue: 99})
		})

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{39}})
			require.NoError(t, err)

			assert.Equal(t, DefaultColor{}, screen.pen.fgColor)

			checkProp("fg_color", DefaultColor{})
		})

		for i := 40; i < 48; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: i - 40}, screen.pen.bgColor)

				checkProp("bg_color", IndexColor{Index: i - 40})
			})
		}

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{48, 5, 132}})
			require.NoError(t, err)

			assert.Equal(t, IndexColor{Index: 132}, screen.pen.bgColor)

			checkProp("bg_color", IndexColor{Index: 132})
		})

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{48, 2, 55, 77, 99}})
			require.NoError(t, err)

			assert.Equal(t, RGBColor{Red: 55, Green: 77, Blue: 99}, screen.pen.bgColor)

			checkProp("bg_color", RGBColor{Red: 55, Green: 77, Blue: 99})
		})

		wrap(PenNormal, PenNormal, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{49}})
			require.NoError(t, err)

			assert.Equal(t, DefaultColor{}, screen.pen.bgColor)

			checkProp("bg_color", DefaultColor{})
		})

		wrap(PenFramed, PenWrapper, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{51}})
			require.NoError(t, err)

			checkProp("wrapper", PenFramed)
		})

		wrap(PenEncircled, PenWrapper, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{52}})
			require.NoError(t, err)

			checkProp("wrapper", PenEncircled)
		})

		wrap(PenOverlined, PenOverlined, func() {
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{53}})
			require.NoError(t, err)

			checkProp("overlined", true)
		})

		wrap(PenNormal, PenWrapper, func() {
			screen.pen.attrs |= PenEncircled
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{54}})
			require.NoError(t, err)

			checkProp("wrapper", PenNormal)
		})

		wrap(PenNormal, PenOverlined, func() {
			screen.pen.attrs |= PenOverlined
			err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{55}})
			require.NoError(t, err)

			checkProp("overlined", false)
		})

		for i := 90; i < 98; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: (i - 90) + 8}, screen.pen.fgColor)

				checkProp("fg_color", IndexColor{Index: (i - 90) + 8})
			})
		}

		for i := 100; i < 108; i++ {
			wrap(PenNormal, PenNormal, func() {
				err = screen.HandleEvent(&parser.CSIEvent{Command: 'm', Args: []int{i}})
				require.NoError(t, err)

				assert.Equal(t, IndexColor{Index: (i - 100) + 8}, screen.pen.bgColor)

				checkProp("bg_color", IndexColor{Index: (i - 100) + 8})
			})
		}

	})

	n.Meow()

}
