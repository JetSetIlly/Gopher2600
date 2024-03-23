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
type CallStats struct {
	// cycle statistics for the entire program
	Overall Calls

	// kernel specific cycle statistics for the program. accumulated only once TV is stable
	VBLANK   Calls
	Screen   Calls
	Overscan Calls
}

func (cl *CallStats) Reset() {
	cl.Overall.reset()
	cl.VBLANK.reset()
	cl.Screen.reset()
	cl.Overscan.reset()
}

func (cl *CallStats) Call(focus Focus) {
	cl.Overall.call()

	switch focus {
	case FocusAll:
		// deliberately ignore
	case FocusVBLANK:
		cl.VBLANK.call()
	case FocusScreen:
		cl.Screen.call()
	case FocusOverscan:
		cl.Overscan.call()
	}
}

// Check is like call except that it only makes sure that the call figure is at
// least one. It's useful to make sure a function has been registered at least
// once if it is part of the call stack
//
// It's a bit of a hack and this problem should probably be solved in a
// different way
func (cl *CallStats) Check(focus Focus) {
	cl.Overall.check()

	switch focus {
	case FocusAll:
		// deliberately ignore
	case FocusVBLANK:
		cl.VBLANK.check()
	case FocusScreen:
		cl.Screen.check()
	case FocusOverscan:
		cl.Overscan.check()
	}
}

// NewFrame commits accumulated statistics for the frame. The rewinding flag
// indicates that the emulation is in the rewinding state and that some
// statistics should not be updated
func (cl *CallStats) NewFrame(rewinding bool) {
	cl.Overall.newFrame(rewinding)
	cl.VBLANK.newFrame(rewinding)
	cl.Screen.newFrame(rewinding)
	cl.Overscan.newFrame(rewinding)
}

// Calls records the number of times the entity being measured has been "hit".
// For a function this equates to the number of times it has been called.
//
// For a source line, it means the number of times the source line has been
// reached by the program counter. However, it much compiled code, this can be a
// misleading statistics due to how instructions are interleaved
//
// Like the Cycles type, the Calls type records figures for the most recent and
// for the average and maximum cases
type Calls struct {
	FrameCount   float32
	AverageCount float32
	MaxCount     float32

	calls      float32
	totalCalls float32
	numFrames  float32
}

func (cl *Calls) reset() {
	cl.FrameCount = 0.0
	cl.AverageCount = 0.0
	cl.MaxCount = 0.0
	cl.calls = 0.0
	cl.totalCalls = 0.0
	cl.numFrames = 0.0
}

func (cl *Calls) newFrame(rewinding bool) {
	if !rewinding {
		cl.totalCalls += cl.calls
		cl.numFrames++
	}

	cl.FrameCount = cl.calls
	cl.AverageCount = cl.totalCalls / cl.numFrames

	if cl.calls > cl.MaxCount {
		cl.MaxCount = cl.calls
	}

	// reset for next frame
	cl.calls = 0.0
}

func (cl *Calls) call() {
	cl.calls++
}

// see documentation for Check() in the CallStats type
func (cl *Calls) check() {
	if cl.calls == 0.0 {
		cl.calls = 1.0
	}
}
