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
	scalePixel
)

type limiter struct {
	tv *Television

	// whether to wait for fps limited each frame
	limit bool

	// the requested number of frames per second
	requested atomic.Value // float32

	// event pulse
	pulse *time.Ticker
	scale limitScale

	// measurement
	actual         atomic.Value // float32
	actualCt       int
	actualCtTarget int
	actualTime     time.Time
}

func (lmtr *limiter) init(tv *Television) {
	lmtr.actual.Store(float32(0))
	lmtr.requested.Store(float32(0))
	lmtr.tv = tv
	lmtr.limit = true
	lmtr.actualTime = time.Now()
	lmtr.pulse = time.NewTicker(time.Millisecond * 10)
}

// there's no science behind when we flip from scales these values are based simply on
// what looks effective and what seems to be useable.
const (
	threshScanlineScale float32 = 10.0
	thresPixelScale     float32 = 1.0
)

func (lmtr *limiter) setRate(fps float32) {
	// if number is negative then default to ideal FPS rate
	if fps < 0 {
		fps = lmtr.tv.state.spec.FramesPerSecond
	}

	// not selected rate
	lmtr.requested.Store(fps)

	// set scale and duration to wait according to requested FPS rate
	if fps < thresPixelScale {
		lmtr.scale = scalePixel
		dur := time.Duration(279000 * fps)
		lmtr.pulse.Reset(dur)
	} else if fps < threshScanlineScale {
		lmtr.scale = scaleScanline
		rate := float32(1.0) / (fps * float32(lmtr.tv.state.spec.ScanlinesTotal))
		dur, _ := time.ParseDuration(fmt.Sprintf("%fs", rate))
		lmtr.pulse.Reset(dur)
	} else {
		lmtr.scale = scaleFrame
		rate := float32(1.0) / fps
		dur, _ := time.ParseDuration(fmt.Sprintf("%fs", rate))
		lmtr.pulse.Reset(dur)
	}

	// restart acutal FPS rate measurement values
	lmtr.actualCt = 0
	lmtr.actualCtTarget = int(lmtr.requested.Load().(float32)) / 2
	lmtr.actualTime = time.Now()
}

func (lmtr *limiter) checkFrame() {
	if lmtr.scale != scaleFrame || !lmtr.limit {
		return
	}

	<-lmtr.pulse.C
	lmtr.measureActual()
}

func (lmtr *limiter) checkScanline() {
	if lmtr.scale != scaleScanline || !lmtr.limit {
		return
	}

	<-lmtr.pulse.C
	lmtr.measureActual()
}

func (lmtr *limiter) checkPixel() {
	if lmtr.scale != scalePixel || !lmtr.limit {
		return
	}

	<-lmtr.pulse.C
	lmtr.measureActual()
}

// called every scanline (although internally limited) to calculate the actual
// frame rate being achieved.
func (lmtr *limiter) measureActual() {
	lmtr.actualCt++
	if lmtr.actualCt >= lmtr.actualCtTarget {
		t := time.Now()
		lmtr.actual.Store(float32(lmtr.actualCtTarget) / float32(t.Sub(lmtr.actualTime).Seconds()))

		actual := lmtr.actual.Load().(float32)
		switch lmtr.scale {
		case scaleScanline:
			lmtr.actual.Store(actual / float32(lmtr.tv.state.spec.ScanlinesTotal))
		case scalePixel:
			lmtr.actual.Store(actual / float32(lmtr.tv.state.spec.IdealPixelsPerFrame))
		}

		// reset time and count ready for next measurement
		lmtr.actualTime = t
		lmtr.actualCt = 0
	}
}
