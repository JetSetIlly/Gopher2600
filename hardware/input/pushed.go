// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package input

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// AllowPushedEvents or not. Should not be allowed unless absolutely necessary.
func (inp *Input) AllowPushedEvents(allow bool) {
	if allow {
		inp.tv.AddFrameTrigger(inp)
	} else {
		inp.tv.RemoveFrameTrigger(inp)
	}
}

// PushEvent pushes an InputEvent onto the queue. Will drop the event and
// return an error if queue is full.
func (inp *Input) PushEvent(ev ports.InputEvent) error {
	select {
	case inp.pushed <- ev:
	default:
		return fmt.Errorf("ports: pushed event queue is full: input dropped")
	}
	return nil
}

func (inp *Input) handlePushed() error {
	done := false
	for !done {
		select {
		case ev := <-inp.pushed:
			_, err := inp.ports.HandleInputEvent(ev)
			if err != nil {
				return err
			}
		default:
			done = true
		}
	}
	return nil
}
