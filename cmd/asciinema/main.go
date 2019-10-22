package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/evanphx/vterm/parser"
	"github.com/evanphx/vterm/state"
)

type Frame struct {
	Delay float64
	Data  []byte
}

type Header struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (f *Frame) MarshalJSON() ([]byte, error) {
	s, _ := json.Marshal(string(f.Data))
	json := fmt.Sprintf(`[%.6f, %s]`, f.Delay, s)
	return []byte(json), nil
}

func (f *Frame) UnmarshalJSON(data []byte) error {
	var x interface{}

	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	f.Delay = x.([]interface{})[0].(float64)

	s := []byte(x.([]interface{})[2].(string))
	b := make([]byte, len(s))
	copy(b, s)
	f.Data = b

	return nil
}

type debugOutput struct {
}

func (d *debugOutput) dump(kind string, args ...interface{}) {
	fmt.Printf("%s\t ", kind)

	for _, a := range args {
		if str, ok := a.(interface{ String() string }); ok {
			fmt.Printf("• %s(%#v) ", str.String(), a)
		} else {
			fmt.Printf("• %#v ", a)
		}
	}

	fmt.Println()
}

func (d *debugOutput) SetCell(pos state.Pos, val state.CellRune) error {
	d.dump("set-cell", pos, val, string(val.Rune))
	return nil
}

func (d *debugOutput) AppendCell(pos state.Pos, r rune) error {
	d.dump("append-cell", pos, r)
	return nil
}

func (d *debugOutput) ClearRect(r state.Rect) error {
	d.dump("clear-rect", r)
	return nil
}

func (d *debugOutput) ScrollRect(s state.ScrollRect) error {
	d.dump("scroll-rect", s)
	return nil
}

func (d *debugOutput) Output(data []byte) error {
	d.dump("output", data)
	return nil
}

func (d *debugOutput) SetTermProp(prop state.TermAttr, val interface{}) error {
	d.dump("set-term-prop", prop, val)
	return nil
}

func (d *debugOutput) SetPenProp(prop state.PenAttr, val interface{}) error {
	d.dump("set-pen-prop", prop, val)
	return nil
}

func (d *debugOutput) StringEvent(kind string, data []byte) error {
	d.dump("string-event", kind, string(data))
	return nil
}

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(f)

	var info Header

	err = dec.Decode(&info)
	if err != nil {
		log.Fatal(err)
	}

	var do debugOutput

	st, err := state.NewState(info.Height, info.Width, &do)
	if err != nil {
		log.Fatal(err)
	}

	r, w := io.Pipe()

	parser, err := parser.NewParser(r, st)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := parser.Drive()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		var frame Frame

		err = dec.Decode(&frame)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Fatal(err)
		}

		// time.Sleep(time.Duration(frame.Delay * float64(time.Second)))

		w.Write([]byte(frame.Data))
	}
}
