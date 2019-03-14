package debugger

import (
	"fmt"
	"gopher2600/debugger/ui"
	"gopher2600/gui"
)

func (dbg *Debugger) setupTVCallbacks() error {
	var err error

	// add break on right mouse button
	err = dbg.tv.RegisterCallback(gui.ReqOnMouseButtonRight, dbg.interruptChannel, func() {
		// this callback function may be running inside a different goroutine
		// so care must be taken not to cause a deadlock
		hp, _ := dbg.tv.GetMetaState(gui.ReqLastMouseHorizPos)
		sl, _ := dbg.tv.GetMetaState(gui.ReqLastMouseScanline)

		_, err := dbg.parseCommand(fmt.Sprintf("%s sl %d & hp %d", KeywordBreak, sl, hp))
		if err == nil {
			dbg.print(ui.Feedback, "mouse break on sl->%d and hp->%d", sl, hp)
		} else {
			dbg.print(ui.Error, "%s", err)
		}
	})
	if err != nil {
		return err
	}

	// respond to keyboard
	err = dbg.tv.RegisterCallback(gui.ReqOnKeyboard, dbg.interruptChannel, func() {
		key, _ := dbg.tv.GetMetaState(gui.ReqLastKeyboard)
		switch key {
		case "`":
			// back-tick: toggle masking
			err = dbg.tv.SetFeature(gui.ReqToggleMasking)
		case "1":
			// toggle debugging colours
			err = dbg.tv.SetFeature(gui.ReqToggleAltColors)
		case "2":
			// toggle metasignals overlay
			err = dbg.tv.SetFeature(gui.ReqToggleShowSystemState)
		case "=":
			// toggle debugging colours
			err = dbg.tv.SetFeature(gui.ReqIncScale)
		case "-":
			// toggle debugging colours
			err = dbg.tv.SetFeature(gui.ReqDecScale)
		}
		if err != nil {
			dbg.print(ui.Error, "%s", err)
		}
	})
	if err != nil {
		return err
	}

	return nil
}
