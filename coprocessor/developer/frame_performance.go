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

import (
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type FrameSegmentStats struct {
	// counts expressed as VCS clocks
	ClocksCount  float32
	AverageCount float32
	MaxCount     float32

	// counts expressed as percentages
	Clocks  float32
	Average float32
	Max     float32

	count           float32
	cumulativeCount float32
}

// HasRun returns false if no coprocessor activity has been seen in the segment
func (seg *FrameSegmentStats) HasRun() bool {
	return seg.cumulativeCount > 0
}

func (seg *FrameSegmentStats) reset() {
	seg.ClocksCount = 0.0
	seg.AverageCount = 0.0
	seg.MaxCount = 0.0

	seg.Clocks = 0.0
	seg.Average = 0.0
	seg.Max = 0.0

	seg.count = 0.0
	seg.cumulativeCount = 0.0
}

func (seg *FrameSegmentStats) accumulate(clocks int) {
	seg.count += float32(clocks)
}

func (seg *FrameSegmentStats) newFrame(numFrames int, totalClocks int) {
	seg.ClocksCount = seg.count
	seg.Clocks = float32(seg.ClocksCount) / float32(totalClocks) * 100

	if numFrames > 1 {
		if seg.count > 0 {
			seg.cumulativeCount += seg.count
			seg.AverageCount = seg.cumulativeCount / float32(numFrames-1)
			seg.Average = float32(seg.AverageCount) / float32(totalClocks) * 100
		}
	}

	if seg.ClocksCount > seg.MaxCount {
		seg.MaxCount = seg.ClocksCount
		seg.Max = seg.Clocks
	}

	seg.count = 0
}

// FrameStats measures coprocessor performance against the TV frame.
//
// InKernel value of InROMSetup is not tracked by the FrameStats struct.
type FrameStats struct {
	FrameInfo television.FrameInfo

	Frame    FrameSegmentStats
	VBLANK   FrameSegmentStats
	Screen   FrameSegmentStats
	Overscan FrameSegmentStats

	numFrames int
}

// accumulate() should not be called for InKernel value of InROMSetup
func (stats *FrameStats) accumulate(clocks int, kernel KernelVCS) {
	stats.Frame.accumulate(clocks)
	switch kernel {
	case KernelVBLANK:
		stats.VBLANK.accumulate(clocks)
	case KernelScreen:
		stats.Screen.accumulate(clocks)
	case KernelOverscan:
		stats.Overscan.accumulate(clocks)
	default:
	}
}

// newFrame() should not be called for InKernel value of InROMSetup
func (stats *FrameStats) newFrame(frameInfo television.FrameInfo) {
	stats.FrameInfo = frameInfo
	stats.numFrames++
	totalClocks := frameInfo.TotalClocks()
	stats.Frame.newFrame(stats.numFrames, totalClocks)
	stats.VBLANK.newFrame(stats.numFrames, totalClocks)
	stats.Screen.newFrame(stats.numFrames, totalClocks)
	stats.Overscan.newFrame(stats.numFrames, totalClocks)
}

// Reset frame statistics.
func (stats *FrameStats) Reset() {
	stats.Frame.reset()
	stats.VBLANK.reset()
	stats.Screen.reset()
	stats.Overscan.reset()
	stats.numFrames = 0
}
