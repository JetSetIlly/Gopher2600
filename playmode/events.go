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
			_, err := MouseButtonEventHandler(ev, pl.vcs, pl.scr)
			return err == nil, err
		case gui.EventMouseMotion:
			_, err := MouseMotionEventHandler(ev, pl.vcs)
			return err == nil, err
		}
	default:
	}

	return true, nil
}

// MouseMotionEventHandler handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func MouseMotionEventHandler(ev gui.EventMouseMotion, vcs *hardware.VCS) (bool, error) {
	return true, vcs.HandController0.Handle(input.PaddleSet, ev.X)
}

// MouseButtonEventHandler handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func MouseButtonEventHandler(ev gui.EventMouseButton, vcs *hardware.VCS, scr gui.GUI) (bool, error) {
	var handled bool
	var err error

	switch ev.Button {
	case gui.MouseButtonLeft:
		handled = true

		err = scr.SetFeature(gui.ReqCaptureMouse, true)
		if err != nil {
			return handled, err
		}

		if ev.Down {
			err = vcs.HandController0.Handle(input.PaddleFire, true)
		} else {
			err = vcs.HandController0.Handle(input.PaddleFire, false)
		}

	case gui.MouseButtonRight:
		err = scr.SetFeature(gui.ReqCaptureMouse, false)
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
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelect, true)
			handled = true
		case "F2":
			err = vcs.Panel.Handle(input.PanelReset, true)
			handled = true
		case "F3":
			err = vcs.Panel.Handle(input.PanelToggleColor, nil)
			handled = true
		case "F4":
			err = vcs.Panel.Handle(input.PanelTogglePlayer0Pro, nil)
			handled = true
		case "F5":
			err = vcs.Panel.Handle(input.PanelTogglePlayer1Pro, nil)
			handled = true
		case "Left":
			err = vcs.HandController0.Handle(input.Left, true)
			handled = true
		case "Right":
			err = vcs.HandController0.Handle(input.Right, true)
			handled = true
		case "Up":
			err = vcs.HandController0.Handle(input.Up, true)
			handled = true
		case "Down":
			err = vcs.HandController0.Handle(input.Down, true)
			handled = true
		case "Space":
			err = vcs.HandController0.Handle(input.Fire, true)
			handled = true
		case "1", "2", "3":
			err = vcs.HandController0.Handle(input.KeyboardDown, rune(ev.Key[0]))
			handled = true
		case "q":
			err = vcs.HandController0.Handle(input.KeyboardDown, '4')
			handled = true
		case "w":
			err = vcs.HandController0.Handle(input.KeyboardDown, '5')
			handled = true
		case "e":
			err = vcs.HandController0.Handle(input.KeyboardDown, '6')
			handled = true
		case "a":
			err = vcs.HandController0.Handle(input.KeyboardDown, '7')
			handled = true
		case "s":
			err = vcs.HandController0.Handle(input.KeyboardDown, '8')
			handled = true
		case "d":
			err = vcs.HandController0.Handle(input.KeyboardDown, '9')
			handled = true
		case "z":
			err = vcs.HandController0.Handle(input.KeyboardDown, '*')
			handled = true
		case "x":
			err = vcs.HandController0.Handle(input.KeyboardDown, '0')
			handled = true
		case "c":
			err = vcs.HandController0.Handle(input.KeyboardDown, '#')
			handled = true
		}
	} else {
		switch ev.Key {
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelect, false)
			handled = true
		case "F2":
			err = vcs.Panel.Handle(input.PanelReset, false)
			handled = true
		case "Left":
			err = vcs.HandController0.Handle(input.Left, false)
			handled = true
		case "Right":
			err = vcs.HandController0.Handle(input.Right, false)
			handled = true
		case "Up":
			err = vcs.HandController0.Handle(input.Up, false)
			handled = true
		case "Down":
			err = vcs.HandController0.Handle(input.Down, false)
			handled = true
		case "Space":
			err = vcs.HandController0.Handle(input.Fire, false)
			handled = true
		case "1", "2", "3", "q", "w", "e", "a", "s", "d", "z", "x", "c":
			err = vcs.HandController0.Handle(input.KeyboardUp, nil)
			handled = true
		}
	}

	return handled, err
}
