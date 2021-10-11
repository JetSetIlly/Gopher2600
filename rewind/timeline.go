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

package rewind

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// Timeline provides a summary of the current state of the rewind system.
//
// Useful for GUIs for example, to present the range of frame numbers that are
// available in the rewind history.
type Timeline struct {
	FrameNum       []int
	TotalScanlines []int

	// These two "available" fields state the earliest and latest frames that
	// are available in the rewind history.
	//
	// The earliest information in the Timeline array fields may be different.
	AvailableStart int
	AvailableEnd   int
}

const timelineLength = 1000

func newTimeline() Timeline {
	return Timeline{
		FrameNum:       make([]int, 0),
		TotalScanlines: make([]int, 0),
	}
}

func (tl *Timeline) checkIntegrity() error {
	if len(tl.FrameNum) != len(tl.TotalScanlines) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) == 0 {
		return nil
	}

	if tl.AvailableStart < tl.FrameNum[0] {
		return curated.Errorf("earliest rewind state not in timeline")
	}

	if tl.AvailableEnd > tl.FrameNum[len(tl.FrameNum)-1] {
		return curated.Errorf("most recent rewind state not in timeline")
	}

	if len(tl.FrameNum) > 1 {
		prev := tl.FrameNum[0]
		for _, fn := range tl.FrameNum[1:] {
			if fn != prev+1 {
				return curated.Errorf("frame numbers in timeline are not consecutive")
			}
			prev = fn
		}
	}

	return nil
}

func (tl *Timeline) splice(frameNumber int) {
	for i := range tl.FrameNum {
		if frameNumber == tl.FrameNum[i] {
			tl.FrameNum = tl.FrameNum[:i]
			tl.TotalScanlines = tl.TotalScanlines[:i]
			break // for loop
		}
	}
}

func (r Rewind) GetTimeline() Timeline {
	if err := r.timeline.checkIntegrity(); err != nil {
		panic(err)
	}

	e := r.end - 1
	if e < 0 {
		e += len(r.entries)
	}

	// because of how we generate visual state we cannot generate the image for
	// the first frame in the history unless the first entry represents a
	// machine reset
	//
	// this has a consequence when the first time the circular array wraps
	// around for the first time (the number of available entries drops by one)
	sf := r.entries[r.start].TV.GetState(signal.ReqFramenum)
	if r.entries[r.start].level != levelReset {
		sf++
	}

	r.timeline.AvailableStart = sf
	r.timeline.AvailableEnd = r.entries[e].TV.GetState(signal.ReqFramenum)

	return r.timeline
}
