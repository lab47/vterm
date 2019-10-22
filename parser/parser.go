package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type EventHandler interface {
	HandleEvent(Event) error
}

type Parser struct {
	br      *bufio.Reader
	plain   bytes.Buffer
	handler EventHandler
}

func NewParser(r io.Reader, h EventHandler) (*Parser, error) {
	br := bufio.NewReader(r)

	parser := &Parser{
		br:      br,
		handler: h,
	}

	return parser, nil
}

type Event interface{}

const (
	NUL = 0x0
	DEL = 0x7f
	CAN = 0x18
	SUB = 0x1a
	ESC = 0x1b
	BEL = 0x7
	C0  = 0x20
)

func (p *Parser) Drive() error {
	for {
		b, err := p.br.ReadByte()
		if err != nil {
			return err
		}

		switch b {
		case NUL, DEL:
			continue
		case CAN, SUB:
			continue
		case ESC:
			err := p.readEsc()
			if err != nil {
				return err
			}
			continue
		default:
			if b < C0 {
				err := p.readControl(b)
				if err != nil {
					return err
				}

				continue
			}
		}

		err = p.br.UnreadByte()
		if err != nil {
			return err
		}

	normal:
		for {
			b, err := p.br.ReadByte()
			if err != nil {
				if p.plain.Len() > 0 {
					p.readSpan()
				}

				return err
			}

			switch b {
			case NUL, DEL, CAN, SUB:
				continue normal
			case ESC:
				err = p.br.UnreadByte()
				if err != nil {
					return err
				}

				err := p.readSpan()
				if err != nil {
					return err
				}

				break normal
			default:
				if b < C0 {
					err = p.br.UnreadByte()
					if err != nil {
						return err
					}

					err := p.readSpan()
					if err != nil {
						return err
					}

					break normal
				}
			}

			err = p.br.UnreadByte()
			if err != nil {
				return err
			}

			r, _, err := p.br.ReadRune()
			if err != nil {
				return err
			}

			_, err = p.plain.WriteRune(r)
			if err != nil {
				return err
			}
		}
	}
}

type TextEvent struct {
	Text []byte
}

func (p *Parser) readSpan() error {
	buf := make([]byte, p.plain.Len())

	_, err := p.plain.Read(buf)
	if err != nil {
		return err
	}

	return p.handler.HandleEvent(&TextEvent{buf})
}

type ControlEvent struct {
	Control byte
}

func (c *ControlEvent) String() string {
	return fmt.Sprintf("CTL: %#v (0x%x)", string(c.Control), c.Control)
}

func (p *Parser) readControl(b byte) error {
	return p.handler.HandleEvent(&ControlEvent{b})
}

func isIntermed(b byte) bool {
	return b >= 0x20 && b <= 0x2f
}

type EscapeEvent struct {
	Data []byte
}

func (p *Parser) readEsc() error {
	var intermed []byte
top:
	for {
		b, err := p.br.ReadByte()
		if err != nil {
			return err
		}

		switch b {
		case NUL, DEL:
			continue
		case ESC:
			intermed = nil
			continue
		case CAN, SUB:
			return nil
		default:
			if b < C0 {
				p.readControl(b)
				continue top
			}
		}

		switch b {
		case 0x50: // DCS
			return p.readString("DCS")
		case 0x5b: // CSI
			return p.readCSI()
		case 0x5d: // OSC
			return p.readString("OSC")
		default:
			if isIntermed(b) {
				intermed = append(intermed, b)
			} else if len(intermed) == 0 && b >= 0x40 && b < 0x60 {
				return p.readControl(b + 0x40)
			} else if b >= 0x30 && b < 0x7f {
				intermed = append(intermed, b)
				return p.handler.HandleEvent(&EscapeEvent{intermed})
			} else {
				L.Debug("Unhandled byte in escape", "byte", b)
			}
		}
	}
}

