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
	"os"
	"os/signal"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hiscore"
	"github.com/jetsetilly/gopher2600/patch"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/userinput"
)

type playmode struct {
	state emulation.State

	vcs         *hardware.VCS
	scr         gui.GUI
	controllers userinput.Controllers

	intChan   chan os.Signal
	userinput chan userinput.Event
	rawEvents chan func()
}

// VCS implements the emulation.Emulation interface.
func (pl *playmode) VCS() emulation.VCS {
	return pl.vcs
}

// Debugger implements the emulation.Emulation interface.
func (pl *playmode) Debugger() emulation.Debugger {
	return nil
}

// UserInput implements the emulation.Emulation interface.
func (pl *playmode) UserInput() chan userinput.Event {
	return pl.userinput
}

// State implements the emulation.Emulation interface.
func (pl *playmode) State() emulation.State {
	return pl.state
}

// Pause implements the emulation.Emulation interface.
func (pl *playmode) Pause(set bool) {
	if set {
		pl.state = emulation.Paused
	} else {
		pl.state = emulation.Running
	}
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
	if err := recorder.IsPlaybackFile(cartload.Filename); err == nil {
		// do not allow this if a new recording has been requested
		if newRecording {
			return curated.Errorf("playmode: cannot make a new recording using a playback file")
		}

		recording = cartload.Filename

		// nullify cartload instance. we'll use the Loader instance in the
		// Playback instance
		cartload = cartridgeloader.Loader{}
	} else if !curated.Is(err, recorder.NotAPlaybackFile) {
		return curated.Errorf("playmode: %v", err)
	}

	// when allocation this channel will be used to halt emulation start until
	// a nil error is received
	var waitForEmulationStart chan error

	// OnLoaded function for specific cartridge formats
	cartload.OnInserted = func(cart mapper.CartMapper) error {
		if _, ok := cart.(*supercharger.Supercharger); ok {
			return tv.Reset(false)
		} else if pr, ok := cart.(*plusrom.PlusROM); ok {
			if pr.Prefs.NewInstallation {
				waitForEmulationStart = make(chan error)

				fi := gui.PlusROMFirstInstallation{Finish: waitForEmulationStart, Cart: pr}
				err := scr.SetFeature(gui.ReqPlusROMFirstInstallation, &fi)
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

	// replace player 1 port with savekey
	if useSavekey {
		err = vcs.RIOT.Ports.Plug(plugging.RightPlayer, savekey.NewSaveKey)
		if err != nil {
			return curated.Errorf("playmode: %v", err)
		}
	}

	// attach the cartridge depending on whether it's a new recording an
	// existing recording (ie. a playback) or when no recording is involved at
	// all.

	if newRecording {
		// new recording requested

		// create a unique filename
		recording = paths.UniqueFilename("recording", cartload.ShortName())

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
		vcs:       vcs,
		scr:       scr,
		intChan:   make(chan os.Signal, 1),
		userinput: make(chan userinput.Event, 10),
		rawEvents: make(chan func(), 1024),
	}

	// connect gui
	err = scr.SetFeature(gui.ReqSetEmulation, pl)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
	}

	vcs.RIOT.Ports.AttachPlugMonitor(pl)

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

	// notify gui that we're running
	pl.state = emulation.Running
	err = scr.SetFeature(gui.ReqState, pl.state)
	if err != nil {
		return curated.Errorf("playmode: %v", err)
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
		if curated.Has(err, quitEvent) {
			// quitEvent is okay and is to be expected. swallow the error
			// message and return as normal
			return nil
		}
		if curated.Has(err, terminal.UserInterrupt) {
			// quitEvent is okay and is to be expected. swallow the error
			// message and return as normal
			return nil
		}
		return curated.Errorf("playmode: %v", err)
	}

	return nil
}

// Plugged implements the pluggin.PlugMonitor interface.
func (pl *playmode) Plugged(port plugging.PortID, description string) {
	pl.scr.SetFeature(gui.ReqControllerChange, port, description)
}
