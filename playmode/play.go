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

package playmode

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hiscore"
	"github.com/jetsetilly/gopher2600/patch"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/setup"
)

type playmode struct {
	vcs     *hardware.VCS
	scr     gui.GUI
	intChan chan os.Signal
	guiChan chan gui.Event
}

// Play creates a 'playable' instance of the emulator.
//
// The cartload argument can be used to specify a recording to playback. The
// contents of the file specified in Filename field of the Loader instance will
// be checked. If it is a playback file then the playback codepath will be
// used.
func Play(tv *television.Television, scr gui.GUI, newRecording bool, cartload cartridgeloader.Loader, patchFile string, hiscoreServer bool, useSavekey bool) error {
	var recording string

	// if supplied cartridge name is actually a playback file then set
	// recording variable and dump cartridgeLoader information
	if recorder.IsPlaybackFile(cartload.Filename) {
		// do not allow this if a new recording has been requested
		if newRecording {
			return curated.Errorf("playmode: %v", "cannot make a new recording using a playback file")
		}

		recording = cartload.Filename

		// nullify cartload instance. we'll use the Loader instance in the
		// Playback instance
		cartload = cartridgeloader.Loader{}
	}

	// when allocation this channel will be used to halt emulation start until
	// a nil error is received
	var waitForEmulationStart chan error

	// OnLoaded function for specific cartridge formats
	cartload.OnLoaded = func(cart mapper.CartMapper) error {
		if _, ok := cart.(*supercharger.Supercharger); ok {
			return tv.Reset()
		} else if pr, ok := cart.(*plusrom.PlusROM); ok {
			if pr.Prefs.NewInstallation {
				waitForEmulationStart = make(chan error)

				fi := gui.PlusROMFirstInstallation{Finish: waitForEmulationStart, Cart: pr}
				err := scr.ReqFeature(gui.ReqPlusROMFirstInstallation, &fi)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	err = scr.ReqFeature(gui.ReqAddVCS, vcs)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	// replace player 1 port with savekey
	if useSavekey {
		err = vcs.RIOT.Ports.AttachPlayer(ports.Player1ID, savekey.NewSaveKey)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	}

	// note that we attach the cartridge in three different branches below,
	// depending on

	if newRecording {
		// new recording requested

		// create a unique filename
		n := time.Now()
		recording = fmt.Sprintf("recording_%s_%s",
			cartload.ShortName(), fmt.Sprintf("%04d%02d%02d_%02d%02d%02d",
				n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second()))

		// prepare new recording
		rec, err := recorder.NewRecorder(recording, vcs)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}

		// making sure we end the recording gracefully when we leave the function
		defer rec.End()

		// attach cartridge after recorder and transcribers have been
		// setup because we want to catch any setup events in the recording
		err = setup.AttachCartridge(vcs, cartload)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	} else if recording != "" {
		// not a new recording but a recording has been supplied. this is a
		// playback request

		plb, err := recorder.NewPlayback(recording)
		if err != nil {
			return err
		}

		// not using setup.AttachCartridge. if the playback was recorded with setup
		// changes the events will have been copied into the playback script and
		// will be applied that way
		err = vcs.AttachCartridge(plb.CartLoad)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}

		// attach playback to VCS
		err = plb.AttachToVCS(vcs)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	} else {
		// no new recording requested and no recording given. this is a 'normal'
		// launch of the emalator for regular play

		err = setup.AttachCartridge(vcs, cartload)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}

		// apply patch if requested. note that this will be in addition to any
		// patches applied during setup.AttachCartridge
		if patchFile != "" {
			_, err := patch.CartridgeMemory(vcs.Mem.Cart, patchFile)
			if err != nil {
				return curated.Errorf("playmode: %v", err)
			}
		}
	}

	pl := &playmode{
		vcs:     vcs,
		scr:     scr,
		intChan: make(chan os.Signal, 1),
		guiChan: make(chan gui.Event, 10),
	}

	// connect gui
	err = scr.ReqFeature(gui.ReqSetEventChan, pl.guiChan)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	// request television visibility
	err = scr.ReqFeature(gui.ReqSetVisibility, true)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	// if a waitForEmulationStart channel has been created then halt the
	// goroutine until we receive a non-error signal
	if waitForEmulationStart != nil {
		if err := <-waitForEmulationStart; err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	}

	// note that we are not setting the interrupt handler until
	// waitForEmulationStart has passed. this is because the handler for
	// os.Interrupt runs inside the emulation, which won't start until we've
	// successfully waited

	// we need to make sure we call the deferred function rec.End() even when
	// ctrl-c is pressed. redirect interrupt signal to an os.Signal channel
	signal.Notify(pl.intChan, os.Interrupt)

	// register game and begin game session
	var sess *hiscore.Session
	if hiscoreServer {
		sess, err = hiscore.NewSession()
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}

		err = sess.StartSession(cartload.ShortName(), vcs.Mem.Cart.Hash)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	}

	// note startime
	startTime := time.Now()

	// run and handle events
	err = vcs.Run(pl.eventHandler)

	// figure out amount of time played
	playTime := time.Since(startTime)

	// send to high score server
	if hiscoreServer {
		if err := sess.EndSession(playTime); err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	}

	if err != nil {
		if curated.Has(err, ports.PowerOff) {
			// PowerOff is okay and is to be expected. swallow the error
			// message and return as normal
			return nil
		}
		return curated.Errorf("playmode: %v", err)
	}

	return nil
}
