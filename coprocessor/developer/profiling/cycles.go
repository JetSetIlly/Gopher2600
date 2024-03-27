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

// Cycles measures the number of cycles consumed in each VCS scope
type Cycles struct {
	Overall  CyclesScope
	VBLANK   CyclesScope
	Screen   CyclesScope
	Overscan CyclesScope
}

// Reset the cycles counts to zero
func (cy *Cycles) Reset() {
	cy.Overall.reset()
	cy.VBLANK.reset()
	cy.Screen.reset()
	cy.Overscan.reset()
}

// Cycle advances the number of cycles for the VCS scope
func (cy *Cycles) Cycle(n float32, focus Focus) {
	switch focus {
	case FocusAll:
		cy.Overall.Cycle(n)
	case FocusVBLANK:
		cy.Overall.Cycle(n)
		cy.VBLANK.Cycle(n)
	case FocusScreen:
		cy.Overall.Cycle(n)
		cy.Screen.Cycle(n)
	case FocusOverscan:
		cy.Overall.Cycle(n)
		cy.Overscan.Cycle(n)
	}
}

// NewFrame commits accumulated cycles for the frame. The rewinding flag
// indicates that the emulation is in the rewinding state and that some data
// should not be updated
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

	// frameCount and averageCount are used by CyclesFigures during calculation
	//
	// they are in fact the same values as the FrameCount fields in CyclesFunction
	// and CyclesProgram but those figures won't always be updated
	frameCount   float32
	averageCount float32
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
		cy.totalCycles += cy.cycles
		cy.numFrames++

		cy.frameCount = cy.cycles
		if cy.numFrames > 0 && cy.cycles > 0 {
			cy.averageCount = cy.totalCycles / cy.numFrames
		}
	}

	// accumulate figures for function and program scopes
	cy.CyclesFunction.newFrame(functionCycles, cy.frameCount, cy.averageCount)
	cy.CyclesProgram.newFrame(programCycles, cy.frameCount, cy.averageCount)

	// reset for next frame
	cy.cycles = 0.0
}

func (cy *CyclesScope) reset() {
	cy.CyclesProgram.reset()
	cy.CyclesFunction.reset()
	cy.totalCycles = 0.0
	cy.numFrames = 0.0
	cy.averageCount = 0.0
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

func (cy *CycleFigures) newFrame(parent *CyclesScope, frameCount float32, averageCount float32) {
	if parent == nil {
		return
	}

	if frameCount > 0.0 && parent.frameCount > 0.0 {
		cy.FrameCount = frameCount
		cy.FrameLoad = cy.FrameCount / parent.frameCount * 100
		cy.FrameValid = true
	} else {
		cy.FrameCount = 0.0
		cy.FrameLoad = 0.0
		cy.FrameValid = false
	}

	if averageCount > 0 && parent.averageCount > 0 {
		cy.AverageCount = averageCount
		cy.AverageLoad = cy.AverageCount / parent.averageCount * 100
		cy.AverageValid = true
	} else {
		cy.AverageCount = 0.0
		cy.AverageLoad = 0.0
		cy.AverageValid = false
	}

	if cy.FrameValid {
		if cy.FrameCount >= cy.MaxCount {
			cy.MaxCount = cy.FrameCount
		}
		if cy.FrameLoad >= cy.MaxLoad {
			cy.MaxLoad = cy.FrameLoad
		}
		cy.MaxValid = true
	}
}
