package main

import (
	"log"
	"os/exec"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"github.com/lab47/vterm/widget"
)

func main() {
	cmd := exec.Command("zsh", "-l")

	widget, err := widget.NewWidget(cmd)
	if err != nil {
		log.Fatal(err)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}

	screen.ShowCursor(0, 0)

	var app views.Application
	app.SetScreen(screen)
	app.SetRootWidget(widget)

	widget.Quit = app.Quit
	widget.Update = app.Update

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
