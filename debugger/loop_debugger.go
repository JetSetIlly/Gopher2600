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
	"errors"
	"io"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/logger"
)

// unwindLoop is called whenever it is required that the inputLoop/catchupLoop
// converge on the next instruction boundary. in other words, if the emulation
// is between CPU instructions, the loops must be unwound so that the emulation
// can continue safely.
//
// generally, this means that unwindLoop should be called whenever a rewind
// function is being called.
//
// note that the debugger state is not changed by this function. it is up to
// the caller of the function to set govern.State appropriately.
func (dbg *Debugger) unwindLoop(onRestart func() error) {
	dbg.unwindLoopRestart = onRestart
}

// catchupLoop is a special purpose loop designed to run inside of the inputLoop. it is called only
// when catchupContinue has been set in CatchUpLoop(), which is called as a consequence of a rewind event.
func (dbg *Debugger) catchupLoop(inputter terminal.Input) error {
	var ended bool

	callback := func(isCycle bool) error {
		if dbg.unwindLoopRestart != nil {
			return nil
		}

		// not updating disassembly. the halt condition in the inputLoop will
		// give us an opportunity to update *even if the catchupLoop is still
		// executing*

		// we do need to update the reflection however
		err := dbg.ref.Step(dbg.liveBankInfo)
		if err != nil {
			return err
		}
		dbg.counter.Step(1, dbg.liveBankInfo)

		if ended {
			if !dbg.vcs.CPU.LastResult.Final {
				// if we're in the rewinding state then a new rewind event has
				// started and we must return immediately so that it can continue...
				if dbg.State() == govern.Rewinding {
					return nil
				}

				// ...otherwise catchup has ended but we've not reached a CPU
				// instruction boundary then continue with nonInstructionQuantum loop.
				// for QuantumCycle we need to pay attention to the isCycle flag
				switch dbg.Quantum() {
				case govern.QuantumInstruction:
					return dbg.inputLoop(inputter, true)
				case govern.QuantumCycle:
					if isCycle {
						return dbg.inputLoop(inputter, true)
					}
				case govern.QuantumClock:
					return dbg.inputLoop(inputter, true)
				}
			}
		} else if dbg.catchupContinue != nil && !dbg.catchupContinue() {
			ended = true
			dbg.catchupEnd()

			// small optimisation if the catchup context is rewind to frame. we
			// never want to call the input loop in this case
			if dbg.catchupContext == catchupRewindToFrame {
				return nil
			}

			// if quantum if QuantumInstruction and we're not at the end of the
			// instruction then we never want to call the input loop
			if dbg.Quantum() == govern.QuantumInstruction && !dbg.vcs.CPU.LastResult.Final {
				return nil
			}

			return dbg.inputLoop(inputter, true)
		}

		return nil
	}

	// loop until the ended flag is false.
	//
	// there are a couple of additional conditions we need to be careful of in
	// this loop. first is what happens when a new cartridge is inserted or the
	// machine is otherwise reset. in those situations the CPU may be in an
	// illegal state so we need to (a) check for the cpu.ResetMidInstruction
	// sentinal error; and (b) whether the CPU has the HasReset() flag raised.
	// in both situations the loop is ended early
	for !ended {
		dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

		// coords of CPU instruction before calling vcs.Step()
		if dbg.vcs.CPU.RdyFlg {
			dbg.cpuBoundaryLastInstruction = dbg.vcs.TV.GetCoords()
		}

		err := dbg.vcs.Step(callback)
		if err != nil {
			if errors.Is(err, cpu.ResetMidInstruction) {
				return nil
			}
			return err
		}

		if dbg.vcs.CPU.HasReset() {
			return nil
		}

		// update disassembly after every CPU instruction. even during a catch
		// up we need to do this.
		dbg.liveDisasmEntry = dbg.Disasm.ExecutedEntry(dbg.liveBankInfo, dbg.vcs.CPU.LastResult, true, dbg.vcs.CPU.PC.Value())

		// make sure reflection has been updated at the end of the instruction
		if err = dbg.ref.Step(dbg.liveBankInfo); err != nil {
			return err
		}
		dbg.counter.Step(1, dbg.liveBankInfo)

		if dbg.unwindLoopRestart != nil {
			return nil
		}
	}

	return nil
}

