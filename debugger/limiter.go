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

package debugger

import (
	"fmt"
	"time"

	"github.com/jetsetilly/gopher2600/television"
)

// a specialised frame-rate limiter for use by the debugger. this
// implementation handles low frame rates more elegantly by changing *when* the
// rate is throttled. if we only ever throttled at the frame level then low
// frame rates would result in a disconcerting stutter - the emulation will run
// at full speed for one frame and then pause. it is far better therefore, to
// throttle at the scanline or even the color clock level for low and very low
// frame rates.
//
// we've "hooked" into the television system by using (abusing?) the
// PixelRenderer interface. It gives us everything we need: a NewFrame() and
// NewScanline() function and also an effective color clock notifier through
// the Pixel() function.
//
// note that frame rate measurement is still performed by the television. Use
// the GetActualFPS() function in the television interface.
//
// we should probably get this version of the limiter to measure actual fps
// to ensure accuracy when we flip to scanline/colclock throttle levels.

// there's no science behind when we flip from one throttle system to another.
// the thresholds are based simply on what looks effective.
const (
	scanlineGranThresh float32 = 10.0
	colorClkGranThresh float32 = 1.0
)

// keeping track of the current throttle level
type throttleLevel int

const (
	throtFrame throttleLevel = iota
	throtScanline
	throtColClock
)

type limiter struct {
	tv        television.Television
	lmtr      *time.Ticker
	throt     throttleLevel
	reqFrames float32

	// it is important that events are still monitored while waiting for the
	// lmtr Ticker to signal. this is of particular importance for very low
	// frame rates in order to keep GUI interfaces responsive
	eventPulse  *time.Ticker
	checkEvents func() error
}

func newLimiter(tv television.Television, checkEvents func() error) *limiter {
	tv.SetFPSCap(false)

	lmtr := &limiter{
		tv:          tv,
		checkEvents: checkEvents,
	}
	tv.AddPixelRenderer(lmtr)
	lmtr.setFPS(-1)

	// set up event pulse. pulse generated infrequently
	dur, _ := time.ParseDuration(fmt.Sprintf("%fs", 1/15.0))
	lmtr.eventPulse = time.NewTicker(dur)

	return lmtr
}

func (lmtr *limiter) setFPS(fps float32) {
	spec, _ := lmtr.tv.GetSpec()
	if fps < 0 {
		fps = spec.FramesPerSecond
	}
	lmtr.reqFrames = fps

	// find the required rate
	var dur time.Duration

	if fps < colorClkGranThresh {
		// very low frame rates requires colorclock granularity
		lmtr.throt = throtColClock
		dur = time.Duration(279000 * fps)
	} else if fps < scanlineGranThresh {
		// low frame rates and we move to scanline granularity
		rate := float32(1.0) / (fps * float32(spec.ScanlinesTotal))
		dur, _ = time.ParseDuration(fmt.Sprintf("%fs", rate))
		lmtr.throt = throtScanline
	} else {
		// frame granularity
		rate := float32(1.0) / fps
		dur, _ = time.ParseDuration(fmt.Sprintf("%fs", rate))
		lmtr.throt = throtFrame
	}

	// start ticker with new duration
	if lmtr.lmtr != nil {
		lmtr.lmtr.Stop()
	}
	lmtr.lmtr = time.NewTicker(dur)

	// event though the tv frame limiter is disabled we call the SetFPS()
	// function so that the frame rate calculator is working with reasonable
	// information
	lmtr.tv.SetFPS(fps)
}

// GetReqFPS returens the requested number of frames per second. The limiter
// type has no GetActualFPS() function. Use the equivalent function from the
// television implementation.
//
// *Use this in preference to SetFPS() from the television implementation*
func (lmtr *limiter) getReqFPS() float32 {
	return lmtr.reqFrames
}

// Resize implements television.PixelRenderer
func (lmtr *limiter) Resize(_ *television.Specification, topScanline int, visibleScanlines int) error {
	lmtr.setFPS(-1)
	return nil
}

// NewFrame implements television.PixelRenderer
func (lmtr *limiter) NewFrame(_ int, _ bool) error {
	if lmtr.throt != throtFrame {
		return nil
	}
	return lmtr.limit()
}

// NewScanline implements television.PixelRenderer
func (lmtr *limiter) NewScanline(_ int) error {
	if lmtr.throt != throtScanline {
		return nil
	}
	return lmtr.limit()
}

// SetPixel implements television.PixelRenderer
func (lmtr *limiter) SetPixel(_ int, _ int, _ byte, _ byte, _ byte, _ bool) error {
	if lmtr.throt != throtColClock {
		return nil
	}
	return lmtr.limit()
}

// EndRendering implements television.PixelRenderer
func (lmtr *limiter) EndRendering() error {
	lmtr.lmtr.Stop()
	return nil
}

func (lmtr *limiter) limit() error {
	done := false
	for !done {
		select {
		case <-lmtr.lmtr.C:
			done = true
		case <-lmtr.eventPulse.C:
			if err := lmtr.checkEvents(); err != nil {
				return err
			}
		}
	}

	return nil
}
