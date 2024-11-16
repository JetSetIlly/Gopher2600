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

package sdlimgui

import (
	"time"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/veandco/go-sdl2/sdl"
)

// time periods in milliseconds that each mode sleeps for at the end of each
// service() call. this changes depending primarily on whether we're in debug
// or play mode.
const (
	// very small sleep period if emulator is in play mode
	playSleepPeriod = 1

	// slightly longer sleep period if emulator is in play mode and is paused
	playPausedSleepPeriod = 40

	// a small sleep if emulator is in the debugger. this strikes a nice
	// balance between CPU usage and responsiveness
	debugSleepPeriod = 40

	// if emulator is actually running however then there should be hardly any
	// sleep period at all
	debugSleepPeriodRunning = 1

	// debugIdleSleepPeriod should not be too long because the sleep is not preempted.
	idleSleepPeriod = 500

	// the period of inactivity required before the main sleep period drops to
	// the idlsSleepPeriod value.
	wakefullnessPeriod = 3000 // 3 seconds

	// the amount of time between thumbnail generation in the timeline. short
	// enough to feel responsive but long enough to give the mail emulation (if
	// running) time to update
	timelineThumbnailerPeriod = 50

	// the timeout length when polling has been alerted
	alertPeriod = 0
)

type polling struct {
	img *SdlImgui

	// ReqFeature() and GetFeature() hands off requests to the featureReq
	// channel for servicing. think of these as pecial instances of the
	// service chan
	featureSet    chan featureRequest
	featureSetErr chan error

	// alert is used to short-circuit any condition that increases the timeout
	// period. it is saying that the next event should be serviced as soon as
	// possible
	alert bool

	// the following are not used in playmode

	// the keepAwake flag is set to true for a short time (defined by
	// wakefullnessPeriod) after the last event was received. this improves
	// responsiveness for certain GUI operations.
	keepAwake     bool
	keepAwakeTime time.Time

	// the last time a thumbnail was created for the timeline. we don't need to
	// generate a thumbnail too often
	lastThmbTime time.Time
}

func newPolling(img *SdlImgui) *polling {
	pol := &polling{
		img:           img,
		featureSet:    make(chan featureRequest, 1),
		featureSetErr: make(chan error, 1),
	}

	return pol
}

// handle any requests for features or for functions to be run in the main thread
func (pol *polling) serviceRequests() {
	select {
	case r := <-pol.featureSet:
		pol.img.serviceSetFeature(r)
	default:
	}
}

// timeout selects the appropriate duration value based on the current state of
// the application. generally, the values are different for the debugger than
// for playmode
func (pol *polling) timeout() time.Duration {
	if pol.alert {
		pol.alert = false
		return alertPeriod
	}

	// the amount of timeout depends on mode and state
	var timeout int

	if pol.img.isPlaymode() {
		if pol.img.dbg.State() != govern.Paused || pol.img.prefs.activePause.Get().(bool) || pol.img.wm.playmodeWindows[winSelectROMID].playmodeIsOpen() {
			timeout = playSleepPeriod
		} else {
			timeout = playPausedSleepPeriod
		}
	} else {
		// if mouse is being held (eg. on the "step scanline" button) then it
		// will not be detected as an event we therefore must explicitely test
		// if it's being held
		_, _, mouseState := sdl.GetMouseState()
		mouseHeld := mouseState&sdl.ButtonLMask() != 0

		if mouseHeld {
			timeout = debugSleepPeriodRunning
			pol.keepAwake = true
		} else if pol.img.dbg.State() == govern.Running || pol.img.wm.debuggerWindows[winSelectROMID].debuggerIsOpen() {
			timeout = debugSleepPeriodRunning
		} else if pol.keepAwake {
			// this branch used to depend on a debugger flag "hasChanged". this
			// no longer seems necessary, maybe because the govern.Running state
			// is so well defined
			timeout = debugSleepPeriod

			// measure how long we've been awake and
			pol.keepAwake = time.Since(pol.keepAwakeTime).Milliseconds() < wakefullnessPeriod
		} else {
			timeout = idleSleepPeriod
		}

	}

	return time.Millisecond * time.Duration(timeout)
}

// sets the keepAwake flag and notes the time
func (pol *polling) awaken() {
	pol.keepAwake = true
	pol.keepAwakeTime = time.Now()
}

// returns true if sufficient time has passed since the last thumbnail generation
func (pol *polling) throttleTimelineThumbnailer() bool {
	if time.Since(pol.lastThmbTime).Milliseconds() > timelineThumbnailerPeriod {
		pol.lastThmbTime = time.Now()
		return true
	}
	return false
}