// inputLoop has two modes, defined by the nonInstructionQuantum argument.
func (dbg *Debugger) inputLoop(inputter terminal.Input, nonInstructionQuantum bool) error {
	var err error

	for dbg.running {
		if dbg.Mode() != govern.ModeDebugger {
			return nil
		}

		if dbg.catchupContinue != nil {
			if nonInstructionQuantum {
				panic("refusing to run catchup loop inside a nonInstructionQuantum step")
			}

			err = dbg.catchupLoop(inputter)
			if err != nil {
				return err
			}

			if dbg.unwindLoopRestart != nil {
				return nil
			}
		}

		// checkTerm is used to decide whether to perform a full call to TermRead() and to potentially
		// halt the inputLoop - which we don't want to do unless there is something to process
		//
		// it will be false unless TermReadCheck() returns true
		var checkTerm bool

		// the select will take the eventCheckPulse channel if there is a tick
		// waiting only when we reach this point in the loop. it will not delay
		// the loop if the tick has not happened yet
		select {
		case <-dbg.readEventsPulse.C:
			err = dbg.readEventsHandler()
			if err != nil {
				if errors.Is(err, terminal.UserInterrupt) {
					dbg.handleInterrupt(inputter)
				} else if errors.Is(err, terminal.UserSignal) {
					return err
				} else {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// emulation has been put into a different mode. exit loop immediately
			if dbg.Mode() != govern.ModeDebugger {
				return nil
			}

			// if debugger is no longer running after checking interrupts and
			// events then break for loop
			if !dbg.running {
				break // dbg.running loop
			}

			// unwindLoopRestart or catchupContinue may have been set as a result
			// of readEventsHandler()

			if dbg.unwindLoopRestart != nil {
				return nil
			}

			if dbg.catchupContinue != nil {
				continue // dbg.running loop
			}

			checkTerm = inputter.TermReadCheck()
		default:
		}

		// return immediately if this inputLoop() is a nonInstructionQuantum
		// AND the prevailing quantum mode has been changed to instruction AND
		// the emulation has been asked to continue (eg. with STEP)
		//
		// this is important in a very specific situation:
		//
		// a) the emulation has been in nonInstrucution quantum mode (eg. CLOCK)
		// b) it is mid-way through a single CPU instruction
		// c) the debugger has been changed to INSTRUCTION quantum mode
		//
		// if we don't do this then debugging output will be wrong and confusing.
		if nonInstructionQuantum && dbg.continueEmulation && dbg.Quantum() == govern.QuantumInstruction {
			return nil
		}

		// check trace and output in context of last CPU result
		//
		// unlike halt conditions, I don't believe there is any need to do
		// check every color clock
		trace := dbg.traces.check()
		if trace != "" {
			if dbg.commandOnTrace != nil {
				err := dbg.processTokensList(dbg.commandOnTrace)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
			dbg.printLine(terminal.StyleFeedback, trace)
		}

		// bring all the halt conditions together
		halt := dbg.halting.halt || !dbg.runUntilHalt || dbg.haltImmediately || dbg.lastStepError

		// reset last step error
		dbg.lastStepError = false

		if halt {
			// check that dbg.running hasn't been unset while we've been
			// waiting for the halt condition.
			//
			// this can sometimes happen if the QUIT event is sent whilst the
			// emulation is halted mid CPU instruction
			if !dbg.running {
				break // dbg.running loop
			}

			// if this is a nonInstructionQuantum step and we've reach this stage then we need to
			// update the disassembly. we do not update the nextAddr however
			if nonInstructionQuantum {
				dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())
				dbg.liveDisasmEntry = dbg.Disasm.ExecutedEntry(dbg.liveBankInfo, dbg.vcs.CPU.LastResult, false, 0)
			}

			// always clear volatile breakpoints/traps. if the emulation has halted for any
			// reason then any existing step trap is stale.
			dbg.halting.volatileBreakpoints.clear()
			dbg.halting.volatileTraps.clear()

			// input has halted. print on halt command if it is defined
			if dbg.commandOnHalt != nil {
				err := dbg.processTokensList(dbg.commandOnHalt)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// set pause emulation state
			dbg.setState(govern.Paused, govern.Normal)

			// take note of current machine state if the emulation was in a running
			// state and is halting just now
			if dbg.continueEmulation && inputter.IsInteractive() {
				dbg.Rewind.RecordExecutionCoords()
			}

			// reset halting flag before we resume execution.
			dbg.halting.reset()

			// reset run until halt flag - it will be set again if the parsed
			// command requires it (eg. the RUN command)
			dbg.runUntilHalt = false

			// reset continueEmulation flag - it will set again by any command
			// that requires it
			dbg.continueEmulation = false

			// reset haltImmediately flag - it will be set again with the next
			// HALT command
			dbg.haltImmediately = false

			// we've been instructed to abandon this inputLoop().
			if dbg.stepOutOfVideoStepInputLoop {
				dbg.stepOutOfVideoStepInputLoop = false
				// check that we really are in a nonInstructionQuantum
				if nonInstructionQuantum {
					return nil
				} else {
					logger.Log(logger.Allow, "debugger", "asked to 'step out of nonInstructionQuantum step input loop' inappropriately")
				}
			}

			// read input from terminal inputter and parse/run commands
			err = dbg.termRead(inputter)
			if err != nil {
				return err
			}

			// emulation has been put into a different mode. exit loop immediately
			if dbg.Mode() != govern.ModeDebugger {
				return nil
			}

			if dbg.unwindLoopRestart != nil {
				return nil
			}

			if dbg.catchupContinue != nil {
				continue // dbg.running loop
			}

			// unpause emulation if we're continuing emulation
			if dbg.runUntilHalt {
				// runUntilHalt is set to true when stepping by more than a
				// clock (ie. by scanline of frame) but in those cases we want
				// to set gui state to StateStepping and not StateRunning.
				//
				// Setting to StateRunning may have different graphical
				// side-effects which would look ugly when we're only in fact
				// stepping.
				if dbg.halting.volatileTraps.isEmpty() {
					if inputter.IsInteractive() {
						dbg.setState(govern.Running, govern.Normal)
					}
				} else {
					dbg.setState(govern.Stepping, govern.Normal)
				}

				// update comparison point before execution continues
				if !nonInstructionQuantum {
					dbg.Rewind.UpdateComparison()
				}
			} else if inputter.IsInteractive() {
				dbg.setState(govern.Stepping, govern.Normal)
			}
		}

		if checkTerm {
			err := dbg.termRead(inputter)
			if err != nil {
				return err
			}

			if dbg.unwindLoopRestart != nil {
				return nil
			}

			if dbg.catchupContinue != nil {
				continue // dbg.running loop
			}
		}

		if dbg.continueEmulation {
			// make sure we still want to continue after the call to resumAfterHalt()
			if dbg.continueEmulation {
				// input loops with the isVideoStep flag must never execute another
				// call to vcs.Step() under any circumstances
				//
				// we also don't allow this call to inputLoop() to loop. if there
				// is any more nonInstructionQuantum steps to handle, the function will be called
				// again
				if nonInstructionQuantum {
					return nil
				}

				err = dbg.step(inputter, false)
				if err != nil {
					return err
				}

				// skip over WSYNC (CPU RDY flag is false) only if we're in instruction quantum
				if dbg.Quantum() == govern.QuantumInstruction {
					for !dbg.vcs.CPU.RdyFlg {
						err = dbg.step(inputter, false)
						if err != nil {
							return err
						}
					}
				}

				// check for unwind loop
				if dbg.unwindLoopRestart != nil {
					return nil
				}
			}
		}
	}

	return nil
}

func (dbg *Debugger) step(inputter terminal.Input, catchup bool) error {
	callback := func(isCycle bool) error {
		var err error

		// check for unwind loop
		if dbg.unwindLoopRestart != nil {
			return nil
		}

		// we do need to update the reflection however
		err = dbg.ref.Step(dbg.liveBankInfo)
		if err != nil {
			return err
		}
		dbg.counter.Step(1, dbg.liveBankInfo)

		q := dbg.Quantum()

		// process commandOnStep for non-instruction quantums (equivalent for
		// instruction quantum is the main body of Debugger.step() below)
		if dbg.commandOnStep != nil {
			// we don't do this if we're in catchup mode or the "final" result from the CPU
			if !catchup && !dbg.vcs.CPU.LastResult.Final {
				if q == govern.QuantumClock || (q == govern.QuantumCycle && isCycle) {
					err := dbg.processTokensList(dbg.commandOnStep)
					if err != nil {
						dbg.printLine(terminal.StyleError, "%s", err)
					}
				}
			}
		}

		// check halt condition. a second check is made after vcs.Step()
		// returns below
		dbg.continueEmulation = dbg.halting.check()

		if q == govern.QuantumClock || (q == govern.QuantumCycle && isCycle) || !dbg.continueEmulation {
			// start another inputLoop() with the clockCycle boolean set to true
			return dbg.inputLoop(inputter, true)
		}

		return nil
	}

	// get bank information before we execute the next instruction. we
	// use this when formatting the last result from the CPU. this has
	// to happen before we call the VCS.Step() function
	dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

	// coords of CPU instruction before calling vcs.Step()
	if dbg.vcs.CPU.RdyFlg {
		dbg.cpuBoundaryLastInstruction = dbg.vcs.TV.GetCoords()
	}

	// not using the err variable because we'll clobber it before we
	// get to check the result of VCS.Step()
	stepErr := dbg.vcs.Step(callback)

	// check halt condition again now that the instruction has finished (the
	// Final flag is true). this does mean that some breakpoints/traps are
	// matched twice but that's not currently a problem
	dbg.halting.check()
	dbg.continueEmulation = !dbg.halting.halt

	// update disassembly after every CPU instruction. no exceptions.
	dbg.liveDisasmEntry = dbg.Disasm.ExecutedEntry(dbg.liveBankInfo, dbg.vcs.CPU.LastResult, true, dbg.vcs.CPU.PC.Value())

	// make sure reflection has been updated at the end of the instruction
	if err := dbg.ref.Step(dbg.liveBankInfo); err != nil {
		return err
	}
	dbg.counter.Step(1, dbg.liveBankInfo)

	if dbg.unwindLoopRestart != nil {
		return nil
	}

	if stepErr != nil {
		// ...set lastStepError instead and allow emulation to halt
		dbg.lastStepError = true
		dbg.printLine(terminal.StyleError, "%s", stepErr)

		// error has occurred before CPU has completed its instruction
		if !dbg.vcs.CPU.LastResult.Final {
			dbg.printLine(terminal.StyleError, "CPU halted mid-instruction. next step may be inaccurate.")
		}
	} else {
		// update rewind state if the last CPU instruction took place during a new
		// frame event. but not if we're in catchup mode
		if !catchup {
			dbg.Rewind.RecordState()
		}

		// process commandOnStep for instruction quantum (equivalent for clock
		// quantum is the vcs.Step() callback above)
		if dbg.vcs.CPU.RdyFlg {
			q := dbg.Quantum()
			if q == govern.QuantumInstruction || (q != govern.QuantumInstruction && dbg.vcs.CPU.LastResult.Final) {
				if dbg.commandOnStep != nil {
					err := dbg.processTokensList(dbg.commandOnStep)
					if err != nil {
						dbg.printLine(terminal.StyleError, "%s", err)
					}
				}
			}
		}
	}

	return nil
}

// termRead uses the TermRead() function of the inputter and process the output
// as required by the debugger.
func (dbg *Debugger) termRead(inputter terminal.Input) error {
	var input string
	var err error

	// process remainingInput before get user input from terminal.Input implementatio
	if s, ok := dbg.scriptHandler.Next(); ok {
		input = s
	} else {
		input, err = inputter.TermRead(dbg.buildPrompt(), dbg.events)
	}

	if dbg.unwindLoopRestart != nil {
		return nil
	}

	// if there was no error from TermRead parse input (leading to execution) of the command
	if err == nil {
		if len(input) > 0 {
			s, err := dbg.parseInput(input, inputter.IsInteractive())
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			}
			dbg.scriptHandler.Push(s)
		}
	} else {
		if errors.Is(err, terminal.UserInterrupt) {
			// user interrupts are used to quit or halt an operation
			dbg.handleInterrupt(inputter)

		} else if errors.Is(err, io.EOF) {
			// an EOF error causes the emulation to exit immediately
			dbg.running = false
			dbg.continueEmulation = false
			return nil

		} else {
			// all other errors are passed upwards to the calling function
			return err
		}
	}

	// recurse into termRead() function if necessary
	if !dbg.continueEmulation && dbg.scriptHandler.More() {
		return dbg.termRead(inputter)
	}

	return nil
}

// interrupt signals need some special care depending on the current state and
// what sort of terminal is being used.
func (dbg *Debugger) handleInterrupt(inputter terminal.Input) {
	// end script scribe (if one is running)
	err := dbg.scriptWrite.EndSession()
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}

	// exit immediately if inputter is not a real terminal
	if !inputter.IsRealTerminal() {
		dbg.running = false
		dbg.continueEmulation = false
		return
	}

	// if the emulation is currently running then stop emulation
	if dbg.runUntilHalt {
		dbg.runUntilHalt = false
		dbg.continueEmulation = false
		return
	}

	// terminal is not interactive so we set running to false which will
	// quit the debugger as soon as possible
	if !inputter.IsInteractive() {
		dbg.running = false
		dbg.continueEmulation = false
		return
	}

	// terminal is interactive so we ask for quit confirmation
	s, err := inputter.TermRead(terminal.Prompt{
		Content: "really quit (y/n) ",
		Type:    terminal.PromptTypeConfirm,
	}, dbg.events)

	if err != nil {
		// another UserInterrupt has occurred. we treat a second UserInterrupt
		// as thought 'y' was pressed
		if errors.Is(err, terminal.UserInterrupt) {
			s = "y"
		} else {
			dbg.printLine(terminal.StyleError, err.Error())
		}
	}

	// check if confirmation has been confirmed
	if s == "y" || s == "Y" {
		dbg.running = false
	}
}
