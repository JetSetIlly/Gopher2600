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

package playmode

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/userinput"
)

// sentinal error returned when GUI detects a quit event.
const quitEvent = "user input quit event"

func (pl *playmode) doRewind(amount int) {
	pl.setState(emulation.Rewinding)
	coords := pl.vcs.TV.GetCoords()
	pl.rewind.GotoFrame(coords.Frame + amount)
	pl.setState(emulation.Paused)
}

func (pl *playmode) userInputHandler(ev userinput.Event) error {
	// handle some events specifically for rewinding.
	rewindKeys := false

	switch ev := ev.(type) {
	case userinput.EventMouseWheel:
		pl.doRewind(int(ev.Delta))
		rewindKeys = true

	case userinput.EventKeyboard:
		if ev.Down {
			switch ev.Key {
			case "Left":
				if ev.Mod == userinput.KeyModShift {
					pl.doRewind(-1)
					rewindKeys = true
				}
			case "Right":
				if ev.Mod == userinput.KeyModShift {
					pl.doRewind(1)
					rewindKeys = true
				}
			}
		} else if ev.Mod != userinput.KeyModNone {
			rewindKeys = true
		}
	}

	// the user event has triggered a rewind so return immediately
	if rewindKeys {
		return nil
	}

	quit, err := pl.controllers.HandleUserInput(ev, pl.vcs.RIOT.Ports)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	if quit {
		return curated.Errorf(quitEvent)
	}

	// resume emulation if the last event was recognised as a controller input
	// from the keyboard
	if pl.state == emulation.Paused && pl.controllers.LastKeyHandled {
		pl.setState(emulation.Running)
	}

	return nil
}

func (pl *playmode) eventHandler() (emulation.State, error) {
	select {
	case <-pl.intChan:
		return emulation.Ending, curated.Errorf(terminal.UserInterrupt)

	case ev := <-pl.userinput:
		return pl.state, pl.userInputHandler(ev)

	case ev := <-pl.rawEvents:
		ev()

	default:
	}

	return pl.state, nil
}
