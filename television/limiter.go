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
	"time"
)

type limiter struct {
	// whether to wait for fps limited each frame
	limit bool

	// the number of scanline since last limit operation
	scanlines int

	// the requested number of frames per second
	requested         float32
	scanlinesPerFrame int

	// actual calculation
	actual         float32
	actualCt       int
	actualCtTarget int
	actualRefTime  time.Time

	// channels
	sync    chan bool
	reqRate chan time.Duration
}

func (lmtr *limiter) init() {
	lmtr.limit = true
	lmtr.actualRefTime = time.Now()
	lmtr.sync = make(chan bool)
	lmtr.reqRate = make(chan time.Duration)

	go func() {
		// new ticker with an arbitrary value. it'll get changed soon enough
		tck := time.NewTicker(1)

		for {
			select {
			case <-tck.C:
				select {
				case lmtr.sync <- true:

				// listen for limtReqRate signals too while signalling the
				// limitTick channel.
				//
				// if we don't do this here, it's possible for the limitTick to
				// deadlock, even with very large buffers on limitReqRate. an
				// exceedingly large buffer might work but it's too risky
				//
				// we could add a small buffer to the limitTick channel but
				// any kind of buffering seems to upset the accuracy of
				// time.Ticker's self regulation.
				case d := <-lmtr.reqRate:
					tck.Stop()
					tck = time.NewTicker(d)
				}

			// listen for limtReqRate signals too while signalling the
			// limitTick channel. we're doing this here in addition to above
			// because this is also a source for deadlocking and just generally
			// slow response times if the Ticker duration is very long.
			case d := <-lmtr.reqRate:
				tck.Stop()
				tck = time.NewTicker(d)
			}
		}
	}()
}

// set target rate and the number of scanlines considered to be a frame
func (lmtr *limiter) setRate(fps float32, scanlinesPerFrame int) {
	if fps < 0 {
		return
	}

	lmtr.requested = fps
	lmtr.scanlinesPerFrame = scanlinesPerFrame

	rate, _ := time.ParseDuration(fmt.Sprintf("%fs", float32(1.0)/lmtr.requested))
	lmtr.reqRate <- rate

	lmtr.actualCtTarget = int(lmtr.requested) / 2
	lmtr.actualCt = 0
	lmtr.actualRefTime = time.Now()
}

// check fps rate and pause if necessary. we call this every scanline iteration
// and not every frame iteration. new frames are unpredictble, scanlines are
// like clockwork.
func (lmtr *limiter) checkRate() {
	lmtr.scanlines++
	if lmtr.scanlines > lmtr.scanlinesPerFrame {
		lmtr.scanlines = 0
		if lmtr.limit {
			<-lmtr.sync
		}
		lmtr.measureActual()
	}
}

// called every frame to calculate the actual frame rate being achieved
func (lmtr *limiter) measureActual() {
	lmtr.actualCt++
	if lmtr.actualCt >= lmtr.actualCtTarget {
		t := time.Now()
		lmtr.actual = float32(lmtr.actualCtTarget) / float32(t.Sub(lmtr.actualRefTime).Seconds())

		// actualCtTarget is the number of frames to count before taking the
		// acutal measurement. we set this to the new actual value, which means
		// we'll be remeasuring every second or so. if actual is less than 1
		// howevre, we set actualCtTarget to 1, which means we'll be
		// re-measuring every frame.
		if lmtr.actual > 1 {
			lmtr.actualCtTarget = int(lmtr.actual)
		} else {
			lmtr.actualCtTarget = 1
		}

		// not start time for next calculation
		lmtr.actualRefTime = t

		lmtr.actualCt = 0
	}
}
