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

// Calls measures the number of times a function has been called in each VCS
// scope. It only makes sense for this type to be used in the context of
// functions
type Calls struct {
	Overall  CallsScope
	VBLANK   CallsScope
	Screen   CallsScope
	Overscan CallsScope
}

// Reset the call counts to zero
func (cl *Calls) Reset() {
	cl.Overall.reset()
	cl.VBLANK.reset()
	cl.Screen.reset()
	cl.Overscan.reset()
}

// Call registers a new instance of the function being called
func (cl *Calls) Call(focus Focus) {
	switch focus {
	case FocusAll:
		cl.Overall.call()
	case FocusVBLANK:
		cl.VBLANK.call()
		cl.Overall.call()
	case FocusScreen:
		cl.Screen.call()
		cl.Overall.call()
	case FocusOverscan:
		cl.Overscan.call()
		cl.Overall.call()
	}
}

// Check is like call except that it only makes sure that the call figure is at
// least one. It's useful to make sure a function has been called at least
// once if it is part of the call stack
func (cl *Calls) Check(focus Focus) {
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

// NewFrame commits accumulated data for the frame. The rewinding flag
// indicates that the emulation is in the rewinding state and that some
// data should not be updated
func (cl *Calls) NewFrame(rewinding bool) {
	cl.Overall.newFrame(rewinding)
	cl.VBLANK.newFrame(rewinding)
	cl.Screen.newFrame(rewinding)
	cl.Overscan.newFrame(rewinding)
}

// CallsScope records the number of times the entity being measured has been "hit".
// For a function this equates to the number of times it has been called.
//
// Like the Cycles type, the CallsScope type records figures for the most recent and
// for the average and maximum cases
type CallsScope struct {
	FrameCount   float32
	AverageCount float32
	MaxCount     float32

	// call count this frame
	calls float32

	// call count over all frames
	totalCalls float32

	// number of frames seen
	numFrames float32
}

func (cl *CallsScope) reset() {
	cl.FrameCount = 0.0
	cl.AverageCount = 0.0
	cl.MaxCount = 0.0
	cl.calls = 0.0
	cl.totalCalls = 0.0
	cl.numFrames = 0.0
}

func (cl *CallsScope) newFrame(rewinding bool) {
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

func (cl *CallsScope) call() {
	cl.calls++
}

func (cl *CallsScope) check() {
	if cl.calls == 0.0 {
		cl.calls = 1.0
	}
}
