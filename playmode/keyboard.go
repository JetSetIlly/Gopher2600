package playmode

import (
	"gopher2600/gui"
	"gopher2600/hardware"
	"time"
)

// TODO: the sleep method for F1 and F2 sucks, better to have separate key
// up/down callbacks

// KeyboardCallback handles keypresses for play/run mode
func KeyboardCallback(tv gui.GUI, vcs *hardware.VCS) string {
	key, _ := tv.GetMetaState(gui.ReqLastKeyboard)
	switch key {
	case "F1":
		vcs.Panel.SetGameSelect(true)
		go func() {
			time.Sleep(10 * time.Millisecond)
			vcs.Panel.SetGameSelect(false)
		}()
	case "F2":
		vcs.Panel.SetGameReset(true)
		go func() {
			time.Sleep(10 * time.Millisecond)
			vcs.Panel.SetGameReset(false)
		}()
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
		return key.(string)
	}

	return ""
}
