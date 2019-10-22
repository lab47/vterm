package state

import (
	"github.com/evanphx/vterm/parser"
)

type IndexColor struct {
	Index int
}

func (i IndexColor) String() string {
	if name, ok := NamedColors[i.Index]; ok {
		return name
	}

	return ""
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

type PenAttr int

const (
	PenAttrIntensity PenAttr = iota
	PenAttrUnderline
	PenAttrStyle
	PenAttrReverse
	PenAttrStrikethrough
	PenAttrBlink
	PenAttrConceal
	PenAttrWrapper
	PenAttrOverlined
	PenAttrFont
	PenAttrFGColor
	PenAttrBGColor
)

//go:generate stringer -type=PenAttr

func (s *State) selectGraphics(ev *parser.CSIEvent) error {
	arg := ev.Args[0]
	switch arg {
	case 0:
		if s.pen.attrs&PenIntensity != PenNormal {
			err := s.output.SetPenProp(PenAttrIntensity, PenNormal)
			if err != nil {
				return err
			}
		}

		if s.pen.attrs&PenUnderline != PenNormal {
			err := s.output.SetPenProp(PenAttrUnderline, PenNormal)
			if err != nil {
				return err
			}
		}

		if s.pen.attrs&PenStyle != PenNormal {
			err := s.output.SetPenProp(PenAttrStyle, PenNormal)
			if err != nil {
				return err
			}
		}

		if s.pen.attrs&PenReverse != PenNormal {
			err := s.output.SetPenProp(PenAttrReverse, false)
			if err != nil {
				return err
			}
		}

		if s.pen.attrs&PenStrikeThrough != PenNormal {
			err := s.output.SetPenProp(PenAttrStrikethrough, false)
			if err != nil {
				return err
			}
		}

		s.pen.attrs = 0

		if s.pen.font != 0 {
			err := s.output.SetPenProp(PenAttrFont, 0)
			if err != nil {
				return err
			}
		}

		s.pen.font = 0

		def := DefaultColor{}

		if s.pen.fgColor != def {
			err := s.output.SetPenProp(PenAttrFGColor, def)
			if err != nil {
				return err
			}
			s.pen.fgColor = def
		}

		if s.pen.bgColor != def {
			err := s.output.SetPenProp(PenAttrBGColor, def)
			if err != nil {
				return err
			}
			s.pen.bgColor = def
		}
	case 1:
		if s.pen.attrs&PenIntensity != PenBold {
			s.pen.attrs &= ^PenIntensity
			s.pen.attrs |= PenBold
			return s.output.SetPenProp(PenAttrIntensity, PenBold)
		}
	case 2:
		s.pen.attrs &= ^PenIntensity
		s.pen.attrs |= PenFaint
		return s.output.SetPenProp(PenAttrIntensity, PenFaint)
	case 3:
		s.pen.attrs &= ^PenStyle
		s.pen.attrs |= PenItalic
		return s.output.SetPenProp(PenAttrStyle, PenItalic)
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

		return s.output.SetPenProp(PenAttrUnderline, s.pen.attrs&PenUnderline)
	case 5:
		s.pen.attrs |= PenBlink
		return s.output.SetPenProp(PenAttrBlink, true)
	case 7:
		s.pen.attrs |= PenReverse
		return s.output.SetPenProp(PenAttrReverse, true)
	case 8:
		s.pen.attrs |= PenConceal
		return s.output.SetPenProp(PenAttrConceal, true)
	case 9:
		s.pen.attrs |= PenStrikeThrough
		return s.output.SetPenProp(PenAttrStrikethrough, true)
	case 10, 11, 12, 13, 14, 15, 16, 17, 18, 19:
		s.pen.font = uint8(arg) - 10
		return s.output.SetPenProp(PenAttrFont, int(s.pen.font))
	case 20:
		s.pen.attrs &= ^PenStyle
		s.pen.attrs |= PenFraktur
		return s.output.SetPenProp(PenAttrStyle, PenFraktur)
	case 21:
		s.pen.attrs &= ^PenUnderline
		s.pen.attrs |= PenUnderlineDouble

		return s.output.SetPenProp(PenAttrUnderline, PenUnderlineDouble)
	case 22:
		s.pen.attrs &= ^PenIntensity
		return s.output.SetPenProp(PenAttrIntensity, PenNormal)
	case 23:
		s.pen.attrs &= ^PenStyle
		return s.output.SetPenProp(PenAttrStyle, PenNormal)
	case 24:
		s.pen.attrs &= ^PenUnderline
		return s.output.SetPenProp(PenAttrUnderline, PenNormal)
	case 25:
		s.pen.attrs &= ^PenBlink
		return s.output.SetPenProp(PenAttrBlink, false)
	case 27:
		s.pen.attrs &= ^PenReverse
		return s.output.SetPenProp(PenAttrReverse, false)
	case 28:
		s.pen.attrs &= ^PenConceal
		return s.output.SetPenProp(PenAttrConceal, false)
	case 29:
		s.pen.attrs &= ^PenStrikeThrough
		return s.output.SetPenProp(PenAttrStrikethrough, false)
	case 30, 31, 32, 33, 34, 35, 36, 37:
		newColor := IndexColor{Index: arg - 30}

		if s.pen.fgColor != newColor {
			s.pen.fgColor = newColor
			return s.output.SetPenProp(PenAttrFGColor, newColor)
		}
	case 38:
		if len(ev.Args) == 3 && ev.Args[1] == 5 {
			newColor := IndexColor{Index: ev.Args[2]}
			if s.pen.fgColor != newColor {
				s.pen.fgColor = newColor
				return s.output.SetPenProp(PenAttrFGColor, newColor)
			}
		}

		if len(ev.Args) == 5 && ev.Args[1] == 2 {
			s.pen.fgColor = RGBColor{
				Red:   uint8(ev.Args[2]),
				Green: uint8(ev.Args[3]),
				Blue:  uint8(ev.Args[4]),
			}

			return s.output.SetPenProp(PenAttrFGColor, s.pen.fgColor)
		}
	case 39:
		newColor := DefaultColor{}

		if s.pen.fgColor != newColor {
			s.pen.fgColor = newColor
			return s.output.SetPenProp(PenAttrFGColor, newColor)
		}
	case 40, 41, 42, 43, 44, 45, 46, 47:
		s.pen.bgColor = IndexColor{Index: arg - 40}
		return s.output.SetPenProp(PenAttrBGColor, s.pen.bgColor)
	case 48:
		if len(ev.Args) == 3 && ev.Args[1] == 5 {
			s.pen.bgColor = IndexColor{Index: ev.Args[2]}
			return s.output.SetPenProp(PenAttrBGColor, s.pen.bgColor)
		}

		if len(ev.Args) == 5 && ev.Args[1] == 2 {
			s.pen.bgColor = RGBColor{
				Red:   uint8(ev.Args[2]),
				Green: uint8(ev.Args[3]),
				Blue:  uint8(ev.Args[4]),
			}

			return s.output.SetPenProp(PenAttrBGColor, s.pen.bgColor)
		}
	case 49:
		s.pen.bgColor = DefaultColor{}
		return s.output.SetPenProp(PenAttrBGColor, s.pen.bgColor)
	case 51:
		s.pen.attrs &= ^PenWrapper
		s.pen.attrs |= PenFramed
		return s.output.SetPenProp(PenAttrWrapper, PenFramed)
	case 52:
		s.pen.attrs &= ^PenWrapper
		s.pen.attrs |= PenEncircled
		return s.output.SetPenProp(PenAttrWrapper, PenEncircled)
	case 53:
		s.pen.attrs |= PenOverlined
		return s.output.SetPenProp(PenAttrOverlined, true)
	case 54:
		s.pen.attrs &= ^PenWrapper
		return s.output.SetPenProp(PenAttrWrapper, PenNormal)
	case 55:
		s.pen.attrs &= ^PenOverlined
		return s.output.SetPenProp(PenAttrOverlined, false)
	case 90, 91, 92, 93, 94, 95, 96, 97:
		s.pen.fgColor = IndexColor{Index: (arg - 90) + 8}
		return s.output.SetPenProp(PenAttrFGColor, s.pen.fgColor)
	case 100, 101, 102, 103, 104, 105, 106, 107:
		s.pen.bgColor = IndexColor{Index: (arg - 100) + 8}
		return s.output.SetPenProp(PenAttrFGColor, s.pen.bgColor)
	}
	return nil
}
