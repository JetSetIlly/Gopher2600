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

	// the amount of time between interface renders caused by window resizing
	resizePeriod = 5
)

type polling struct {
	img *SdlImgui

	// queue of pumped events for the frame
	pumpedEvents []sdl.Event

	// functions that need to be performed in the main thread are queued for
	// serving by the service() function
	service    chan func()
	serviceErr chan error

	// ReqFeature() and GetFeature() hands off requests to the featureReq
	// channel for servicing. think of these as pecial instances of the
	// service chan
	featureSet    chan featureRequest
	featureSetErr chan error

	// the following are not used in playmode

	// alerted is used to preempt the tickers when we want to communicate between
	// iterations of the service loop. for example, closing sdlimgui windows
	// might feel laggy without it (see commentary in service loop for
	// explanation).
	alerted bool

	// the keepAwake flag is set to true for a short time (defined by
	// wakefullnessPeriod) after the last event was received. this improves
	// responsiveness for certain GUI operations.
	keepAwake bool
	lastEvent time.Time

	lastThmbTime   time.Time
	lastResizeTime time.Time
}

func newPolling(img *SdlImgui) *polling {
	pol := &polling{
		img:           img,
		pumpedEvents:  make([]sdl.Event, 64),
		service:       make(chan func(), 1),
		serviceErr:    make(chan error, 1),
		featureSet:    make(chan featureRequest, 1),
		featureSetErr: make(chan error, 1),
	}

	return pol
}

// wait for an SDL event or for a timeout. the timeout duration depends on the
// state of the emulation and receent user input.
func (pol *polling) wait() sdl.Event {
	select {
	case f := <-pol.service:
		f()
	case r := <-pol.featureSet:
		pol.img.serviceSetFeature(r)
	default:
	}

	// the amount of timeout depends on mode and state
	var timeout int

	if pol.img.isPlaymode() {
		if pol.img.dbg.State() != govern.Paused || pol.img.prefs.activePause.Get().(bool) {
			timeout = playSleepPeriod
		} else {
			timeout = playPausedSleepPeriod
		}
	} else {
		if pol.img.dbg.State() == govern.Running {
			timeout = debugSleepPeriodRunning
		} else if pol.alerted || pol.keepAwake || pol.img.lz.Debugger.HasChanged {
			timeout = debugSleepPeriod
		} else {
			timeout = idleSleepPeriod
		}
	}

	ev := sdl.WaitEventTimeout(timeout)

	if ev != nil {
		// an event has been received so set keepAwake flag and note time of event
		pol.keepAwake = true
		pol.lastEvent = time.Now()
	} else if pol.keepAwake {
		// keepAwake flag set for wakefullnessPeriod milliseconds
		pol.keepAwake = time.Since(pol.lastEvent).Milliseconds() < wakefullnessPeriod
	}

	// always reset alerted flag
	pol.alerted = false

	return ev
}

// returns true if sufficient time has passed since the last thumbnail generation
func (pol *polling) throttleTimelineThumbnailer() bool {
	if time.Since(pol.lastThmbTime).Milliseconds() > timelineThumbnailerPeriod {
		pol.lastThmbTime = time.Now()
		return true
	}
	return false
}

// returns true if sufficient time has passed since the last window resize event
func (pol *polling) throttleResize() bool {
	if time.Since(pol.lastResizeTime).Milliseconds() > resizePeriod {
		pol.lastResizeTime = time.Now()
		pol.lastEvent = pol.lastResizeTime
		return true
	}
	return false
}
