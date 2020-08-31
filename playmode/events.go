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
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// MouseMotionEventHandler handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func MouseMotionEventHandler(ev gui.EventMouseMotion, vcs *hardware.VCS) (bool, error) {
	return true, vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.PaddleSet, ev.X)
}

// MouseButtonEventHandler handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func MouseButtonEventHandler(ev gui.EventMouseButton, vcs *hardware.VCS, scr gui.GUI) (bool, error) {
	var handled bool
	var err error

	switch ev.Button {
	case gui.MouseButtonLeft:
		if ev.Down {
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.PaddleFire, true)
		} else {
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.PaddleFire, false)
		}

		handled = true
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
		// panel
		case "F1":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSelect, true)
			handled = true
		case "F2":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelReset, true)
			handled = true
		case "F3":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelToggleColor, nil)
			handled = true
		case "F4":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelTogglePlayer0Pro, nil)
			handled = true
		case "F5":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelTogglePlayer1Pro, nil)
			handled = true

		// joystick
		case "Left":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Left, true)
			handled = true
		case "Right":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Right, true)
			handled = true
		case "Up":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Up, true)
			handled = true
		case "Down":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Down, true)
			handled = true
		case "Space":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Fire, true)
			handled = true

		// keypad (left player)
		case "1", "2", "3":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, rune(ev.Key[0]))
			handled = true
		case "Q":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '4')
			handled = true
		case "W":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '5')
			handled = true
		case "E":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '6')
			handled = true
		case "A":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '7')
			handled = true
		case "S":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '8')
			handled = true
		case "D":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '9')
			handled = true
		case "Z":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '*')
			handled = true
		case "X":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '0')
			handled = true
		case "C":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardDown, '#')
			handled = true

		// keypad (right player)
		case "4":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '1')
			handled = true
		case "5":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '2')
			handled = true
		case "6":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '3')
			handled = true
		case "R":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '4')
			handled = true
		case "T":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '5')
			handled = true
		case "Y":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '6')
			handled = true
		case "F":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '7')
			handled = true
		case "G":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '8')
			handled = true
		case "H":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '9')
			handled = true
		case "V":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '*')
			handled = true
		case "B":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '0')
			handled = true
		case "N":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardDown, '#')
			handled = true
		}
	} else {
		switch ev.Key {
		// panel
		case "F1":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSelect, false)
			handled = true
		case "F2":
			err = vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelReset, false)
			handled = true

		// josytick
		case "Left":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Left, false)
			handled = true
		case "Right":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Right, false)
			handled = true
		case "Up":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Up, false)
			handled = true
		case "Down":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Down, false)
			handled = true
		case "Space":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.Fire, false)
			handled = true

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player0ID, ports.KeyboardUp, nil)
			handled = true

		// keyboard (right player)
		case "4", "5", "6", "R", "T", "Y", "F", "G", "H", "V", "B", "N":
			err = vcs.RIOT.Ports.HandleEvent(ports.Player1ID, ports.KeyboardUp, nil)
			handled = true
		}
	}

	return handled, err
}

func (pl *playmode) guiEventHandler(ev gui.Event) (bool, error) {
	switch ev := ev.(type) {
	case gui.EventQuit:
		return false, nil
	case gui.EventKeyboard:
		_, err := KeyboardEventHandler(ev, pl.vcs)
		return err == nil, err
	case gui.EventMouseButton:
		_, err := MouseButtonEventHandler(ev, pl.vcs, pl.scr)
		return err == nil, err
	case gui.EventMouseMotion:
		_, err := MouseMotionEventHandler(ev, pl.vcs)
		return err == nil, err
	}

	return true, nil
}

func (pl *playmode) eventHandler() (bool, error) {
	select {
	case <-pl.intChan:
		return false, nil
	case ev := <-pl.guiChan:
		return pl.guiEventHandler(ev)
	default:
	}

	return true, nil
}
