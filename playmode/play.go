package playmode

import (
	"fmt"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals/sticks"
	"sync/atomic"
)

// Play sets the emulation running - without any debugging features
func Play(cartridgeFile, tvMode string, scaling float32, stable bool) error {
	playtv, err := sdl.NewGUI(tvMode, scaling)
	if err != nil {
		return fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(playtv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	stk, err := sticks.NewSplaceStick(vcs.Panel)
	if err != nil {
		return err
	}
	vcs.Player0.Attach(stk)

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return err
	}

	// run while value of running variable is positive
	var running atomic.Value
	running.Store(0)

	// connect debugger to gui
	guiChannel := make(chan gui.Event, 2)
	playtv.SetEventChannel(guiChannel)

	// request television visibility
	request := gui.ReqSetVisibilityStable
	if !stable {
		request = gui.ReqSetVisibility
	}
	err = playtv.SetFeature(request, true)
	if err != nil {
		return fmt.Errorf("error preparing television: %s", err)
	}

	return vcs.Run(func() bool {
		select {
		case ev := <-guiChannel:
			switch ev.ID {
			case gui.EventWindowClose:
				return false
			case gui.EventKeyboard:
				KeyboardEventHandler(ev.Data.(gui.EventDataKeyboard), playtv, vcs)
			}
		default:
		}
		return true
	})
}
