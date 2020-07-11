package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/lab47/vterm/multiplex"
	"github.com/pkg/profile"
)

func realMain() {
	var m multiplex.Multiplexer

	err := m.Init()
	if err != nil {
		log.Fatal(err)
	}

	defer m.Cleanup()

	cmd := exec.Command("sh")
	cmd.Env = append(os.Environ(), "PS1=> ")

	err = m.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("PROFILE") != "" {
		defer profile.Start(profile.ProfilePath("."), profile.MemProfile, profile.MemProfileRate(1)).Stop()
	}

	m.InputData(os.Stdin)
}

func main() {
	realMain()
}
