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

type limitScale int

const (
	scaleFrame limitScale = iota
	scaleScanline
)

type limiter struct {
	tv *Television

	// whether to wait for fps limited each frame
	limit bool

	// the requested number of frames per second
	requested atomic.Value // float32

	// the actual number of frames per second
	actual atomic.Value // float32

	// whether to update the screen visually - when frame rate is low enough we
	// want to be able to see the updates
	visualUpdates bool

	// event pulse
	pulse *time.Ticker
	scale limitScale

	// measurement
	measureCt      int
	measureTime    time.Time
	measuringPulse *time.Ticker
}

func (lmtr *limiter) init(tv *Television) {
	lmtr.actual.Store(float32(0))
	lmtr.requested.Store(float32(0))
	lmtr.tv = tv
	lmtr.limit = true
	lmtr.measureTime = time.Now()
	lmtr.pulse = time.NewTicker(time.Millisecond * 10)
	lmtr.measuringPulse = time.NewTicker(time.Second)
}

// there's no science behind when we flip from scales these values are based simply on
// what looks effective and what seems to be useable.
const (
	theshScanlineScale float32 = 5.0
	ThreshVisual       float32 = 3.0
)

func (lmtr *limiter) setRate(fps float32) {
	// if number is negative then default to ideal FPS rate
	if fps <= 0.0 {
		fps = lmtr.tv.state.spec.FramesPerSecond
	}

	// if fps is still zero (spec probably hasn't been set) then don't do anything
	if fps == 0.0 {
		return
	}

	// not selected rate
	lmtr.requested.Store(fps)

	// set scale and duration to wait according to requested FPS rate
	if fps <= theshScanlineScale {
		lmtr.scale = scaleScanline

		// to prevent the emulator from stalling every frame while it waits for
		// the ticker to catch up, frame rates below theshScanlineScale are
		// rate checked every scanline
		//
		// the requires us to multiply the frame rate by the number scanlines
		// in a frame. by default this is the ScanlinesTotal value in the spec
		// but if the screen is "bigger" than that then we use the larger
		// value.
		scanlines := lmtr.tv.state.spec.ScanlinesTotal
		if lmtr.tv.state.resizer.bottom > lmtr.tv.state.spec.ScanlinesTotal {
			scanlines = lmtr.tv.state.resizer.bottom
		}

		rate := float32(1000000.0) / (fps * float32(scanlines))
		dur, _ := time.ParseDuration(fmt.Sprintf("%fus", rate))
		lmtr.pulse.Reset(dur)
	} else {
		lmtr.scale = scaleFrame
		rate := float32(1000000.0) / fps
		dur, _ := time.ParseDuration(fmt.Sprintf("%fus", rate))
		lmtr.pulse.Reset(dur)
	}

	// visual updates or not
	lmtr.visualUpdates = fps <= ThreshVisual

	// restart acutal FPS rate measurement values
	lmtr.measureCt = 0
	lmtr.measureTime = time.Now()
}

func (lmtr *limiter) checkFrame() {
	lmtr.measureCt++
	if lmtr.scale == scaleFrame && lmtr.limit {
		<-lmtr.pulse.C
	}
}

func (lmtr *limiter) checkScanline() {
	if lmtr.scale == scaleScanline && lmtr.limit {
		<-lmtr.pulse.C
	}
}

// measures frame rate on every tick of the measuringPulse ticker.
func (lmtr *limiter) measureActual() {
	select {
	case <-lmtr.measuringPulse.C:
		t := time.Now()
		lmtr.actual.Store(float32(lmtr.measureCt) / float32(t.Sub(lmtr.measureTime).Seconds()))

		// reset time and count ready for next measurement
		lmtr.measureTime = t
		lmtr.measureCt = 0
	default:
	}
}
