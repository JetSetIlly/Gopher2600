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
	"os"
	"os/signal"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hiscore"
	"github.com/jetsetilly/gopher2600/patch"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/television"
)

type playmode struct {
	vcs     *hardware.VCS
	scr     gui.GUI
	intChan chan os.Signal
	guiChan chan gui.Event
}

// Play is a quick of setting up a playable instance of the emulator.
func Play(tv television.Television, scr gui.GUI, newRecording bool, cartload cartridgeloader.Loader, patchFile string, hiscoreServer bool) error {
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

	pl := &playmode{
		vcs:     vcs,
		scr:     scr,
		intChan: make(chan os.Signal, 1),
		guiChan: make(chan gui.Event, 2),
	}

	// connect gui
	err = scr.ReqFeature(gui.ReqSetEventChan, pl.guiChan)
	if err != nil {
		return errors.New(errors.PlayError, err)
	}

	// request television visibility
	err = scr.ReqFeature(gui.ReqSetVisibility, true)
	if err != nil {
		return errors.New(errors.PlayError, err)
	}

	// we need to make sure we call the deferred function rec.End() even when
	// ctrl-c is pressed. redirect interrupt signal to an os.Signal channel
	signal.Notify(pl.intChan, os.Interrupt)

	// register game and begin game session
	var sess *hiscore.Session
	if hiscoreServer {
		sess, err = hiscore.NewSession()
		if err != nil {
			return errors.New(errors.PlayError, err)
		}

		err = sess.StartSession(cartload.ShortName(), vcs.Mem.Cart.Hash)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}
	}

	// note startime
	startTime := time.Now()

	// run and handle events
	err = vcs.Run(pl.eventHandler)

	// figure out amount of time played
	playTime := time.Now().Sub(startTime)

	// send to high score server
	if hiscoreServer {
		if err := sess.EndSession(playTime); err != nil {
			return errors.New(errors.PlayError, err)
		}
	}

	if err != nil {
		if errors.Is(err, errors.PowerOff) {
			return nil
		}
		return errors.New(errors.PlayError, err)
	}

	return nil
}
