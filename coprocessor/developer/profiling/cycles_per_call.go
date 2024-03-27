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

// CyclesPerCall measures the number of cycles consumed by a function divided by
// the number of times its been called, in each VCS scope. It only makes sense
// for this type to be used in the context of functions
type CyclesPerCall struct {
	Overall  CyclesPerCallScope
	VBLANK   CyclesPerCallScope
	Screen   CyclesPerCallScope
	Overscan CyclesPerCallScope
}

// Reset the counts to zero
func (cl *CyclesPerCall) Reset() {
	cl.Overall.reset()
	cl.VBLANK.reset()
	cl.Screen.reset()
	cl.Overscan.reset()
}

// Call registers a new instance of the function being called
func (cl *CyclesPerCall) Call(focus Focus) {
	switch focus {
	case FocusAll:
		cl.Overall.call()
	case FocusVBLANK:
		cl.Overall.call()
		cl.VBLANK.call()
	case FocusScreen:
		cl.Overall.call()
		cl.Screen.call()
	case FocusOverscan:
		cl.Overall.call()
		cl.Overscan.call()
	}
}

// Check is like call except that it only makes sure that the call figure is at
// least one. It's useful to make sure a function has been called at least
// once if it is part of the call stack
func (cl *CyclesPerCall) Check(focus Focus) {
	switch focus {
	case FocusAll:
		cl.Overall.check()
	case FocusVBLANK:
		cl.Overall.check()
		cl.VBLANK.check()
	case FocusScreen:
		cl.Overall.check()
		cl.Screen.check()
	case FocusOverscan:
		cl.Overall.check()
		cl.Overscan.check()
	}
}

// Cycle advances the number of cycles for the VCS scope
func (cl *CyclesPerCall) Cycle(n float32, focus Focus) {
	switch focus {
	case FocusAll:
		cl.Overall.cycle(n)
	case FocusVBLANK:
		cl.Overall.cycle(n)
		cl.VBLANK.cycle(n)
	case FocusScreen:
		cl.Overall.cycle(n)
		cl.Screen.cycle(n)
	case FocusOverscan:
		cl.Overall.cycle(n)
		cl.Overscan.cycle(n)
	}
}

// NewFrame commits accumulated cycles and calls for the frame. The rewinding
// flag indicates that the emulation is in the rewinding state and that some
// data should not be updated
func (cl *CyclesPerCall) NewFrame(rewinding bool) {
	cl.Overall.newFrame(rewinding)
	cl.VBLANK.newFrame(rewinding)
	cl.Screen.newFrame(rewinding)
	cl.Overscan.newFrame(rewinding)
}

type CyclesPerCallScope struct {
	FrameCount   float32
	AverageCount float32
	MaxCount     float32

	// whether the corresponding values are valid
	FrameValid   bool
	AverageValid bool
	MaxValid     bool

	// cycle and call count this frame
	cycles float32
	calls  float32

	// number of frames seen
	numFrames float32

	// cycle and call count over all frames
	totalCycles float32
	totalCalls  float32
}

func (cl *CyclesPerCallScope) call() {
	cl.calls++
}

func (cl *CyclesPerCallScope) check() {
	if cl.calls == 0.0 {
		cl.calls = 1.0
	}
}

func (cl *CyclesPerCallScope) cycle(n float32) {
	cl.cycles += n
}

func (cl *CyclesPerCallScope) reset() {
	cl.FrameCount = 0.0
	cl.AverageCount = 0.0
	cl.MaxCount = 0.0
	cl.FrameValid = false
	cl.AverageValid = false
	cl.MaxValid = false
	cl.cycles = 0.0
	cl.calls = 0.0
	cl.numFrames = 0.0
	cl.totalCycles = 0.0
	cl.totalCalls = 0.0
}

func (cl *CyclesPerCallScope) newFrame(rewinding bool) {
	if !rewinding {
		cl.totalCycles += cl.cycles
		cl.totalCalls += cl.calls
		cl.numFrames++
	}

	if cl.cycles > 0.0 && cl.calls > 0.0 {
		cl.FrameCount = cl.cycles / cl.calls
		cl.FrameValid = true
	} else {
		cl.FrameCount = 0
		cl.FrameValid = false
	}

	if cl.totalCycles > 0.0 && cl.totalCalls > 0.0 && cl.numFrames > 0.0 {
		cl.AverageCount = cl.totalCycles / cl.totalCalls
		cl.AverageValid = true
	} else {
		cl.AverageCount = 0.0
		cl.AverageValid = false
	}

	if cl.FrameCount > cl.MaxCount {
		cl.MaxCount = cl.FrameCount
		cl.MaxValid = cl.FrameValid
	}

	// reset for next frame
	cl.cycles = 0.0
	cl.calls = 0.0
}
