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
			vcs.Panel.SetGameSelect(true)
		case "F2":
			vcs.Panel.SetGameReset(true)
		case "F3":
			vcs.Panel.SetColor(true)
		case "F4":
			vcs.Panel.SetColor(false)
		case "F5":
			vcs.Panel.SetPlayer0Pro(true)
		case "F6":
			vcs.Panel.SetPlayer0Pro(false)
		case "F7":
			vcs.Panel.SetPlayer1Pro(true)
		case "F8":
			vcs.Panel.SetPlayer1Pro(false)
		default:
			return false
		}
	} else {
		switch keyEvent.Key {
		case "F1":
			vcs.Panel.SetGameSelect(false)
		case "F2":
			vcs.Panel.SetGameReset(false)
		default:
			return false
		}
	}

	return true
}
