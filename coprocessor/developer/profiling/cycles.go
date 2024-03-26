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

type Cycles struct {
	Overall  CyclesScope
	VBLANK   CyclesScope
	Screen   CyclesScope
	Overscan CyclesScope
}

func (cy *Cycles) Reset() {
	cy.Overall.reset()
	cy.VBLANK.reset()
	cy.Screen.reset()
	cy.Overscan.reset()
}

// NewFrame commits accumulated statistics for the frame. The rewinding flag
// indicates that the emulation is in the rewinding state and that some
// statistics should not be updated
//
// The programCycles and functionCycles parameters reprenent the parents of the
// entity being measured by Cycles instance.
//
// For dwarf.SourceLines the programCycles parameter will point to the Cycles
// instance for the entire program; and the functionCycles parameter will point
// to the Cycles instance for the function the line is part of
//
// For dwarf.SourceFunction the functionCycles parameter will be nil.
//
// For dwarf.Source both parameters will be nil
func (cy *Cycles) NewFrame(programCycles *Cycles, functionCycles *Cycles, rewinding bool) {
	if programCycles == nil {
		cy.Overall.newFrame(nil, nil, rewinding)
		cy.VBLANK.newFrame(nil, nil, rewinding)
		cy.Screen.newFrame(nil, nil, rewinding)
		cy.Overscan.newFrame(nil, nil, rewinding)
		return
	}

	if functionCycles == nil {
		cy.Overall.newFrame(&programCycles.Overall, nil, rewinding)
		cy.VBLANK.newFrame(&programCycles.VBLANK, nil, rewinding)
		cy.Screen.newFrame(&programCycles.Screen, nil, rewinding)
		cy.Overscan.newFrame(&programCycles.Overscan, nil, rewinding)
		return
	}

	cy.Overall.newFrame(&programCycles.Overall, &functionCycles.Overall, rewinding)
	cy.VBLANK.newFrame(&programCycles.VBLANK, &functionCycles.VBLANK, rewinding)
	cy.Screen.newFrame(&programCycles.Screen, &functionCycles.Screen, rewinding)
	cy.Overscan.newFrame(&programCycles.Overscan, &functionCycles.Overscan, rewinding)
}

// CyclesScope records the cycle count over time and can be used to the frame
// (or current) load as well as average and maximum load.
//
// The actual percentage values are accessed through the OverProgram and
// OverFunction fields. These fields provide the necessary scale by which
// the load is measured.
//
// The validity of the CycleScope fields depends on context. For instance, for
// the SourceFunction type, the CyclesFunction field is invalid. For the Source
// type meanwhile, neither field is valid.
//
// For the SourceLine type however, both CycleScopes can be used to provide a
// different scaling to the load values.
type CyclesScope struct {
	// number of cycles in relation to function and to the entire program
	CyclesFunction CycleFigures
	CyclesProgram  CycleFigures

	// cycle count this frame
	cycles float32

	// cycle count over all frames
	totalCycles float32

	// number of frames seen
	numFrames float32

	// frame and average count form the basis of the frame, average and max
	// counts (and percentages) in the Cycles type
	frameCount float32
	avgCount   float32
}

// Cycle advances the number of cycles for the current frame
func (cy *CyclesScope) Cycle(n float32) {
	cy.cycles += n
}

// HasExecuted returns true if the entity (program, function or line) has ever
// been executed
func (cy *CyclesScope) HasExecuted() bool {
	return cy.totalCycles > 0
}

func (cy *CyclesScope) newFrame(programCycles *CyclesScope, functionCycles *CyclesScope, rewinding bool) {
	if !rewinding {
		cy.numFrames++
		if cy.numFrames > 1 {
			if cy.cycles > 0 {
				cy.totalCycles += cy.cycles
				cy.avgCount = cy.totalCycles / (cy.numFrames - 1)
			}
		}
	}

	cy.frameCount = cy.cycles

	if functionCycles != nil {
		frameLoad := cy.frameCount / functionCycles.frameCount * 100

		cy.CyclesFunction.FrameCount = cy.frameCount
		cy.CyclesFunction.FrameLoad = frameLoad

		cy.CyclesFunction.AverageCount = cy.avgCount
		cy.CyclesFunction.AverageLoad = cy.avgCount / functionCycles.avgCount * 100

		cy.CyclesFunction.FrameValid = cy.frameCount > 0 && functionCycles.frameCount > 0

		if cy.CyclesFunction.FrameValid {
			if cy.frameCount >= cy.CyclesFunction.MaxCount {
				cy.CyclesFunction.MaxCount = cy.frameCount
			}
			if frameLoad >= cy.CyclesFunction.MaxLoad {
				cy.CyclesFunction.MaxLoad = frameLoad
			}
		}

		cy.CyclesFunction.AverageValid = cy.avgCount > 0 && functionCycles.avgCount > 0
		cy.CyclesFunction.MaxValid = cy.CyclesFunction.MaxValid || cy.CyclesFunction.FrameValid
	}

	if programCycles != nil {
		frameLoad := cy.frameCount / programCycles.frameCount * 100

		cy.CyclesProgram.FrameCount = cy.frameCount
		cy.CyclesProgram.FrameLoad = frameLoad

		cy.CyclesProgram.AverageCount = cy.avgCount
		cy.CyclesProgram.AverageLoad = cy.avgCount / programCycles.avgCount * 100

		cy.CyclesProgram.FrameValid = cy.frameCount > 0 && programCycles.frameCount > 0

		if cy.CyclesProgram.FrameValid {
			if cy.frameCount >= cy.CyclesProgram.MaxCount {
				cy.CyclesProgram.MaxCount = cy.frameCount
			}
			if frameLoad >= cy.CyclesProgram.MaxLoad {
				cy.CyclesProgram.MaxLoad = frameLoad
			}
		}

		cy.CyclesProgram.AverageValid = cy.avgCount > 0 && programCycles.avgCount > 0
		cy.CyclesProgram.MaxValid = cy.CyclesProgram.MaxValid || cy.CyclesProgram.FrameValid
	}

	// reset for next frame
	cy.cycles = 0.0
}

func (cy *CyclesScope) reset() {
	cy.CyclesProgram.reset()
	cy.CyclesFunction.reset()
	cy.totalCycles = 0.0
	cy.numFrames = 0.0
	cy.avgCount = 0.0
	cy.frameCount = 0.0
	cy.cycles = 0.0
	cy.numFrames = 0
}

// CycleFigures records the number of cycles for the entity being measured, over the
// course of the most recent frame and the average & maximum number of cycles
// over all frames
type CycleFigures struct {
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

func (cy *CycleFigures) reset() {
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
