package multiplex

import (
	"bytes"
	"io"
	"time"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/y0ssar1an/q"
)

type Event interface{}

type ControlEvent byte

type TextEvent []byte

const (
	Motion byte = 1
	Down   byte = 2
	Up     byte = 3

	Shift byte = 0x1
	Alt   byte = 0x2
	Ctrl  byte = 0x4
)

type MouseEvent struct {
	Op       byte
	Button   byte
	Modifier byte
	Row, Col int
}

type InputHandler interface {
	HandleInput(ev Event) error
}

type InputReader struct {
	r        io.Reader
	h        InputHandler
	newData  chan []byte
	escTimer *time.Timer

	mouseBuf bytes.Buffer
	curData  []byte
	pos      int
	lastByte byte
}

func NewInputReader(r io.Reader, h InputHandler) (*InputReader, error) {
	ip := &InputReader{
		r:        r,
		h:        h,
		newData:  make(chan []byte, 3),
		escTimer: time.NewTimer(0),
	}

	return ip, nil
}

func (i *InputReader) readInput() {
	for {
		buf := make([]byte, 128)
		n, err := i.r.Read(buf)
		if err != nil {
			close(i.newData)
			return
		}

		i.newData <- buf[:n]
	}
}

func (i *InputReader) waitDataOrTimeout(c <-chan time.Time) (bool, error) {
	select {
	case data := <-i.newData:
		if data == nil {
			return false, io.EOF
		}

		if len(i.curData) == i.pos {
			i.curData = data
		} else {
			i.curData = append(i.curData, data...)
		}

		return false, nil
	case <-c:
		return true, nil
	}
}

func (i *InputReader) peekByte() (byte, error) {
	if len(i.curData) == i.pos {
		data := <-i.newData
		if data == nil {
			return 0, io.EOF
		}

		i.curData = data
		i.pos = 0
	}

	b := i.curData[i.pos]

	return b, nil
}

func (i *InputReader) readByte() (byte, error) {
	b, err := i.peekByte()
	if err != nil {
		return 0, err
	}

	i.pos++

	return b, nil
}

func (i *InputReader) buffered() int {
	return len(i.curData) - i.pos
}

func (i *InputReader) unreadByte() error {
	if i.pos == 0 {
		return io.EOF
	}

	i.pos--

	return nil
}

func (i *InputReader) curRuneSize() (size int, err error) {
	if len(i.curData) == i.pos {
		return 0, nil
	}

	r, size := rune(i.curData[i.pos]), 1
	if r < utf8.RuneSelf {
		return 1, nil
	}

	for !utf8.FullRune(i.curData[i.pos:]) {
		data := <-i.newData
		if data == nil {
			return 0, io.EOF
		}

		i.curData = append(i.curData, data...)
	}

	_, size = utf8.DecodeRune(i.curData[i.pos:])

	return size, nil
}

func (i *InputReader) readRune() (out []byte, size int, err error) {
	if len(i.curData) == i.pos {
		data := <-i.newData
		if data == nil {
			return nil, 0, io.EOF
		}

		i.curData = data
	}

	r, size := rune(i.curData[i.pos]), 1
	if r < utf8.RuneSelf {
		b := i.curData[i.pos:i.pos:1]
		i.pos++
		return b, size, nil
	}

	for !utf8.FullRune(i.curData[i.pos:]) {
		data := <-i.newData
		if data == nil {
			return nil, 0, io.EOF
		}

		i.curData = append(i.curData, data...)
	}

	_, size = utf8.DecodeRune(i.curData[i.pos:])

	b := i.curData[i.pos : i.pos+size]
	i.pos += size
	return b, size, nil
}

const (
	NUL = 0x0
	DEL = 0x7f
	CAN = 0x18
	SUB = 0x1a
	ESC = 0x1b
	BEL = 0x7
	C0  = 0x20
)

