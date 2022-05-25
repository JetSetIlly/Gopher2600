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

type limiter struct {
	tv *Television

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

	// pulse that performs the limiting. the duration of the ticker will be set
	// when the frame rate changes.
	//
	// some kernels will fluctuate wildly between
	pulse *time.Ticker

	// measurement
	measureCt      int
	measureTime    time.Time
	measuringPulse *time.Ticker

	// the number of frames to wait after setRefreshRate() before the frame
	// limiter is adjusted to match
	//
	// some kernels will cause the refresh rate to flucutate wildly and
	// immediately altering the frame limiter will cause performance problems
	//
	// value will decrease to zero on every checkFrame(). rate will change when
	// it reaches zero
	//
	// if matchRefreshRate is true then the matchRefreshRateDelay will be set
	// to a value of half refresh-rate
	//
	// is not set if setRate() is called directly
	matchRefreshRateDelay int
}

func (lmtr *limiter) init(tv *Television) {
	lmtr.tv = tv
	lmtr.active = true
	lmtr.refreshRate.Store(float32(0))
	lmtr.matchRefreshRate.Store(true)
	lmtr.requested.Store(float32(0))
	lmtr.actual.Store(float32(0))
	lmtr.measureTime = time.Now()
	lmtr.pulse = time.NewTicker(time.Millisecond * 10)
	lmtr.measuringPulse = time.NewTicker(time.Second)
}

func (lmtr *limiter) setRefreshRate(refreshRate float32) {
	lmtr.refreshRate.Store(refreshRate)
	if lmtr.matchRefreshRate.Load().(bool) {
		lmtr.matchRefreshRateDelay = int(refreshRate / 2)
	}
}

func (lmtr *limiter) setRate(fps float32) {
	// if number is negative then default to ideal FPS rate
	if fps <= 0.0 {
		lmtr.matchRefreshRate.Store(true)
		fps = lmtr.refreshRate.Load().(float32)
	} else {
		lmtr.matchRefreshRate.Store(fps == lmtr.refreshRate.Load().(float32))
	}

	// reset refresh rate delay counter
	lmtr.matchRefreshRateDelay = 0

	// if fps is still zero (spec probably hasn't been set) then don't do anything
	if fps == 0.0 {
		return
	}

	// not selected rate
	lmtr.requested.Store(fps)

	// set scale and duration to wait according to requested FPS rate
	rate := float32(1000000.0) / fps
	dur, _ := time.ParseDuration(fmt.Sprintf("%fus", rate))
	lmtr.pulse.Reset(dur)

	// restart acutal FPS rate measurement values
	lmtr.measureCt = 0
	lmtr.measureTime = time.Now()
}

// checkFrame should be called every frame.
func (lmtr *limiter) checkFrame() {
	lmtr.measureCt++
	if lmtr.active {
		<-lmtr.pulse.C
	}

	// check to see if rate is to change
	if lmtr.matchRefreshRateDelay > 0 {
		lmtr.matchRefreshRateDelay--
		if lmtr.matchRefreshRateDelay == 0 {
			lmtr.setRate(lmtr.refreshRate.Load().(float32))
		}
	}
}

// checkFrame should be called every scanline.
func (lmtr *limiter) checkScanline() {
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

	default:
	}
}
