package debugger

import (
	"fmt"
	"gopher2600/debugger/console"
	"gopher2600/gui"
	"gopher2600/playmode"
)

func (dbg *Debugger) guiEventHandler(event gui.Event) error {
	var err error

	switch event.ID {
	case gui.EventWindowClose:
	case gui.EventKeyboard:
		data := event.Data.(gui.EventDataKeyboard)

		// check playmode key presses first
		err = playmode.KeyboardEventHandler(data, dbg.gui, dbg.vcs)
		if err != nil {
			break // switch event.ID
		}

		if data.Down == true {
			switch data.Key {
			case "`":
				// back-tick: toggle masking
				err = dbg.gui.SetFeature(gui.ReqToggleMasking)

			case "1":
				// toggle debugging colours
				err = dbg.gui.SetFeature(gui.ReqToggleAltColors)
			case "2":
				// toggle metasignals overlay
				err = dbg.gui.SetFeature(gui.ReqToggleShowMetaPixels)

			case "=":
				fallthrough // equal sign is the same as plus, for convenience
			case "+":
				// increase scaling
				err = dbg.gui.SetFeature(gui.ReqIncScale)
			case "-":
				// decrease window scanling
				err = dbg.gui.SetFeature(gui.ReqDecScale)
			}
		}

	case gui.EventMouseRight:
		data := event.Data.(gui.EventDataMouse)
		_, err = dbg.parseInput(fmt.Sprintf("%s sl %d & hp %d", cmdBreak, data.Scanline, data.HorizPos), false, false)
		if err == nil {
			dbg.print(console.Feedback, "mouse break on sl->%d and hp->%d", data.Scanline, data.HorizPos)
		}
	}

	return err
}

func (dbg *Debugger) checkInterruptsAndEvents() {
	// check interrupt channel and run any functions we find in there
	select {
	case <-dbg.interruptChannel:
		if dbg.runUntilHalt {
			// stop emulation at the next step
			dbg.runUntilHalt = false
		} else {
			// runUntilHalt is false which means that the emulation is
			// not running. at this point, an input loop is probably
			// running.
			//
			// note that ctrl-c signals do not always reach
			// this far into the program.  for instance, the colorterm
			// implementation of UserRead() puts the terminal into raw
			// mode and so must handle ctrl-c events differently.

			if dbg.recording.IsRecording() {
				dbg.recording.End()
			} else {
				dbg.running = false
			}
		}
	case ev := <-dbg.guiChannel:
		dbg.guiEventHandler(ev)
	default:
		// pro-tip: default case required otherwise the select will block
		// indefinately.
	}
}
