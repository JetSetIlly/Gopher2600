// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package playmode

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware"
	"gopher2600/patch"
	"gopher2600/recorder"
	"gopher2600/setup"
	"gopher2600/television"
	"os"
	"os/signal"
	"time"
)

// Play is a quick of setting up a playable instance of the emulator.
func Play(tv television.Television, scr gui.GUI, showOnStable bool, newRecording bool, cartload cartridgeloader.Loader, patchFile string) error {
	var transcript string

	// if supplied cartridge name is actually a playback file then set
	// transcript and dump cartridgeLoader information
	if recorder.IsPlaybackFile(cartload.Filename) {

		// do not allow this if a new recording has been requested
		if newRecording {
			return errors.New(errors.PlayError, "cannot make a new recording using a playback file")
		}

		transcript = cartload.Filename
		cartload = cartridgeloader.Loader{}
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return errors.New(errors.PlayError, err)
	}

	// note that we attach the cartridge in three different branches below,
	// depending on

	if newRecording {
		// new recording requested

		// create a unique filename
		n := time.Now()
		transcript = fmt.Sprintf("recording_%s_%s",
			cartload.ShortName(), fmt.Sprintf("%04d%02d%02d_%02d%02d%02d",
				n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second()))

		// prepare new recording
		rec, err := recorder.NewRecorder(transcript, vcs)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

		// making sure we end the recording gracefully when we leave the function
		defer func() {
			rec.End()
		}()

		// attach cartridge after recorder and transcribers have been
		// setup because we want to catch any setup events in the recording
		err = setup.AttachCartridge(vcs, cartload)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

	} else if transcript != "" {
		// not a new recording but a transcript has been supplied. this is a
		// playback request

		plb, err := recorder.NewPlayback(transcript)
		if err != nil {
			return err
		}

		if cartload.Filename != "" && cartload.Filename != plb.CartLoad.Filename {
			return errors.New(errors.PlayError, "cartridge doesn't match name in the playback recording")
		}

		// not using setup.AttachCartridge. if the playback was recorded with setup
		// changes the events will have been copied into the playback script and
		// will be applied that way
		err = vcs.AttachCartridge(plb.CartLoad)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

		// the following will fail if the recording was made with different tv
		// parameters. currently, the only parameter is the tv spec (ie. AUTO,
		// NTSC or PAL) but we may need to worry about this if we ever add
		// another television implementation.
		err = plb.AttachToVCS(vcs)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

	} else {
		// no new recording requested and no transcript given. this is a 'normal'
		// launch of the emalator for regular play

		err = setup.AttachCartridge(vcs, cartload)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

		// apply patch if requested. note that this will be in addition to any
		// patches applied during setup.AttachCartridge
		if patchFile != "" {
			_, err := patch.CartridgeMemory(vcs.Mem.Cart, patchFile)
			if err != nil {
				return errors.New(errors.PlayError, err)
			}
		}
	}

	// connect gui
	guiChannel := make(chan gui.Event, 2)
	scr.SetEventChannel(guiChannel)

	// request television visibility
	request := gui.ReqSetVisibility
	if showOnStable {
		request = gui.ReqSetVisibleOnStable
	}
	err = scr.SetFeature(request, true)
	if err != nil {
		return errors.New(errors.PlayError, err)
	}

	// we need to make sure we call the deferred function rec.End() even when
	// ctrl-c is pressed. redirect interrupt signal to an os.Signal channel
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)

	// run and handle gui events
	err = vcs.Run(func() (bool, error) {
		select {
		case <-intChan:
			return false, nil
		case ev := <-guiChannel:
			switch ev.ID {
			case gui.EventWindowClose:
				return false, nil
			case gui.EventKeyboard:
				err = KeyboardEventHandler(ev.Data.(gui.EventDataKeyboard), scr, vcs)
				return err == nil, err
			}
		default:
		}
		return true, nil
	})

	if err != nil {
		if errors.Is(err, errors.PowerOff) {
			return nil
		}
		return errors.New(errors.PlayError, err)
	}

	return nil
}
