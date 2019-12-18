package playmode

import (
	"gopher2600/gui"
	"gopher2600/hardware"
	"gopher2600/hardware/riot/input"
)

// KeyboardEventHandler handles keypresses sent from a GUI. Returns true if
// key has been handled, false otherwise.
//
// For reasons of consistency, this handler is used by the debugger too.
func KeyboardEventHandler(keyEvent gui.EventDataKeyboard, tv gui.GUI, vcs *hardware.VCS) error {
	var err error

	if keyEvent.Down && keyEvent.Mod == gui.KeyModNone {
		switch keyEvent.Key {
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelectPress)
		case "F2":
			err = vcs.Panel.Handle(input.PanelResetPress)
		case "F3":
			err = vcs.Panel.Handle(input.PanelToggleColor)
		case "F4":
			err = vcs.Panel.Handle(input.PanelTogglePlayer0Pro)
		case "F5":
			err = vcs.Panel.Handle(input.PanelTogglePlayer1Pro)
		case "Left":
			err = vcs.Player0.Handle(input.Left)
		case "Right":
			err = vcs.Player0.Handle(input.Right)
		case "Up":
			err = vcs.Player0.Handle(input.Up)
		case "Down":
			err = vcs.Player0.Handle(input.Down)
		case "Space":
			err = vcs.Player0.Handle(input.Fire)
		}
	} else {
		switch keyEvent.Key {
		case "F1":
			err = vcs.Panel.Handle(input.PanelSelectRelease)
		case "F2":
			err = vcs.Panel.Handle(input.PanelResetRelease)
		case "Left":
			err = vcs.Player0.Handle(input.NoLeft)
		case "Right":
			err = vcs.Player0.Handle(input.NoRight)
		case "Up":
			err = vcs.Player0.Handle(input.NoUp)
		case "Down":
			err = vcs.Player0.Handle(input.NoDown)
		case "Space":
			err = vcs.Player0.Handle(input.NoFire)
		}
	}

	return err
}
