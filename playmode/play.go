package playmode

import (
	"fmt"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"sync/atomic"
)

// Play sets the emulation running - without any debugging features
func Play(cartridgeFile, tvMode string, scaling float32, stable bool) error {
	tv, err := sdl.NewGUI(tvMode, scaling)
	if err != nil {
		return fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return err
	}

	// run while value of running variable is positive
	var running atomic.Value
	running.Store(0)

	// register quit function
	err = tv.RegisterCallback(gui.ReqOnWindowClose, nil, func() {
		running.Store(-1)
	})
	if err != nil {
		return err
	}

	// register quit function
	err = tv.RegisterCallback(gui.ReqOnKeyboard, nil, func() {
		_ = KeyboardCallback(tv, vcs)
	})
	if err != nil {
		return err
	}

	if stable {
		err = tv.SetFeature(gui.ReqSetVisibilityStable, true)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	} else {
		err = tv.SetFeature(gui.ReqSetVisibility, true)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	}

	return vcs.Run(&running)
}
