package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/evanphx/vterm/multiplex"
	"github.com/pkg/profile"
)

func realMain() {
	var m multiplex.Multiplexer

	err := m.Init()
	if err != nil {
		log.Fatal(err)
	}

	defer m.Cleanup()

	cmd := exec.Command("zsh", "-l")

	err = m.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("PROFILE") != "" {
		defer profile.Start(profile.ProfilePath(".")).Stop()
	}

	m.InputData(os.Stdin)
}

func main() {
	realMain()
}
