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

package profiling

// CycleStats collates the Stats instance for all kernel views of the program.
type CycleStats struct {
	// cycle statistics for the entire program
	Overall Cycles

	// kernel specific cycle statistics for the program. accumulated only once TV is stable
	VBLANK   Cycles
	Screen   Cycles
	Overscan Cycles
}

// CycleLoad records the number of cycles for the entity being measured, over the
// course of the most recent frame and the average & maximum number of cycles
// over all frames
type CycleLoad struct {
	// cycle count
	FrameCount   float32
	AverageCount float32
	MaxCount     float32

	// cycle count expressed as a percentage
	FrameLoad   float32
	AverageLoad float32
	MaxLoad     float32

	// whether the corresponding values are valid
	FrameValid   bool
	AverageValid bool
	MaxValid     bool
}

func (cy *CycleLoad) reset() {
	cy.FrameCount = 0.0
	cy.AverageCount = 0.0
	cy.MaxCount = 0.0
	cy.FrameLoad = 0.0
	cy.AverageLoad = 0.0
	cy.MaxLoad = 0.0
	cy.FrameValid = false
	cy.AverageValid = false
	cy.MaxValid = false
}

// Cycles records the cycle count over time and can be used to the frame
// (or current) load as well as average and maximum load.
//
// The actual percentage values are accessed through the OverProgram and
// OverFunction fields. These fields provide the necessary scale by which
// the load is measured.
//
// The validity of the OverProgram and OverFunction fields depends on context.
// For instance, for the SourceFunction type, the corresponding OverFunction
// field is invalid. For the Source type meanwhile, neither field is valid.
//
// For the SourceLine type however, both OverProgram and OverFunction can be
// used to provide a different scaling to the load values.
type Cycles struct {
	// number of cycles in relation to function and to the entire program
	CyclesFunction CycleLoad
	CyclesProgram  CycleLoad

	// cycle count this frame
	CycleCount float32

	// cycle count over all frames
	allFrameCycleCount float32

	// number of frames seen
	numFrames float32

	// frame and average count form the basis of the frame, average and max
	// counts (and percentages) in the Cycles type
	frameCount float32
	avgCount   float32
}

// HasExecuted returns true if the statistics have ever been updated. ie. the
// source associated with this statistic has ever executed.
//
// Not to be confused with the FrameValid, AverageValid and MaxValid fields of
// the Cycles type.
func (cy *Cycles) HasExecuted() bool {
	return cy.allFrameCycleCount > 0
}

// NewFrame update statistics, using source and function to update the Cycles values as
// appropriate
//
// The rewinding flag indicates that the emulation is in the rewinding state and
// that some statistics should not be updated
func (cy *Cycles) NewFrame(byProg *Cycles, byFunc *Cycles, rewinding bool) {
	if !rewinding {
		cy.numFrames++
		if cy.numFrames > 1 {
			if cy.CycleCount > 0 {
				cy.allFrameCycleCount += cy.CycleCount
				cy.avgCount = cy.allFrameCycleCount / (cy.numFrames - 1)
			}
		}
	}

	cy.frameCount = cy.CycleCount

	if byFunc != nil {
		frameLoad := cy.frameCount / byFunc.frameCount * 100

		cy.CyclesFunction.FrameCount = cy.frameCount
		cy.CyclesFunction.FrameLoad = frameLoad

		cy.CyclesFunction.AverageCount = cy.avgCount
		cy.CyclesFunction.AverageLoad = cy.avgCount / byFunc.avgCount * 100

		cy.CyclesFunction.FrameValid = cy.frameCount > 0 && byFunc.frameCount > 0

		if cy.CyclesFunction.FrameValid {
			if cy.frameCount >= cy.CyclesFunction.MaxCount {
				cy.CyclesFunction.MaxCount = cy.frameCount
			}
			if frameLoad >= cy.CyclesFunction.MaxLoad {
				cy.CyclesFunction.MaxLoad = frameLoad
			}
		}

		cy.CyclesFunction.AverageValid = cy.avgCount > 0 && byFunc.avgCount > 0
		cy.CyclesFunction.MaxValid = cy.CyclesFunction.MaxValid || cy.CyclesFunction.FrameValid
	}

	if byProg != nil {
		frameLoad := cy.frameCount / byProg.frameCount * 100

		cy.CyclesProgram.FrameCount = cy.frameCount
		cy.CyclesProgram.FrameLoad = frameLoad

		cy.CyclesProgram.AverageCount = cy.avgCount
		cy.CyclesProgram.AverageLoad = cy.avgCount / byProg.avgCount * 100

		cy.CyclesProgram.FrameValid = cy.frameCount > 0 && byProg.frameCount > 0

		if cy.CyclesProgram.FrameValid {
			if cy.frameCount >= cy.CyclesProgram.MaxCount {
				cy.CyclesProgram.MaxCount = cy.frameCount
			}
			if frameLoad >= cy.CyclesProgram.MaxLoad {
				cy.CyclesProgram.MaxLoad = frameLoad
			}
		}

		cy.CyclesProgram.AverageValid = cy.avgCount > 0 && byProg.avgCount > 0
		cy.CyclesProgram.MaxValid = cy.CyclesProgram.MaxValid || cy.CyclesProgram.FrameValid
	}

	// reset for next frame
	cy.CycleCount = 0.0
}

func (cy *Cycles) Reset() {
	cy.CyclesProgram.reset()
	cy.CyclesFunction.reset()
	cy.allFrameCycleCount = 0.0
	cy.numFrames = 0.0
	cy.avgCount = 0.0
	cy.frameCount = 0.0
	cy.CycleCount = 0.0
	cy.numFrames = 0
}