type OSCEvent struct {
	Command int
	Data    string
}

type StringEvent struct {
	Kind string
	Data []byte
}

func (p *Parser) emitStringEvent(kind string, data []byte) error {
	if kind == "OSC" {
		str := string(data)
		if sc := strings.IndexByte(str, ';'); sc != -1 {
			if cmd, err := strconv.Atoi(str[:sc]); err == nil {
				return p.handler.HandleEvent(&OSCEvent{
					Command: cmd,
					Data:    str[sc+1:],
				})
			}
		}
	}

	return p.handler.HandleEvent(&StringEvent{
		Kind: kind,
		Data: data,
	})
}

func (p *Parser) readString(kind string) error {
	var data []byte

top:
	for {
		b, err := p.br.ReadByte()
		if err != nil {
			return err
		}

		switch b {
		case NUL, DEL:
			continue top
		case CAN, SUB:
			return nil
		case ESC:
			b, err := p.br.ReadByte()
			if err != nil {
				return err
			}

			if b == 0x5c {
				return p.emitStringEvent(kind, data)
			}

			err = p.br.UnreadByte()
			if err != nil {
				return err
			}

			return p.readEsc()
		default:
			switch {
			case b == 0x7:
				return p.emitStringEvent(kind, data)
			case b < C0:
				p.readControl(b)
				continue top
			default:
				data = append(data, b)
			}
		}
	}
}

type CSIEvent struct {
	Command  byte
	Leader   []byte
	Args     []int
	Intermed []byte
}

func (c *CSIEvent) CSICommand() CSICommand {
	idx := CSICommand(c.Command)
	if len(c.Leader) == 1 {
		idx = LEADER(c.Leader[0], c.Command)
	}

	if len(c.Intermed) == 1 {
		idx = INTERMED(c.Intermed[0], c.Command)
	}

	return idx
}

func (c *CSIEvent) String() string {
	cmd := c.CSICommand()

	return fmt.Sprintf("CSI: %s (0x%x) Leader=%#v Args=%#v Intermed=%#v", cmd.String(), c.Command, c.Leader, c.Args, c.Intermed)
}

func (p *Parser) readCSI() error {
	const (
		LEADER   = 1
		ARG      = 2
		INTERMED = 3
	)

	var (
		leader   []byte
		state    int = LEADER
		arg      int = -1
		args     []int
		intermed []byte
	)

top:
	for {
		b, err := p.br.ReadByte()
		if err != nil {
			p.handler.HandleEvent(&CSIEvent{
				Command:  b,
				Leader:   leader,
				Args:     args,
				Intermed: intermed,
			})
			return err
		}

		switch b {
		case NUL, DEL:
			continue
		case CAN, SUB:
			return nil
		case ESC:
			return p.readEsc()
		default:
			if b < C0 {
				p.readControl(b)
				continue top
			}
		}

		switch state {
		case LEADER:
			if b >= 0x3c && b <= 0x3f {
				leader = append(leader, b)
				continue
			} else {
				state = ARG
			}

			fallthrough
		case ARG:
			if b >= '0' && b <= '9' {
				if arg == -1 {
					arg = 0
				}

				arg *= 10
				arg += int(b - '0')
				continue top
			}

			if b == ':' {
				b = ';'
			}

			if b == ';' {
				args = append(args, arg)
				arg = -1
				continue top
			}

			if arg != -1 {
				args = append(args, arg)
			}

			state = INTERMED
			fallthrough
		case INTERMED:
			switch {
			case isIntermed(b):
				intermed = append(intermed, b)
				continue top
			case b == ESC:
				return nil
			case b >= 0x40 && b <= 0x7e:
				return p.handler.HandleEvent(&CSIEvent{
					Command:  b,
					Leader:   leader,
					Args:     args,
					Intermed: intermed,
				})
			}

			// Invalid in CSI, cancel it.
			return nil
		}
	}
}
