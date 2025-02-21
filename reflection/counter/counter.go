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

package counter

import (
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/rewind"
)

// Counter inspects hardware.VCS and counts for how long a state has been
// active/or in active.
type Counter struct {
	vcs            *hardware.VCS
	counts         rewind.TimelineCounts
	mostRecentStep coords.TelevisionCoords
}

// NewCounter is the preferred method of implementation for the Counter type.
func NewCounter(vcs *hardware.VCS) *Counter {
	return &Counter{vcs: vcs}
}

// Accumulate count values at end of CPU cycle. The clocks argument says how
// many color clocks the Step() represents. Count values will be increased by
// that amount.
func (ct *Counter) Step(clocks int, bank mapper.BankInfo) {
	t := ct.vcs.TV.GetCoords()
	if coords.GreaterThan(t, ct.mostRecentStep) {
		if !ct.vcs.CPU.RdyFlg {
			ct.counts.WSYNC += clocks
		}
		if bank.ExecutingCoprocessor {
			ct.counts.CoProc += clocks
		}
		ct.mostRecentStep = t
	}
}

// Clear count data.
func (ct *Counter) Clear() {
	ct.counts = rewind.TimelineCounts{}
	ct.mostRecentStep = coords.TelevisionCoords{}
}

// TimelineCounts implements the rewind.TimelineCounter interface.
func (ct *Counter) TimelineCounts() rewind.TimelineCounts {
	return ct.counts
}

// NewFrame implements the television.FrameTrigger interface.
func (ct *Counter) NewFrame(info frameinfo.Current) error {
	ct.Clear()
	return nil
}
