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

package debugger

// support functions for the rewind package that require more knowledge of
// the debugger than would otherwise be available.

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// RewindByAmount moves forwards or backwards by specified frames. Negative
// numbers indicate backwards
func (dbg *Debugger) RewindByAmount(amount int) bool {
	switch dbg.Mode() {
	default:
		panic(fmt.Sprintf("Rewind: unsupported mode (%v)", dbg.Mode()))

	case emulation.ModePlay:
		coords := dbg.vcs.TV.GetCoords()
		tl := dbg.Rewind.GetTimeline()

		if amount < 0 && coords.Frame-1 <= tl.AvailableStart {
			dbg.setStateQuiet(emulation.Paused, true)
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindAtStart)
			return false
		}

		if amount > 0 && coords.Frame+1 >= tl.AvailableEnd {
			dbg.setStateQuiet(emulation.Paused, true)
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindAtEnd)
			return false
		}

		dbg.setStateQuiet(emulation.Rewinding, true)
		dbg.Rewind.GotoFrame(coords.Frame + amount)
		dbg.setStateQuiet(emulation.Paused, true)

		if amount < 0 {
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindBack)
		} else {
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindFoward)
		}

		return true
	}

	return false
}

// RewindToFrame measure from the current frame.
func (dbg *Debugger) RewindToFrame(fn int, last bool) bool {
	switch dbg.Mode() {
	default:
		panic(fmt.Sprintf("RewindToFrame: unsupported mode (%v)", dbg.Mode()))

	case emulation.ModeDebugger:
		if dbg.State() == emulation.Rewinding {
			return false
		}

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			// upate catchupQuantum before starting rewind process
			dbg.catchupQuantum = dbg.stepQuantum

			if last {
				err := dbg.Rewind.GotoLast()
				if err != nil {
					return curated.Errorf("push goto last: %v", err)
				}
			} else {
				err := dbg.Rewind.GotoFrame(fn)
				if err != nil {
					return curated.Errorf("push goto frame: %v", err)
				}
			}

			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushRawEventReturn(func() {
			// set state to emulation.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)
			dbg.setState(emulation.Rewinding)

			dbg.unwindLoop(doRewind)
		})

		return true
	}

	return false
}

// GotoCoords rewinds the emulation to the specified coordinates.
func (dbg *Debugger) GotoCoords(coords coords.TelevisionCoords) bool {
	switch dbg.Mode() {
	default:
		panic(fmt.Sprintf("GotoCoords: unsupported mode (%v)", dbg.Mode()))

	case emulation.ModeDebugger:
		if dbg.State() == emulation.Rewinding {
			return false
		}

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			// upate catchupQuantum before starting rewind process
			dbg.catchupQuantum = QuantumClock

			err := dbg.Rewind.GotoCoords(coords)
			if err != nil {
				return curated.Errorf("push goto coords: %v", err)
			}
			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushRawEventReturn(func() {
			// set state to emulation.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)
			dbg.setState(emulation.Rewinding)

			dbg.unwindLoop(doRewind)
		})

		return true
	}

	return false
}

// RerunLastNFrames measured from the current frame.
func (dbg *Debugger) RerunLastNFrames(frames int) bool {
	switch dbg.Mode() {
	default:
		panic(fmt.Sprintf("RerunLastNFrames: unsupported mode (%v)", dbg.Mode()))

	case emulation.ModeDebugger:
		if dbg.State() == emulation.Rewinding {
			return false
		}

		// the disadvantage of RerunLastNFrames() is that it will always land on a
		// CPU instruction boundary (this is because we must unwind the existing
		// input loop before calling the rewind function)
		//
		// if we're in between instruction boundaries therefore we need to push a
		// GotoCoords() request. get the current coordinates now
		correctCoords := !dbg.liveDisasmEntry.Result.Final
		coords := dbg.vcs.TV.GetCoords()

		// the function to push to the debugger/emulation routine
		doRewind := func() error {
			err := dbg.Rewind.RerunLastNFrames(frames)
			if err != nil {
				return curated.Errorf("push rerun last N Frame: %v", err)
			}

			if correctCoords {
				err = dbg.Rewind.GotoCoords(coords)
				if err != nil {
					return curated.Errorf("push rerun last N Frame: %v", err)
				}
			}

			return nil
		}

		// how we push the doRewind() function depends on what kind of inputloop we
		// are currently in
		dbg.PushRawEventReturn(func() {
			// upate catchupQuantum before starting rewind process
			dbg.catchupQuantum = QuantumClock

			// set state to emulation.Rewinding as soon as possible (but
			// remembering that we must do it in the debugger goroutine)
			dbg.setState(emulation.Rewinding)
			dbg.unwindLoop(doRewind)
		})

		return true
	}

	return false
}
