package state

import (
	"github.com/evanphx/vterm/parser"
	"github.com/y0ssar1an/q"
)

func (s *State) mouseEvent(ev *parser.CSIEvent) error {
	q.Q(ev)
	return nil
}
