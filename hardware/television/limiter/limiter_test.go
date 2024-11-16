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

package limiter_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/television/limiter"
	"github.com/jetsetilly/gopher2600/test"
)

// tolerance of measurenment
const measurementTolerance = 0.01
const numFramesPerTest = 2

func TestTicker(t *testing.T) {
	lmtr := limiter.NewLimiter()

	var hz float32
	var rate float32

	hz = float32(60.0)
	lmtr.SetRefreshRate(hz)
	for range int(hz * numFramesPerTest) {
		lmtr.CheckFrame()
		lmtr.MeasureActual()
	}
	rate = lmtr.Measured.Load().(float32)
	test.ExpectSuccess(t, rate >= hz*(1.0-measurementTolerance) && rate <= hz*(1.0+measurementTolerance))

	hz = float32(50.0)
	lmtr.SetRefreshRate(hz)
	for range int(hz * numFramesPerTest) {
		lmtr.CheckFrame()
		lmtr.MeasureActual()
	}
	rate = lmtr.Measured.Load().(float32)
	test.ExpectSuccess(t, rate >= hz*(1.0-measurementTolerance) && rate <= hz*(1.0+measurementTolerance))

	hz = float32(60.0)
	lmtr.SetRefreshRate(hz)
	for range int(hz * numFramesPerTest) {
		lmtr.CheckFrame()
		lmtr.MeasureActual()
	}
	rate = lmtr.Measured.Load().(float32)
	test.ExpectSuccess(t, rate >= hz*(1.0-measurementTolerance) && rate <= hz*(1.0+measurementTolerance))
}
