package playmode

import (
	"fmt"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals/sticks"
	"gopher2600/recorder"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

// Play sets the emulation running - without any debugging features
func Play(cartridgeFile, tvMode string, scaling float32, stable bool, recording string, newRecording bool) error {
	playtv, err := sdl.NewGUI(tvMode, scaling)
	if err != nil {
		return fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(playtv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	stk, err := sticks.NewSplaceStick()
	if err != nil {
		return err
	}
	vcs.Ports.Player0.Attach(stk)

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return err
	}

	// create default recording file name if no name has been supplied
	if newRecording && recording == "" {
		shortCartName := path.Base(cartridgeFile)
		shortCartName = strings.TrimSuffix(shortCartName, path.Ext(cartridgeFile))
		n := time.Now()
		timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
		recording = fmt.Sprintf("recording_%s_%s", shortCartName, timestamp)
	}

	if recording != "" {
		if newRecording {
			recording, err := recorder.NewRecorder(recording, vcs)
			if err != nil {
				return fmt.Errorf("error preparing VCS: %s", err)
			}

			defer func() {
				recording.End()
			}()

			vcs.Ports.Player0.AttachRecorder(recording)
			vcs.Ports.Player1.AttachRecorder(recording)
		} else {
			recording, err := recorder.NewPlayback(recording, vcs)
			if err != nil {
				return fmt.Errorf("error preparing VCS: %s", err)
			}

			vcs.Ports.Player0.Attach(recording)
			vcs.Ports.Player1.Attach(recording)
		}
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
