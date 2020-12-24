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
	actualTime     time.Time
	measuringPulse *time.Ticker
}

func (lmtr *limiter) init(tv *Television) {
	lmtr.actual.Store(float32(0))
	lmtr.requested.Store(float32(0))
	lmtr.tv = tv
	lmtr.limit = true
	lmtr.actualTime = time.Now()
	lmtr.pulse = time.NewTicker(time.Millisecond * 10)
	lmtr.measuringPulse = time.NewTicker(time.Second)
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
		dur := time.Duration(fps * float32(lmtr.tv.state.spec.IdealPixelsPerFrame))
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
	lmtr.actualTime = time.Now()
}

func (lmtr *limiter) checkFrame() {
	lmtr.actualCt++
	lmtr.measureActual()
	if lmtr.scale == scaleFrame && lmtr.limit {
		<-lmtr.pulse.C
	}
}

func (lmtr *limiter) checkScanline() {
	lmtr.measureActual()
	if lmtr.scale == scaleScanline && lmtr.limit {
		<-lmtr.pulse.C
	}
}

func (lmtr *limiter) checkPixel() {
	lmtr.measureActual()
	if lmtr.scale == scalePixel && lmtr.limit {
		<-lmtr.pulse.C
	}
}

// called every scanline (although internally limited) to calculate the actual
// frame rate being achieved.
func (lmtr *limiter) measureActual() {
	select {
	case <-lmtr.measuringPulse.C:
		t := time.Now()
		lmtr.actual.Store(float32(lmtr.actualCt) / float32(t.Sub(lmtr.actualTime).Seconds()))

		// reset time and count ready for next measurement
		lmtr.actualTime = t
		lmtr.actualCt = 0
	default:
	}
}
