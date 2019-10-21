package screen

import (
	"github.com/evanphx/vterm/parser"
)

type IndexColor struct {
	Index int
}

type RGBColor struct {
	Red, Green, Blue uint8
}

type DefaultColor struct{}

type Color interface{}

type PenState struct {
	attrs PenGraphic
	font  uint8

	fgColor Color
	bgColor Color
}

type PenGraphic uint16

const (
	// used as separate bits, so specified directly
	PenNormal PenGraphic = iota

	PenBold  PenGraphic = (1 << iota)
	PenFaint PenGraphic = (1 << iota)

	PenBlink   PenGraphic = (1 << iota)
	PenConceal PenGraphic = (1 << iota)

	PenItalic  PenGraphic = (1 << iota)
	PenFraktur PenGraphic = (1 << iota)

	PenUnderlineSingle PenGraphic = (1 << iota)
	PenUnderlineDouble PenGraphic = (1 << iota)

	PenReverse       PenGraphic = (1 << iota)
	PenStrikeThrough PenGraphic = (1 << iota)

	PenFramed    PenGraphic = (1 << iota)
	PenEncircled PenGraphic = (1 << iota)

	PenOverlined PenGraphic = (1 << iota)

	PenIntensity PenGraphic = PenBold | PenFaint
	PenStyle     PenGraphic = PenItalic | PenFraktur

	PenUnderlineCurly PenGraphic = PenUnderlineSingle | PenUnderlineDouble
	PenUnderline      PenGraphic = PenUnderlineCurly

	PenWrapper PenGraphic = PenFramed | PenEncircled
)

//go:generate stringer -type=PenGraphic

func (s *State) selectGraphics(ev *parser.CSIEvent) error {
	arg := ev.Args[0]
	switch arg {
	case 1:
		s.pen.attrs &= ^PenIntensity
		s.pen.attrs |= PenBold
		return s.output.SetPenProp("intensity", PenBold)
	case 2:
		s.pen.attrs &= ^PenIntensity
		s.pen.attrs |= PenFaint
		return s.output.SetPenProp("intensity", PenFaint)
	case 3:
		s.pen.attrs &= ^PenStyle
		s.pen.attrs |= PenItalic
		return s.output.SetPenProp("style", PenItalic)
	case 4:
		// Reset all underline values to reset them properly
		s.pen.attrs &= ^PenUnderline

		if len(ev.Args) > 1 {
			switch ev.Args[1] {
			case 0:
				// nothing, this is off
			case 1:
				s.pen.attrs |= PenUnderlineSingle
			case 2:
				s.pen.attrs |= PenUnderlineDouble
			case 3:
				s.pen.attrs |= PenUnderlineCurly
			}
		} else {
			s.pen.attrs |= PenUnderlineSingle
		}

		return s.output.SetPenProp("underline", s.pen.attrs&PenUnderline)
	case 5:
		s.pen.attrs |= PenBlink
		return s.output.SetPenProp("blink", true)
	case 7:
		s.pen.attrs |= PenReverse
		return s.output.SetPenProp("reverse", true)
	case 8:
		s.pen.attrs |= PenConceal
		return s.output.SetPenProp("conceal", true)
	case 9:
		s.pen.attrs |= PenStrikeThrough
		return s.output.SetPenProp("strikethrough", true)
	case 10, 11, 12, 13, 14, 15, 16, 17, 18, 19:
		s.pen.font = uint8(arg) - 10
		return s.output.SetPenProp("font", int(s.pen.font))
	case 20:
		s.pen.attrs &= ^PenStyle
		s.pen.attrs |= PenFraktur
		return s.output.SetPenProp("style", PenFraktur)
	case 21:
		s.pen.attrs &= ^PenUnderline
		s.pen.attrs |= PenUnderlineDouble

		return s.output.SetPenProp("underline", PenUnderlineDouble)
	case 22:
		s.pen.attrs &= ^PenIntensity
		return s.output.SetPenProp("intensity", PenNormal)
	case 23:
		s.pen.attrs &= ^PenStyle
		return s.output.SetPenProp("style", PenNormal)
	case 24:
		s.pen.attrs &= ^PenUnderline
		return s.output.SetPenProp("underline", PenNormal)
	case 25:
		s.pen.attrs &= ^PenBlink
		return s.output.SetPenProp("blink", false)
	case 27:
		s.pen.attrs &= ^PenReverse
		return s.output.SetPenProp("reverse", false)
	case 28:
		s.pen.attrs &= ^PenConceal
		return s.output.SetPenProp("conceal", false)
	case 29:
		s.pen.attrs &= ^PenStrikeThrough
		return s.output.SetPenProp("strikethrough", false)
	case 30, 31, 32, 33, 34, 35, 36, 37:
		s.pen.fgColor = IndexColor{Index: arg - 30}
		return s.output.SetPenProp("fg_color", s.pen.fgColor)
	case 38:
		if len(ev.Args) == 3 && ev.Args[1] == 5 {
			s.pen.fgColor = IndexColor{Index: ev.Args[2]}
			return s.output.SetPenProp("fg_color", s.pen.fgColor)
		}

		if len(ev.Args) == 5 && ev.Args[1] == 2 {
			s.pen.fgColor = RGBColor{
				Red:   uint8(ev.Args[2]),
				Green: uint8(ev.Args[3]),
				Blue:  uint8(ev.Args[4]),
			}

			return s.output.SetPenProp("fg_color", s.pen.fgColor)
		}
	case 39:
		s.pen.fgColor = DefaultColor{}
		return s.output.SetPenProp("fg_color", s.pen.fgColor)
	case 40, 41, 42, 43, 44, 45, 46, 47:
		s.pen.bgColor = IndexColor{Index: arg - 40}
		return s.output.SetPenProp("bg_color", s.pen.bgColor)
	case 48:
		if len(ev.Args) == 3 && ev.Args[1] == 5 {
			s.pen.bgColor = IndexColor{Index: ev.Args[2]}
			return s.output.SetPenProp("bg_color", s.pen.bgColor)
		}

		if len(ev.Args) == 5 && ev.Args[1] == 2 {
			s.pen.bgColor = RGBColor{
				Red:   uint8(ev.Args[2]),
				Green: uint8(ev.Args[3]),
				Blue:  uint8(ev.Args[4]),
			}

			return s.output.SetPenProp("bg_color", s.pen.bgColor)
		}
	case 49:
		s.pen.bgColor = DefaultColor{}
		return s.output.SetPenProp("bg_color", s.pen.bgColor)
	case 51:
		s.pen.attrs &= ^PenWrapper
		s.pen.attrs |= PenFramed
		return s.output.SetPenProp("wrapper", PenFramed)
	case 52:
		s.pen.attrs &= ^PenWrapper
		s.pen.attrs |= PenEncircled
		return s.output.SetPenProp("wrapper", PenEncircled)
	case 53:
		s.pen.attrs |= PenOverlined
		return s.output.SetPenProp("overlined", true)
	case 54:
		s.pen.attrs &= ^PenWrapper
		return s.output.SetPenProp("wrapper", PenNormal)
	case 55:
		s.pen.attrs &= ^PenOverlined
		return s.output.SetPenProp("overlined", false)
	case 90, 91, 92, 93, 94, 95, 96, 97:
		s.pen.fgColor = IndexColor{Index: (arg - 90) + 8}
		return s.output.SetPenProp("fg_color", s.pen.fgColor)
	case 100, 101, 102, 103, 104, 105, 106, 107:
		s.pen.bgColor = IndexColor{Index: (arg - 100) + 8}
		return s.output.SetPenProp("bg_color", s.pen.bgColor)
	}
	return nil
}
