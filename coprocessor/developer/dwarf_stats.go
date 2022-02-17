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
	load      float32
	loadValid bool

	// the average load for this entity
	avgLoad      float32
	avgLoadValid bool

	// support fields for calculating the average
	cumulativeLoad float32
	numSteps       float32

	// the highest load value we've ever seen
	maxLoad      float32
	maxLoadValid bool

	// working value that will be assigned to FrameCycles on the next television.NewFrame()
	cyclesCount float32
}

func (stats *SourceStats) newFrame(allLoadCycles float32) {
	if stats.cyclesCount == 0 || allLoadCycles == 0 {
		stats.load = 0
		stats.loadValid = false
	} else {
		stats.load = stats.cyclesCount / allLoadCycles * 100.0
		stats.loadValid = true

		if stats.load > stats.maxLoad {
			stats.maxLoad = stats.load
			stats.maxLoadValid = true
		}

		stats.numSteps++
		stats.cumulativeLoad += stats.load
		stats.avgLoad = stats.cumulativeLoad / stats.numSteps
		stats.avgLoadValid = true
	}

	stats.cyclesCount = 0
}

// FrameLoad returns the load for this entity over the course of the most
// recent frame. Return false if load cannot be determined.
func (stats SourceStats) FrameLoad() (float32, bool) {
	return stats.load, stats.loadValid
}

// AverageLoad returns the average load for this entity over the course of the
// emulation's lifetime. Return false if load cannot be determined.
func (stats SourceStats) AverageLoad() (float32, bool) {
	return stats.avgLoad, stats.avgLoadValid
}

// AverageLoad returns the average load for this entity over the course of the
// emulation's lifetime. Return false if load cannot be determined.
func (stats SourceStats) MaximumLoad() (float32, bool) {
	return stats.maxLoad, stats.maxLoadValid
}
