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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// TimelineCounts is returned by a TimelineCounter implementation. The value
// should be updated every *video* cycle. Users can divide by three to get the
// color-clock count.
type TimelineCounts struct {
	WSYNC  int
	CoProc int
}

// TimelineCounter implementations provide system information for the
// timeline that would otherwise be awkward to collate.
//
// Implementations should reset counts in time for the next call to
// TimelineCounts().
type TimelineCounter interface {
	TimelineCounts() TimelineCounts
}

type TimelineRatios struct {
	WSYNC  float32
	CoProc float32
}

// Timeline provides a summary of the current state of the rewind system.
//
// Useful for GUIs for example, to present the range of frame numbers that are
// available in the rewind history.
type Timeline struct {
	FrameNum       []int
	TotalScanlines []int
	Counts         []TimelineCounts
	Ratios         []TimelineRatios

	// peripheral input (including the panel). entry is true if the peripheral
	// was "active" at the end of a frame
	LeftPlayerInput  []bool
	RightPlayerInput []bool
	PanelInput       []bool

	// will be updated if reflection has been attached

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
		FrameNum:         make([]int, 0),
		TotalScanlines:   make([]int, 0),
		Counts:           make([]TimelineCounts, 0),
		Ratios:           make([]TimelineRatios, 0),
		LeftPlayerInput:  make([]bool, 0),
		RightPlayerInput: make([]bool, 0),
		PanelInput:       make([]bool, 0),
	}
}

func (tl *Timeline) reset() {
	tl.FrameNum = tl.FrameNum[:0]
	tl.TotalScanlines = tl.TotalScanlines[:0]
	tl.Counts = tl.Counts[:0]
	tl.Ratios = tl.Ratios[:0]
	tl.LeftPlayerInput = tl.LeftPlayerInput[:0]
	tl.RightPlayerInput = tl.RightPlayerInput[:0]
	tl.PanelInput = tl.PanelInput[:0]
}

func (tl *Timeline) checkIntegrity() error {
	if len(tl.FrameNum) != len(tl.TotalScanlines) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) != len(tl.Counts) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) != len(tl.LeftPlayerInput) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) != len(tl.RightPlayerInput) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) != len(tl.PanelInput) {
		return curated.Errorf("timeline arrays are different lengths")
	}

	if len(tl.FrameNum) == 0 {
		return nil
	}

	if len(tl.FrameNum) > 1 {
		prev := tl.FrameNum[0]
		for _, fn := range tl.FrameNum[1:] {
			if fn != prev+1 {
				return curated.Errorf("frame numbers in timeline are not consecutive (mising frame %d)", prev+1)
			}
			prev = fn
		}
	}

	return nil
}

// GetTimeline returns a copy of the current Timeline. Checks integrity of the
// timeline and will cause the program to panic if it is insane.
func (r *Rewind) GetTimeline() Timeline {
	if err := r.timeline.checkIntegrity(); err != nil {
		panic(err)
	}

	e := r.lastEntryIdx()

	// because of how we generate visual state we cannot generate the image for
	// the first frame in the history unless the first entry represents a
	// machine reset
	//
	// this has a consequence when the first time the circular array wraps
	// around for the first time (the number of available entries drops by one)
	sf := r.entries[r.start].TV.GetCoords().Frame
	if r.entries[r.start].level != levelReset {
		sf++
	}

	r.timeline.AvailableStart = sf
	r.timeline.AvailableEnd = r.entries[e].TV.GetCoords().Frame

	return r.timeline
}

func (r *Rewind) addTimelineEntry(frameInfo television.FrameInfo) {
	// do not alter the timeline information if we're in the rewinding state
	if r.emulation.State() == emulation.Rewinding {
		return
	}

	// splice timeline if newframe is less than the current end framenum. this
	// can happen when the  machine has been rewound to an earlier state
	if len(r.timeline.FrameNum) > 0 && frameInfo.FrameNum <= r.timeline.FrameNum[len(r.timeline.FrameNum)-1] {
		for i := range r.timeline.FrameNum {
			if frameInfo.FrameNum <= r.timeline.FrameNum[i] {
				r.timeline.FrameNum = r.timeline.FrameNum[:i]
				r.timeline.TotalScanlines = r.timeline.TotalScanlines[:i]
				r.timeline.Counts = r.timeline.Counts[:i]
				r.timeline.LeftPlayerInput = r.timeline.LeftPlayerInput[:i]
				r.timeline.RightPlayerInput = r.timeline.RightPlayerInput[:i]
				r.timeline.PanelInput = r.timeline.PanelInput[:i]
				break // for loop
			}
		}
	}

	cts := TimelineCounts{}
	if r.ctr != nil {
		cts = r.ctr.TimelineCounts()
	}

	r.timeline.FrameNum = append(r.timeline.FrameNum, frameInfo.FrameNum)
	r.timeline.TotalScanlines = append(r.timeline.TotalScanlines, frameInfo.TotalScanlines)
	r.timeline.Counts = append(r.timeline.Counts, cts)
	r.timeline.LeftPlayerInput = append(r.timeline.LeftPlayerInput, r.vcs.RIOT.Ports.LeftPlayer.IsActive())
	r.timeline.RightPlayerInput = append(r.timeline.RightPlayerInput, r.vcs.RIOT.Ports.RightPlayer.IsActive())
	r.timeline.PanelInput = append(r.timeline.PanelInput, r.vcs.RIOT.Ports.Panel.IsActive())

	// ratios
	b := float32(frameInfo.TotalScanlines * specification.ClksScanline)
	if b == 0 {
		r.timeline.Ratios = append(r.timeline.Ratios, TimelineRatios{
			WSYNC:  0,
			CoProc: 0,
		})
	} else {
		r.timeline.Ratios = append(r.timeline.Ratios, TimelineRatios{
			WSYNC:  float32(cts.WSYNC) / b,
			CoProc: float32(cts.CoProc) / b,
		})
	}

	if len(r.timeline.FrameNum) > timelineLength {
		r.timeline.FrameNum = r.timeline.FrameNum[1:]
		r.timeline.TotalScanlines = r.timeline.TotalScanlines[1:]
		r.timeline.Counts = r.timeline.Counts[1:]
		r.timeline.Ratios = r.timeline.Ratios[1:]
		r.timeline.LeftPlayerInput = r.timeline.LeftPlayerInput[1:]
		r.timeline.RightPlayerInput = r.timeline.RightPlayerInput[1:]
		r.timeline.PanelInput = r.timeline.PanelInput[1:]
	}

}
