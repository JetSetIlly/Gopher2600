package playmode

import (
	"gopher2600/gui"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
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
			err = vcs.Panel.Handle(peripherals.PanelSelectPress)
		case "F2":
			err = vcs.Panel.Handle(peripherals.PanelResetPress)
		case "F3":
			err = vcs.Panel.Handle(peripherals.PanelToggleColor)
		case "F4":
			err = vcs.Panel.Handle(peripherals.PanelTogglePlayer0Pro)
		case "F5":
			err = vcs.Panel.Handle(peripherals.PanelTogglePlayer1Pro)
		case "Left":
			err = vcs.Ports.Player0.Handle(peripherals.Left)
		case "Right":
			err = vcs.Ports.Player0.Handle(peripherals.Right)
		case "Up":
			err = vcs.Ports.Player0.Handle(peripherals.Up)
		case "Down":
			err = vcs.Ports.Player0.Handle(peripherals.Down)
		case "Space":
			err = vcs.Ports.Player0.Handle(peripherals.Fire)
		}
	} else {
		switch keyEvent.Key {
		case "F1":
			err = vcs.Panel.Handle(peripherals.PanelSelectRelease)
		case "F2":
			err = vcs.Panel.Handle(peripherals.PanelResetRelease)
		case "Left":
			err = vcs.Ports.Player0.Handle(peripherals.NoLeft)
		case "Right":
			err = vcs.Ports.Player0.Handle(peripherals.NoRight)
		case "Up":
			err = vcs.Ports.Player0.Handle(peripherals.NoUp)
		case "Down":
			err = vcs.Ports.Player0.Handle(peripherals.NoDown)
		case "Space":
			err = vcs.Ports.Player0.Handle(peripherals.NoFire)
		}
	}

	return err
}
