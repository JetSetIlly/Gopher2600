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
	"time"
)

// Play sets the emulation running - without any debugging features
func Play(cartridgeFile, tvType string, scaling float32, stable bool, recording string, newRecording bool) error {
	playtv, err := sdl.NewGUI(tvType, scaling, nil)
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

	// create default recording file name if no name has been supplied
	if newRecording && recording == "" {
		shortCartName := path.Base(cartridgeFile)
		shortCartName = strings.TrimSuffix(shortCartName, path.Ext(cartridgeFile))
		n := time.Now()
		timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
		recording = fmt.Sprintf("recording_%s_%s", shortCartName, timestamp)
	}

	var rec *recorder.Recorder
	var plb *recorder.Playback

	// note that we attach the cartridge in three different branches below - we
	// need to do this at different times depending on whether a new recording
	// or playback is taking place; or if it's just a regular playback

	if recording != "" {
		if newRecording {
			err = vcs.AttachCartridge(cartridgeFile)
			if err != nil {
				return err
			}

			rec, err = recorder.NewRecorder(recording, vcs)
			if err != nil {
				return fmt.Errorf("error preparing recording: %s", err)
			}

			defer func() {
				rec.End()
			}()

			vcs.Ports.Player0.AttachTranscriber(rec)
			vcs.Ports.Player1.AttachTranscriber(rec)
			vcs.Panel.AttachTranscriber(rec)
		} else {
			plb, err = recorder.NewPlayback(recording, vcs)
			if err != nil {
				return fmt.Errorf("error playing back recording: %s", err)
			}

			vcs.Ports.Player0.Attach(plb)
			vcs.Ports.Player1.Attach(plb)
			vcs.Panel.Attach(plb)

			if cartridgeFile != "" && cartridgeFile != plb.CartName {
				return fmt.Errorf("error playing back recording: cartridge name doesn't match the name in the recording")
			}

			// if no cartridge filename has been provided then use the one in
			// the playback file
			cartridgeFile = plb.CartName

			err = vcs.AttachCartridge(cartridgeFile)
			if err != nil {
				return err
			}
		}
	} else {
		err = vcs.AttachCartridge(cartridgeFile)
		if err != nil {
			return err
		}
	}

	// now that we've attached the cartridge check the hash against the
	// playback has (if it exists)
	if plb != nil && plb.CartHash != vcs.Mem.Cart.Hash {
		return fmt.Errorf("error playing back recording: cartridge hash doesn't match")
	}

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

	// run and handle gui events
	return vcs.Run(func() (bool, error) {
		select {
		case ev := <-guiChannel:
			switch ev.ID {
			case gui.EventWindowClose:
				return false, nil
			case gui.EventKeyboard:
				err = KeyboardEventHandler(ev.Data.(gui.EventDataKeyboard), playtv, vcs)
				return err == nil, err
			}
		default:
		}
		return true, nil
	})
}