func (i *InputReader) Drive() error {
	go i.readInput()

	for {
		b, err := i.readByte()
		if err != nil {
			return err
		}

		switch {
		case b == ESC:
			err := i.readEsc()
			if err != nil {
				return err
			}
		case b < C0 || b == DEL:
			err := i.h.HandleInput(ControlEvent(b))
			if err != nil {
				return err
			}
		default:
			i.unreadByte()
			err := i.readText()
			if err != nil {
				return err
			}
		}
	}
}

func (i *InputReader) readText() error {
	start := i.pos

	for len(i.curData) > i.pos {
		r := rune(i.curData[i.pos])
		if r < utf8.RuneSelf {
			if r < C0 || r == DEL {
				break
			}

			i.pos++
			continue
		}

		if utf8.FullRune(i.curData[i.pos:]) {
			_, size := utf8.DecodeRune(i.curData[i.pos:])
			i.pos += size

			spew.Dump(size)
			continue
		}

		if i.pos > start {
			err := i.h.HandleInput(TextEvent(i.curData[start:i.pos]))
			if err != nil {
				return nil
			}
		}

		for !utf8.FullRune(i.curData[i.pos:]) {
			q.Q("blocking")
			data := <-i.newData
			if data == nil {
				return io.EOF
			}

			i.curData = append(i.curData, data...)
		}
	}

	if i.pos > start {
		return i.h.HandleInput(TextEvent(i.curData[start:i.pos]))
	}

	return nil
}

/*
func (i *InputReader) olddreadText() error {
	var te TextEvent

	for {
		b, err := i.peekByte()
		if err != nil {
			return err
		}

		if b < C0 || b == DEL {
			if len(te) > 0 {
				return i.h.HandleInput(te)
			} else {
				return nil
			}
		}

		data, _, err := i.readRune()
		if err != nil {
			return err
		}

		te = append(te, r)

		if i.buffered() == 0 {
			return i.h.HandleInput(te)
		}
	}
}
*/

func (i *InputReader) readDigit() (int, error) {
	var x int

	for {
		b, err := i.readByte()
		if err != nil {
			return 0, err
		}

		if b < '0' || b > '9' {
			i.unreadByte()
			break
		}

		x *= 10
		x += int(b - '0')
	}

	return x, nil
}

func (i *InputReader) readEsc() error {
	if i.buffered() == 0 {
		i.escTimer.Reset(10 * time.Millisecond)
		timeout, err := i.waitDataOrTimeout(i.escTimer.C)
		if err != nil {
			return err
		}

		if timeout {
			return i.h.HandleInput(ControlEvent(ESC))
		}
	}

	b, err := i.readByte()
	if err != nil {
		return err
	}

	if b != '[' {
		return nil
	}

	b, err = i.readByte()
	if err != nil {
		return err
	}

	if b != '<' {
		return nil
	}

	var x, y, btn int

	btn, err = i.readDigit()
	if err != nil {
		return err
	}

	b, err = i.readByte()
	if err != nil {
		return err
	}

	if b != ';' {
		return nil
	}

	x, err = i.readDigit()
	if err != nil {
		return err
	}

	b, err = i.readByte()
	if err != nil {
		return err
	}

	if b != ';' {
		return nil
	}

	y, err = i.readDigit()
	if err != nil {
		return err
	}

	b, err = i.readByte()
	if err != nil {
		return err
	}

	if !(b == 'm' || b == 'M') {
		return nil
	}

	var motion bool

	if btn&0x20 != 0 {
		motion = true
		btn &^= 32
	}

	var me MouseEvent
	me.Col = x
	me.Row = y

	if motion {
		me.Op = Motion
	} else if b == 'm' {
		me.Op = Up
	} else {
		me.Op = Down
	}

	var btnBase int

	if btn&0x40 != 0 {
		btnBase = 4
		btn -= 64
	}

	me.Button = byte(btnBase) + byte(btn&0x3)
	me.Modifier = byte(btn >> 2)

	return i.h.HandleInput(me)
}
