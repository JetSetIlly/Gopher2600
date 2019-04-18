package playmode

import (
	"gopher2600/gui"
	"gopher2600/hardware"
)

// KeyboardEventHandler handles keypresses for play/run mode
// returns true if key has been handled, false if not
//
// (public declaration because we want to use this in the debugger as well)
func KeyboardEventHandler(keyEvent gui.EventDataKeyboard, tv gui.GUI, vcs *hardware.VCS) bool {
	if keyEvent.Down {
		switch keyEvent.Key {
		case "F1":
			vcs.Panel.PressSelect()
		case "F2":
			vcs.Panel.PressReset()
		case "F3":
			vcs.Panel.ToggleColour()
		case "F4":
			vcs.Panel.TogglePlayer0Pro()
		case "F5":
			vcs.Panel.TogglePlayer1Pro()
		default:
			return false
		}
	} else {
		switch keyEvent.Key {
		case "F1":
			vcs.Panel.ReleaseSelect()
		case "F2":
			vcs.Panel.ReleaseReset()
		default:
			return false
		}
	}

	return true
}
