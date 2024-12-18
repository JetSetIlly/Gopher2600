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

import (
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/userinput"
)

func (dbg *Debugger) playLoop() error {
	// if forcedROMSelection is active (is not nil) then the event loop is
	// simplified for the duration (until it is set to nil)
	for dbg.forcedROMselection != nil {
		select {
		case sig := <-dbg.events.Signal:
			return dbg.events.SignalHandler(sig)
		case ev := <-dbg.events.PushedFunction:
			ev()
		case ev := <-dbg.events.PushedFunctionImmediate:
			ev()
			return nil
		case ev := <-dbg.events.UserInput:
			if _, ok := ev.(userinput.EventQuit); ok {
				return terminal.UserQuit
			}
		case <-dbg.forcedROMselection:
			dbg.forcedROMselection = nil
		}
	}

	// only check for end of measurement period every PerformanceBrake CPU
	// instructions
	performanceBrake := 0

	// update lastBank at the start of the play loop
	dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

	// run and handle events
	return dbg.vcs.Run(func() (govern.State, error) {

		// update counters. because of the way LastResult works we need to make
		// sure we only use it in the event that the CPU RdyFlag is set
		//
		// if it isn't then we know that the CPU ticked once before returning
		// and allowing this function to run. meaning the the number of cycles
		// tell the counter to Step() is ColorClocksPerCPUCycle
		//
		// in all other cases the number of cycles to count is ColorClocksPerCPUCycle
		// multiplied by the number of cycles in the instruction
		if dbg.vcs.CPU.RdyFlg {
			dbg.counter.Step(dbg.vcs.CPU.LastResult.Cycles*hardware.ColorClocksPerCPUCycle, dbg.liveBankInfo)
		} else {
			dbg.counter.Step(hardware.ColorClocksPerCPUCycle, dbg.liveBankInfo)
		}

		// we must keep lastBank updated during the play loop
		dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

		// record state. we do this before any of the conditions below that may
		// result in an early return from the function
		if dbg.state.Load().(govern.State) == govern.Running {
			dbg.Rewind.RecordState()
		}

		// run continueCheck() function is called every CPU instruction. for
		// some halt conditions this is too infrequent
		//
		// for this reason we should never find ourselves in the playLoop if
		// these halt conditions exist. see setMode() function
		dbg.halting.check()
		if dbg.halting.halt {
			// set debugging mode. halting messages will be preserved and
			// shown when entering debugging mode
			dbg.setMode(govern.ModeDebugger)
			return govern.Ending, nil
		}

		if dbg.Mode() != govern.ModePlay {
			return govern.Ending, nil
		}

		// return without checking interface unless we exceed the
		// PerformanceBrake value
		performanceBrake++
		if performanceBrake < hardware.PerformanceBrake {
			return dbg.State(), nil
		}
		performanceBrake = 0

		// how we wait for read events depends on whether the emulation is
		// paused or not. if emulation is paused then we halt until an event is
		// received. this reduces CPU usage when paused
		if dbg.state.Load().(govern.State) == govern.Paused {
			select {
			case <-dbg.readEventsPulse.C:
				err := dbg.readEventsHandler()
				if err != nil {
					return govern.Ending, err
				}
			}
		} else {
			select {
			case <-dbg.readEventsPulse.C:
				err := dbg.readEventsHandler()
				if err != nil {
					return govern.Ending, err
				}
			default:
			}
		}

		// resolve rewinding
		if dbg.rewindKeyboardAccumulation != 0 {
			amount := 0
			if dbg.rewindKeyboardAccumulation < 0 {
				if dbg.rewindKeyboardAccumulation > -100 {
					dbg.rewindKeyboardAccumulation--
				}
				amount = (dbg.rewindKeyboardAccumulation / 10) - 1
			} else {
				if dbg.rewindKeyboardAccumulation < 100 {
					dbg.rewindKeyboardAccumulation++
				}
				amount = (dbg.rewindKeyboardAccumulation / 10) + 1
			}
			dbg.RewindByAmount(amount)
		}

		return dbg.State(), nil
	})
}
