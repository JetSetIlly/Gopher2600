package debugger

import (
	"fmt"
	"gopher2600/debugger/console"
	"gopher2600/gui"
	"gopher2600/playmode"
)

func (dbg *Debugger) setupGUICallbacks() error {
	var err error

	// add break on right mouse button
	err = dbg.gui.RegisterCallback(gui.ReqOnMouseButtonRight, dbg.interruptChannel, func() {
		// this callback function may be running inside a different goroutine
		// so care must be taken not to cause a deadlock
		hp, _ := dbg.gui.GetMetaState(gui.ReqLastMouseHorizPos)
		sl, _ := dbg.gui.GetMetaState(gui.ReqLastMouseScanline)

		_, err := dbg.parseInput(fmt.Sprintf("%s sl %d & hp %d", cmdBreak, sl, hp), false)
		if err == nil {
			dbg.print(console.Feedback, "mouse break on sl->%d and hp->%d", sl, hp)
		} else {
			dbg.print(console.Error, "%s", err)
		}
	})
	if err != nil {
		return err
	}

	// respond to keyboard - use playmode keyboard callback functions in the
	// first instance and catch unhandled keys to see if we can use them for
	// debugging mode
	err = dbg.gui.RegisterCallback(gui.ReqOnKeyboard, dbg.interruptChannel, func() {
		switch playmode.KeyboardCallback(dbg.gui, dbg.vcs) {
		case "`":
			// back-tick: toggle masking
			err = dbg.gui.SetFeature(gui.ReqToggleMasking)
		case "1":
			// toggle debugging colours
			err = dbg.gui.SetFeature(gui.ReqToggleAltColors)
		case "2":
			// toggle metasignals overlay
			err = dbg.gui.SetFeature(gui.ReqToggleShowSystemState)
		case "=":
			// equal sign is the same as plus, for convenience
			fallthrough
		case "+":
			// increase scaling
			err = dbg.gui.SetFeature(gui.ReqIncScale)
		case "-":
			// decrease window scanling
			err = dbg.gui.SetFeature(gui.ReqDecScale)
		}
		if err != nil {
			dbg.print(console.Error, "%s", err)
		}
	})
	if err != nil {
		return err
	}

	return nil
}
