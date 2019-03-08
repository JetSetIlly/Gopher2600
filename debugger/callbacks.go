package debugger

import (
	"fmt"
	"gopher2600/debugger/ui"
	"gopher2600/television"
)

func (dbg *Debugger) setupTVCallbacks() error {
	var err error

	// add break on right mouse button
	err = dbg.vcs.TV.RegisterCallback(television.ReqOnMouseButtonRight, dbg.dbgChannel, func() {
		// this callback function may be running inside a different goroutine
		// so care must be taken not to cause a deadlock
		hp, _ := dbg.vcs.TV.GetMetaState(television.ReqLastMouseHorizPos)
		sl, _ := dbg.vcs.TV.GetMetaState(television.ReqLastMouseScanline)

		_, err := dbg.parseCommand(fmt.Sprintf("%s sl %s & hp %s", KeywordBreak, sl, hp))
		if err == nil {
			dbg.print(ui.Feedback, "mouse break on sl->%s and hp->%s", sl, hp)
		} else {
			dbg.print(ui.Error, "%s", err)
		}
	})
	if err != nil {
		return err
	}

	// respond to keyboard
	err = dbg.vcs.TV.RegisterCallback(television.ReqOnKeyboard, dbg.dbgChannel, func() {
		key, _ := dbg.vcs.TV.GetMetaState(television.ReqLastKeyboard)
		switch key {
		case "`":
			// back-tick: toggle masking
			err = dbg.vcs.TV.SetFeature(television.ReqToggleMasking)
		case "1":
			// toggle debugging colours
			err = dbg.vcs.TV.SetFeature(television.ReqToggleAltColors)
		case "2":
			// toggle metasignals overlay
			err = dbg.vcs.TV.SetFeature(television.ReqToggleShowSystemState)
		case "=":
			// toggle debugging colours
			err = dbg.vcs.TV.SetFeature(television.ReqIncScale)
		case "-":
			// toggle debugging colours
			err = dbg.vcs.TV.SetFeature(television.ReqDecScale)
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
