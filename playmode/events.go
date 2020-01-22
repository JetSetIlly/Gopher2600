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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package playmode

import (
	"gopher2600/gui"
	"gopher2600/hardware"
	"gopher2600/hardware/riot/input"
)

func (pl *playmode) guiEventHandler() (bool, error) {
	select {
	case <-pl.intChan:
		return false, nil
	case ev := <-pl.guiChan:
		switch ev := ev.(type) {
		case gui.EventWindowClose:
			return false, nil
		case gui.EventKeyboard:
			_, err := KeyboardEventHandler(ev, pl.vcs)
			return err == nil, err
		case gui.EventMouseButton:
			_, err := MouseButtonEventHandler(ev, pl.vcs)
			return err == nil, err
		}
	default:
	}

	return true, nil
}

// MouseButtonEventHandler handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
//
// For reasons of consistency, this handler is used by the debugger too
func MouseButtonEventHandler(ev gui.EventMouseButton, vcs *hardware.VCS) (bool, error) {
	var handled bool
	var err error

	switch ev.Button {
	case gui.MouseButtonLeft:
		if ev.Down {
			err = vcs.Player0.Handle(input.PaddleFire)
			handled = true
		} else {
			err = vcs.Player0.Handle(input.PaddleNoFire)
			handled = true
		}
	}

	return handled, err
}

// KeyboardEventHandler handles keypresses sent from a GUI. Returns true if
// key has been handled, false otherwise.
//
// For reasons of consistency, this handler is used by the debugger too.
func KeyboardEventHandler(ev gui.EventKeyboard, vcs *hardware.VCS) (bool, error) {
	var handled bool
	var err error

	if ev.Down && ev.Mod == gui.KeyModNone {
		switch ev.Key {
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelectPress)
			handled = true
		case "F2":
			err = vcs.Panel.Handle(input.PanelResetPress)
			handled = true
		case "F3":
			err = vcs.Panel.Handle(input.PanelToggleColor)
			handled = true
		case "F4":
			err = vcs.Panel.Handle(input.PanelTogglePlayer0Pro)
			handled = true
		case "F5":
			err = vcs.Panel.Handle(input.PanelTogglePlayer1Pro)
			handled = true
		case "Left":
			err = vcs.Player0.Handle(input.Left)
			handled = true
		case "Right":
			err = vcs.Player0.Handle(input.Right)
			handled = true
		case "Up":
			err = vcs.Player0.Handle(input.Up)
			handled = true
		case "Down":
			err = vcs.Player0.Handle(input.Down)
			handled = true
		case "Space":
			err = vcs.Player0.Handle(input.Fire)
			handled = true
		}
	} else {
		switch ev.Key {
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelectRelease)
			handled = true
		case "F2":
			err = vcs.Panel.Handle(input.PanelResetRelease)
			handled = true
		case "Left":
			err = vcs.Player0.Handle(input.NoLeft)
			handled = true
		case "Right":
			err = vcs.Player0.Handle(input.NoRight)
			handled = true
		case "Up":
			err = vcs.Player0.Handle(input.NoUp)
			handled = true
		case "Down":
			err = vcs.Player0.Handle(input.NoDown)
			handled = true
		case "Space":
			err = vcs.Player0.Handle(input.NoFire)
			handled = true
		}
	}

	return handled, err
}
