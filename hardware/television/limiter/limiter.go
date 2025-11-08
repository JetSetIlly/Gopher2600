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

package limiter

import (
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type Display interface {
	DisplayRefreshRate() (float32, bool)
}

type Limiter struct {
	// whether to wait for fps limited each frame
	Active bool

	// the refresh rate of the TV signal. this is a copy of what is stored in
	// the FrameInfo type of the TV/state type. that value however is in a
	// critical section and it is easier/cleaner to store a copy locally as an
	// atomic value
	RefreshRate atomic.Value // float32

	// the ideal number of frames per second if everything was working nicely
	IdealFPS atomic.Value // float32

	// the actual value sent to the SetLimit() function. no
	requestedFPS atomic.Value // float32

	// whether the requested frame rate is to equal the refresh rate
	requestMatchRefreshRate atomic.Value // bool

	// pulse that performs the limiting. the duration of the ticker will be set
	// when SetLimit() is called with a new fps value
	pulse *time.Ticker

	// we don't want to measure the frame rate too often because it's
	// relatively expensive. a simple counter is good enough for this
	pulseCt      int
	pulseCtLimit int

	// pulse that performs the FPS measurement
	measuringPulse *time.Ticker

	// the measured FPS is the number of frames divided by the amount of
	// elapsed time since the previous measurement
	measureTime time.Time
	measureCt   int

	// the measured number of frames per second
	Measured atomic.Value // float32

	// the number of frames to wait after SetRefreshRate() before the frame
	// limiter is adjusted to match
	//
	// some kernels will cause the refresh rate to flucutate wildly and
	// immediately altering the frame limiter will cause performance problems
	//
	// value will decrease to zero on every CheckFrame(). rate will change when
	// it reaches zero
	//
	// if matchRefreshRate is true then the syncWithRefreshRateDelay will be set
	// to a value of half refresh-rate
	//
	// is not set if SetRate() is called directly
	syncWithRefreshRateDelay int

	// nudge the limiter so that it doesn't wait for the specified number of frames
	Nudge atomic.Int32

	// the display the limiter is working for
	display Display
}

// NewLimiter is preferred method of initialising a new instance of the Limiter
// type. The refresh rate will be set to 60Hz and the limited rate set to match
// the refresh rate.
func NewLimiter() *Limiter {
	lmtr := Limiter{}
	lmtr.Active = true
	lmtr.requestMatchRefreshRate.Store(false)
	lmtr.Measured.Store(float32(0.0))

	lmtr.pulse = time.NewTicker(time.Millisecond * 16)
	lmtr.measuringPulse = time.NewTicker(time.Millisecond * 1000)

	lmtr.SetRefreshRate(specification.SpecNTSC.RefreshRate)
	lmtr.SetLimit(MatchRefreshRate)

	return &lmtr
}

// Set the display interface for the limiter
func (lmtr *Limiter) SetDisplay(display Display) {
	lmtr.display = display
	lmtr.SetLimit(lmtr.requestedFPS.Load().(float32))
}

// Set the refresh rate for the limiter. This is equivalent to the refresh rate
// of the television. It is distinict from the limit value but is related and
// the limit value (see SetLimit() function) will usually equal the refresh rate
func (lmtr *Limiter) SetRefreshRate(refreshRate float32) {
	lmtr.RefreshRate.Store(refreshRate)
	if lmtr.requestMatchRefreshRate.Load().(bool) {
		lmtr.syncWithRefreshRateDelay = int(refreshRate / 2)
	}
}

const MatchRefreshRate float32 = -1.0

// Set frame limit. Use a value of MatchRefreshRate to indicate that the limiter should equal the
// television refresh rate.
func (lmtr *Limiter) SetLimit(fps float32) {
	// the actual fps value passed to the function
	lmtr.requestedFPS.Store(fps)

	if fps <= 0.0 {
		fps = lmtr.RefreshRate.Load().(float32)
		lmtr.requestMatchRefreshRate.Store(fps == lmtr.RefreshRate.Load().(float32))
	} else {
		lmtr.requestMatchRefreshRate.Store(false)
	}

	// reset refresh rate delay counter
	lmtr.syncWithRefreshRateDelay = 0

	// if fps is still zero (spec probably hasn't been set) then don't do anything
	if fps == 0.0 {
		return
	}

	// quantise refresh rate based on refresh rate of television
	if lmtr.display != nil {
		hz, quantise := lmtr.display.DisplayRefreshRate()
		if quantise {
			if fps >= hz*0.96 && fps <= hz*1.04 {
				fps = hz
			}
		}
	}

	// the ideal fps rate including quantisation
	lmtr.IdealFPS.Store(fps)

	// set scale and duration to wait according to requested FPS rate
	lmtr.pulseCt = 0
	lmtr.pulseCtLimit = 1 + int(fps/20)
	lmtr.pulse.Stop()
	lmtr.pulse.Reset(time.Duration(1000000000 / fps * float32(lmtr.pulseCtLimit)))

	// restart acutal FPS rate measurement values
	lmtr.measureCt = 0
	lmtr.measureTime = time.Now()
}

// CheckFrame should be called every frame.
func (lmtr *Limiter) CheckFrame() {
	lmtr.measureCt++

	nudge := lmtr.Nudge.Load()
	if nudge > 0 {
		lmtr.Nudge.Store(nudge - 1)
	} else {
		if lmtr.Active {
			lmtr.pulseCt++
			if lmtr.pulseCt >= lmtr.pulseCtLimit {
				lmtr.pulseCt = 0
				<-lmtr.pulse.C
			}
		}
	}

	// check to see if rate is to change
	if lmtr.syncWithRefreshRateDelay > 0 {
		lmtr.syncWithRefreshRateDelay--
		if lmtr.syncWithRefreshRateDelay == 0 {
			lmtr.SetLimit(MatchRefreshRate)
		}
	}
}

// MeasureActual measures frame rate on every tick of the measuringPulse ticker.
// callers of MeasureActual() should be mindful of how ofter the function is
// called, regardless of the throttle provided by the measuring pulse - checking
// the pulse channel is itself expensive.
func (lmtr *Limiter) MeasureActual() {
	select {
	case <-lmtr.measuringPulse.C:
		t := time.Now()
		m := float32(lmtr.measureCt) / float32(t.Sub(lmtr.measureTime).Seconds())
		lmtr.Measured.Store(m)

		// reset time and count ready for next measurement
		lmtr.measureTime = t
		lmtr.measureCt = 0
	default:
	}
}
