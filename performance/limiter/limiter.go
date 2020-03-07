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

// Package limiter provides a rough and ready way of limiting events to a fixed
// rate.
//
// A new FpsLimiter can be created with (error handling removed for clarity):
//
//	fps, _ := limiter.NewFPSLimiter(60)
//
// Operations can then be stalled with the Wait() function. For example:
//
//	for {
//		fps.Wait()
//		renderImage()
//	}
package limiter

import (
	"fmt"
	"sync/atomic"
	"time"
)

// this is a really rough attempt at frame rate limiting. probably only any
// good if base performance of the machine is well above the required rate.

// FpsLimiter will trigger every frames per second
type FpsLimiter struct {
	// toggle limited on and off
	Active bool

	RequestedFPS    float32
	secondsPerFrame time.Duration
	tick            chan bool
	tickNow         chan bool

	trackFrameCt uint32
	ActualFPS    float32
}

// NewFPSLimiter is the preferred method of initialisation for FpsLimiter type
func NewFPSLimiter(RequestedFPS float32) *FpsLimiter {
	lim := &FpsLimiter{Active: true}
	lim.tick = make(chan bool)
	lim.tickNow = make(chan bool)

	rateTimer := time.NewTimer(lim.secondsPerFrame)

	// run ticker concurrently
	go func() {
		for {
			lim.tick <- true
			select {
			case <-rateTimer.C:
			case <-lim.tickNow:
				rateTimer.Stop()
			}
			rateTimer.Reset(lim.secondsPerFrame)
		}
	}()

	// fun fps calculator concurrently
	go func() {
		t := time.Now()

		updateRate, _ := time.ParseDuration("0.5s")

		for {
			// wait for spcified duration
			time.Sleep(updateRate)

			// acutal end time
			et := time.Now()

			// calculate actual rate
			frames := float32(atomic.LoadUint32(&lim.trackFrameCt))
			lim.ActualFPS = frames / float32(et.Sub(t).Seconds())
			atomic.StoreUint32(&lim.trackFrameCt, 0)

			// new start time
			t = et
		}
	}()

	lim.RequestedFPS = RequestedFPS
	lim.secondsPerFrame, _ = time.ParseDuration(fmt.Sprintf("%fs", float32(1.0)/float32(RequestedFPS)))

	return lim
}

// SetFPS changes the limit at which the FpsLimiter waits
func (lim *FpsLimiter) SetFPS(RequestedFPS float32) {
	lim.RequestedFPS = RequestedFPS
	lim.secondsPerFrame, _ = time.ParseDuration(fmt.Sprintf("%fs", float32(1.0)/float32(RequestedFPS)))

	select {
	case lim.tickNow <- true:
	default:
	}
}

// Wait will block until trigger
func (lim *FpsLimiter) Wait() {
	if lim.Active {
		<-lim.tick
	}
	ct := atomic.AddUint32(&lim.trackFrameCt, 1)
	atomic.StoreUint32(&lim.trackFrameCt, ct)
}

// HasWaited will return true if time has already elapsed and false it it is
// still yet to happen
func (lim *FpsLimiter) HasWaited() bool {
	select {
	case <-lim.tick:
		return true
	default:
		// default case means that the channel receiving case doesn't block
		return false
	}
}
