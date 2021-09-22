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

	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/logger"
)

// CatchupLoop is an implementation of the rewind.Runner interface.
//
// Runs the emulation from it's current state until the supplied continueCheck
// callback function returns false.
func (dbg *Debugger) CatchUpLoop(continueCheck func() bool) error {
	var err error

	dbg.lastBank = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())
	dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
	if err != nil {
		return err
	}

	// the function to call every video step (via vcs.Step())
	var onStep func() error

	switch dbg.quantum {
	case QuantumInstruction:
		onStep = func() error {
			return dbg.ref.OnVideoCycle(dbg.lastBank)
		}
	case QuantumVideo:
		onStep = func() error {
			if dbg.quantum == QuantumVideo {
				dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
				if err != nil {
					return nil
				}
			}
			return dbg.ref.OnVideoCycle(dbg.lastBank)
		}
	default:
		return (fmt.Errorf("rewind: unknown quantum mode"))
	}

	for continueCheck() {
		err = dbg.vcs.Step(onStep)
		if err != nil {
			return err
		}

		err = dbg.ref.OnInstructionEnd(dbg.lastBank)
		if err != nil {
			return err
		}

		dbg.lastBank = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}
	}

	return nil
}

// PushRewind is a special case of PushRawEvent(). It prevents too many pushed
// rewind.Goto*() function calls. Returns false if the rewind hasn't been
// pushed. The caller should try again.
//
// To be used from the GUI thread.
func (dbg *Debugger) PushRewind(fn int, last bool) bool {
	// try pushing to the rewinding channel.
	//
	// if we cannot then that means a rewind is currently taking place and we
	// return false to indicate that the requested rewind has not taken place
	// yet.
	select {
	case dbg.rewinding <- true:
	default:
		return false
	}

	// the function to push to the debugger/emulation routine
	doRewind := func() error {
		if last {
			err := dbg.Rewind.GotoLast()
			if err != nil {
				return err
			}
		} else {
			err := dbg.Rewind.GotoFrame(fn)
			if err != nil {
				return err
			}
		}

		dbg.runUntilHalt = false

		return nil
	}

	// how we push the doRewind() function depends on what kind of inputloop we
	// are currently in
	dbg.PushRawEvent(func() {
		state := dbg.State()
		dbg.setState(emulation.Rewinding)
		defer func() {
			dbg.setState(state)
		}()

		if dbg.isClockCycleInputLoop {
			dbg.unwindInputLoop(doRewind)

			// read rewinding channel, this unblocks the channel and allows
			// calls to PushRewind() run to completion
			select {
			case <-dbg.rewinding:
			default:
			}
		} else {
			err := doRewind()
			if err != nil {
				logger.Log("rewind", err.Error())
			}

			// read rewinding channel, this unblocks the channel and allows
			// calls to PushRewind() run to completion
			select {
			case <-dbg.rewinding:
			default:
			}
		}
	})

	return true
}

// PushGotoCoords is a special case of PushRawEvent(). It prevents too may
// pushed rewind.GotoFrameCoords() function calls.
//
// To be used from the GUI thread.
func (dbg *Debugger) PushGotoCoords(frame int, scanline int, clock int) {
	// try pushing to rewinding channel. do not continue if we cannot.
	//
	// unlike PushRewind() no indicator of success is returned. the request is
	// just dropped.
	select {
	case dbg.rewinding <- true:
	default:
		return
	}

	dbg.PushRawEventImm(func() {
		state := dbg.State()
		dbg.setState(emulation.Rewinding)
		defer func() {
			dbg.setState(state)
		}()

		f := func() error {
			err := dbg.Rewind.GotoFrameCoords(frame, scanline, clock)
			if err != nil {
				return err
			}

			dbg.runUntilHalt = false

			return nil
		}

		dbg.unwindInputLoop(f)

		select {
		case <-dbg.rewinding:
		default:
		}
	})
}
