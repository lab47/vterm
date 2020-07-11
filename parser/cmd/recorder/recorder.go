package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/davecgh/go-spew/spew"
	"github.com/lab47/vterm/parser"
	"golang.org/x/crypto/ssh/terminal"
)

type recordHandler struct {
	f *os.File
}

func (r *recordHandler) HandleEvent(ev parser.Event) error {
	spew.Fdump(r.f, ev)
	return nil
}

func record() error {
	f, err := os.Create(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	r, w := io.Pipe()

	parser, err := parser.NewParser(r, &recordHandler{f})
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		parser.Drive()
	}()

	defer wg.Wait()

	defer w.Close()

	ts, err := terminal.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal(err)
	}

	defer terminal.Restore(int(os.Stdout.Fd()), ts)

	cmd := exec.Command(os.Args[2], os.Args[3:]...)

	out, err := pty.Start(cmd)
	if err != nil {
		log.Fatal(err)
	}

	go io.Copy(out, os.Stdin)
	io.Copy(io.MultiWriter(os.Stdout, w), out)

	return cmd.Wait()
}

func main() {
	record()
}
