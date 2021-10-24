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

package television

import (
	"fmt"
	"sync/atomic"
	"time"
)

// VisualUpdating is the value at which the screen drawing process should be
// shown to the user. ie. the FPS is low enough to require a visual indicator
// that something is happening.
const VisualUpdating float32 = 5.0

type limiter struct {
	// whether to wait for fps limited each frame
	active bool

	// the refresh rate of the TV signal. this is a copy of what is stored in
	// the FrameInfo type of the TV/state type. that value however is in a
	// critical section and it is easier/cleaner to store a copy locally as an
	// atomic value
	refreshRate atomic.Value // float32

	// the requested number of frames per second
	requested atomic.Value // float32

	// whether the requested frame rate is equal to the refresh rate
	matchRefreshRate atomic.Value // bool

	// the actual number of frames per second
	actual atomic.Value // float32

	// whether to update the screen visually - when frame rate is low enough we
	// want to be able to see the updates
	visualUpdates bool

	// pulse that performs the limiting. the duration of the ticker will be set
	// when the frame rate changes
	pulse *time.Ticker

	// measurement
	measureCt      int
	measureTime    time.Time
	measuringPulse *time.Ticker

	// realtime audio should not be allowed if actual speed of the emulation is
	// too low or too high
	//
	// this is good for machines that just run too slow (realtime audio might
	// sound odd) but it's also good for the debugger I think
	realtimeAudio bool
}

func (lmtr *limiter) init() {
	lmtr.active = true
	lmtr.refreshRate.Store(float32(0))
	lmtr.matchRefreshRate.Store(true)
	lmtr.requested.Store(float32(0))
	lmtr.actual.Store(float32(0))
	lmtr.measureTime = time.Now()
	lmtr.pulse = time.NewTicker(time.Millisecond * 10)
	lmtr.measuringPulse = time.NewTicker(time.Second)
}

func (lmtr *limiter) setRefreshRate(tv *Television, refreshRate float32) {
	lmtr.refreshRate.Store(refreshRate)
	if lmtr.matchRefreshRate.Load().(bool) {
		lmtr.setRate(tv, refreshRate)
	}
}

func (lmtr *limiter) setRate(tv *Television, fps float32) {
	// if number is negative then default to ideal FPS rate
	if fps <= 0.0 {
		lmtr.matchRefreshRate.Store(true)
		fps = lmtr.refreshRate.Load().(float32)
	}

	// if fps is still zero (spec probably hasn't been set) then don't do anything
	if fps == 0.0 {
		return
	}

	// not selected rate
	lmtr.requested.Store(fps)

	// set scale and duration to wait according to requested FPS rate
	if fps <= VisualUpdating {
		lmtr.visualUpdates = true
		rate := float32(1000000.0) / (fps * float32(tv.state.frameInfo.TotalScanlines))
		dur, _ := time.ParseDuration(fmt.Sprintf("%fus", rate))
		lmtr.pulse.Reset(dur)
	} else {
		lmtr.visualUpdates = false
		rate := float32(1000000.0) / fps
		dur, _ := time.ParseDuration(fmt.Sprintf("%fus", rate))
		lmtr.pulse.Reset(dur)
	}

	// restart acutal FPS rate measurement values
	lmtr.measureCt = 0
	lmtr.measureTime = time.Now()
}

// checkFrame should be called every frame.
func (lmtr *limiter) checkFrame() {
	lmtr.measureCt++
	if lmtr.active && !lmtr.visualUpdates {
		<-lmtr.pulse.C
	}
}

// checkFrame should be called every scanline.
func (lmtr *limiter) checkScanline() {
	if lmtr.active && lmtr.visualUpdates {
		<-lmtr.pulse.C
	}
}

// measures frame rate on every tick of the measuringPulse ticker. callers of
// measureActual() should be mindful of how ofter the function is called,
// regardless of the throttle provided by the measuring pulse - checking the
// pulse channel is itself expensive.
func (lmtr *limiter) measureActual() {
	select {
	case <-lmtr.measuringPulse.C:
		t := time.Now()

		actual := float32(lmtr.measureCt) / float32(t.Sub(lmtr.measureTime).Seconds())
		lmtr.actual.Store(actual)

		// reset time and count ready for next measurement
		lmtr.measureTime = t
		lmtr.measureCt = 0

		// check whether realtimeAudio should be allowed
		refresh := lmtr.refreshRate.Load().(float32)
		lmtr.realtimeAudio = actual >= refresh*0.90 && actual <= refresh*1.1

	default:
	}
}
