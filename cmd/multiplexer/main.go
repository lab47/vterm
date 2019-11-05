package main

import (
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/evanphx/vterm/multiplex"
)

func realMain() {
	var m multiplex.Multiplexer

	err := m.Init()
	if err != nil {
		log.Fatal(err)
	}

	defer m.Cleanup()

	cmd := exec.Command("zsh", "-l")

	w, err := m.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(w, os.Stdin)
}

func main() {
	realMain()
}
