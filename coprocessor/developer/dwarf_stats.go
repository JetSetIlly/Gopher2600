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

package developer

// CycleCounting statistics
type SourceStats struct {
	// the number of cycles this entity has consumed during the course of the previous frame
	FrameCycles float32

	// working value that will be assigned to FrameCycles on the next television.NewFrame()
	nextFrameCycles float32

	totalCycles float32
	numSteps    float32

	// the average number of cycles this entity has consumed in its lifetime
	AvgCycles float32
}

func (stats *SourceStats) Update() {
	stats.FrameCycles = stats.nextFrameCycles
	stats.nextFrameCycles = 0

	stats.totalCycles += stats.FrameCycles
	stats.numSteps++

	stats.AvgCycles = stats.totalCycles / stats.numSteps
}

// FrameLoad returns the load for this entity over the course of the most
// recent frame. Return false if load cannot be determined.
func (stats SourceStats) FrameLoad(src *Source) (float32, bool) {
	if stats.FrameCycles == 0 || src.Stats.FrameCycles == 0 {
		return 0.0, false
	}
	return stats.FrameCycles / src.Stats.FrameCycles * 100.0, true
}

// AverageLoad returns the average load for this entity over the course of the
// emulation's lifetime. Return false if load cannot be determined.
func (stats SourceStats) AverageLoad(src *Source) (float32, bool) {
	if stats.AvgCycles == 0 || src.Stats.FrameCycles == 0 {
		return 0.0, false
	}
	return stats.AvgCycles / src.Stats.FrameCycles * 100.0, true
}
