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
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
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
// the caller of the function to set emulation.State appropriately.
func (dbg *Debugger) unwindLoop(onRestart func() error) {
	dbg.unwindLoopRestart = onRestart
}

// CatchUpLoop implements the rewind.Runner interface.
//
// It is called from the rewind package and sets the functions that are
// required for catchupLoop().
func (dbg *Debugger) CatchUpLoop(frame int, scanline int, clock int) error {
	// turn off TV's fps frame limiter
	fpsCap := dbg.vcs.TV.SetFPSCap(false)

	// we've already set emulation state to emulation.Rewinding

	dbg.catchupContinue = func() bool {
		nf := dbg.vcs.TV.GetState(signal.ReqFramenum)
		ns := dbg.vcs.TV.GetState(signal.ReqScanline)
		nc := dbg.vcs.TV.GetState(signal.ReqClock)

		// returns true if we're to continue
		return !(nf > frame || (nf == frame && ns > scanline) || (nf == frame && ns == scanline && nc >= clock))
	}

	dbg.catchupEnd = func() {
		dbg.vcs.TV.SetFPSCap(fpsCap)
		dbg.catchupContinue = nil
		dbg.catchupEnd = nil
		dbg.setState(emulation.Paused)
		dbg.continueEmulation = false
		dbg.runUntilHalt = false
	}

	dbg.PushRawEventReturn(func() {})

	return nil
}

// catchupLoop is a special purpose loop designed to run inside of the inputLoop. it is called only
// when catchupContinue has been set in CatchUpLoop(), which is called as a consequence of a rewind event.
func (dbg *Debugger) catchupLoop(inputter terminal.Input) error {
	callbackInstruction := func() error {
		return dbg.ref.OnVideoCycle(dbg.lastBank)
	}

	// whether the catch up conditions have been met inside a video step (ie.
	// between CPU instruction boundaries).
	var endedInVideoStep bool

	callbackVideo := func() error {
		var err error

		// update debugger the same way for video quantum as for cpu quantum
		err = callbackInstruction()
		if err != nil {
			return err
		}

		if !endedInVideoStep && dbg.catchupContinue != nil && !dbg.catchupContinue() {
			dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
			if err != nil {
				return err
			}
			endedInVideoStep = true
			dbg.catchupEnd()
			return dbg.inputLoop(inputter, true)
		}

		return nil
	}

	for dbg.catchupContinue() {
		dbg.lastBank = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

		switch dbg.quantum {
		case QuantumInstruction:
			err := dbg.vcs.Step(callbackInstruction)
			if err != nil {
				return err
			}
		case QuantumVideo:
			err := dbg.vcs.Step(callbackVideo)
			if err != nil {
				return err
			}
		default:
			err := fmt.Errorf("unknown quantum mode")
			if err != nil {
				return err
			}
		}

		// make sure reflection has been updated at the end of the instruction
		if err := dbg.ref.OnInstructionEnd(dbg.lastBank); err != nil {
			return err
		}

		// catchup conditions have been met so return immediatly
		if endedInVideoStep {
			return nil
		}
	}

	dbg.catchupEnd()

	var err error

	dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)

	return err
}

// inputLoop has two modes, defined by the clockCycle argument. when clockCycle
// is true then user will be prompted every video cycle; when false the user
// is prompted every cpu instruction.
func (dbg *Debugger) inputLoop(inputter terminal.Input, clockCycle bool) error {
	var err error

	// unwindLoopRestart is checked frequently and will cause the inputLoop()
	// to return early. the function also returns early
	for dbg.running {
		if dbg.unwindLoopRestart != nil {
			return nil
		}

		// how we enter and leave the catchup loop is very important.
		if dbg.catchupContinue != nil {
			if clockCycle {
				panic("refusing to run catchup loop inside a color clock cycle")
			}

			err = dbg.catchupLoop(inputter)
			if err != nil {
				return err
			}

			if dbg.unwindLoopRestart != nil {
				return nil
			}
		}

		// raise hasChanged flag every iteration
		dbg.hasChanged = true

		// checkTerm is used to decide whether to perform a full call to TermRead() and to potentially
		// halt the inputLoop - which we don't want to do unless there is something to process
		//
		// it will be false unless TermReadCheck() returns true
		var checkTerm bool

		// the select will take the eventCheckPulse channel if there is a tick
		// waiting only when we reach this point in the loop. it will not delay
		// the loop if the tick has not happened yet
		select {
		case <-dbg.eventCheckPulse.C:
			err = dbg.readEventsHandler()
			if err != nil {
				dbg.running = false
				dbg.printLine(terminal.StyleError, "%s", err)
			}

			// if debugger is no longer running after checking interrupts and
			// events then break for loop
			if !dbg.running {
				break // dbg.running loop
			}

			// unwindLoopRestart or catchupContinue may have been set as a result
			// of checkEvents()
			if dbg.unwindLoopRestart != nil {
				return nil
			}
			if dbg.catchupContinue != nil {
				continue // dbg.running loop
			}

			checkTerm = inputter.TermReadCheck()
		default:
		}

		// return immediately if this inputLoop() is a clockCycle AND the
		// current quantum mode has been changed to instruction AND the
		// emulation has been asked to continue (eg. with STEP)
		//
		// this is important in a very specific situation:
		//
		// a) the emulation has been in CLOCK quantum mode
		// b) it is mid-way through a single CPU instruction
		// c) the debugger has been changed to INSTRUCTION quantum mode
		//
		// if we don't do this then debugging output will be wrong and confusing.
		if clockCycle && dbg.continueEmulation && dbg.quantum == QuantumInstruction {
			return nil
		}

		// check trace and output in context of last CPU result
		trace := dbg.traces.check()
		if trace != "" {
			if dbg.commandOnTrace != nil {
				err := dbg.processTokensList(dbg.commandOnTrace)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf(" <trace> %s", trace))
		}

		// check for halt conditions and process halt state
		//
		// note that at the moment if we've stopped after a catchupLoop then we
		// may see break messages and commandOnHalt messages. this doesn't
		// affect anything but it's arguably not the correct behaviour. in
		// other words, rewinding should output any of these messages.
		//
		// TODO: halt messages should not be displayed after a rewind event

		var haltEmulation bool
		var stepTrapMessage string
		var breakMessage string
		var trapMessage string
		var watchMessage string

		// check for breakpoints and traps. for video cycle input loops we only
		// do this if the instruction has affected flow.
		if !clockCycle || (dbg.vcs.CPU.LastResult.Defn != nil &&
			(dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Flow ||
				dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Subroutine ||
				dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Interrupt)) {
			breakMessage = dbg.breakpoints.check(breakMessage)
			trapMessage = dbg.traps.check(trapMessage)
			watchMessage = dbg.watches.check(watchMessage)
			stepTrapMessage = dbg.stepTraps.check("")
		}

		// check for halt conditions
		haltEmulation = stepTrapMessage != "" || breakMessage != "" ||
			trapMessage != "" || watchMessage != "" ||
			dbg.lastStepError || dbg.haltImmediately

		// expand halt to include step-once/many flag
		haltEmulation = haltEmulation || !dbg.runUntilHalt

		// reset last step error
		dbg.lastStepError = false

		// if emulation is to be halted or if we need to check the terminal
		if haltEmulation {
			// always clear steptraps. if the emulation has halted for any
			// reason then any existing step trap is stale.
			dbg.stepTraps.clear()

			// print and reset accumulated break/trap/watch messages
			dbg.printLine(terminal.StyleFeedback, breakMessage)
			dbg.printLine(terminal.StyleFeedback, trapMessage)
			dbg.printLine(terminal.StyleFeedback, watchMessage)
			breakMessage = ""
			trapMessage = ""
			watchMessage = ""

			// input has halted. print on halt command if it is defined
			if dbg.commandOnHalt != nil {
				err := dbg.processTokensList(dbg.commandOnHalt)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// set pause emulation state
			dbg.setState(emulation.Paused)

			// take note of current machine state if the emulation was in a running
			// state and is halting just now
			if dbg.continueEmulation && inputter.IsInteractive() && !clockCycle {
				dbg.Rewind.RecordExecutionState()
			}

			// reset run until halt flag - it will be set again if the parsed
			// command requires it (eg. the RUN command)
			dbg.runUntilHalt = false

			// reset haltImmediately flag - it will be set again with the next
			// HALT command
			dbg.haltImmediately = false

			// reset continueEmulation flag - it will set again by any command
			// that requires it
			dbg.continueEmulation = false

			// read input from terminal inputter and parse/run commands
			err = dbg.termRead(inputter)
			if err != nil {
				if curated.Is(err, script.ScriptEnd) {
					dbg.printLine(terminal.StyleFeedback, err.Error())
					return nil
				}
				return err
			}

			// hasChanged flag may have been false for a long time after the
			// termRead() pause. set to true immediately.
			dbg.hasChanged = true

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
				if dbg.stepTraps.isEmpty() {
					if inputter.IsInteractive() {
						dbg.setState(emulation.Running)
					}
				} else {
					dbg.setState(emulation.Stepping)
				}

				// update comparison point before execution continues
				if !clockCycle {
					dbg.Rewind.SetComparison()
				}
			} else if inputter.IsInteractive() {
				dbg.setState(emulation.Stepping)
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
			// input loops with the clockCycle flag must not execute another call to vcs.Step()
			//
			// when this happens the previous calls to inputLoop() unwind partially and continue
			// from where the function was originally called.
			//
			// this will be in the callbackVideo callback function in either of:
			//
			// a) step()
			// b) catchupLoop()
			//
			// in both cases the emulation will continue inside the vcs.Step() function via the
			// step() function
			if clockCycle {
				return nil
			}

			err = dbg.step(inputter, false)
			if err != nil {
				return err
			}

			// check exit video loop
			if dbg.unwindLoopRestart != nil {
				return nil
			}

			// commandOnStep is processed every time emulation has step
			if dbg.commandOnStep != nil {
				err := dbg.processTokensList(dbg.commandOnStep)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
		}
	}

	return nil
}

func (dbg *Debugger) step(inputter terminal.Input, catchup bool) error {
	callbackInstruction := func() error {
		return dbg.ref.OnVideoCycle(dbg.lastBank)
	}

	callbackVideo := func() error {
		var err error

		if dbg.unwindLoopRestart != nil {
			return nil
		}

		// format last CPU execution result for vcs step. this is in addition
		// to the FormatResult() call in the main dbg.running loop below.
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}

		// update debugger the same way for video quantum as for cpu quantum
		err = callbackInstruction()
		if err != nil {
			return err
		}

		// for video quantums we need to run any OnStep commands before
		// starting a new inputLoop
		if dbg.commandOnStep != nil {
			// we don't do this if we're in catchup mode
			if !catchup {
				err := dbg.processTokensList(dbg.commandOnStep)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
		}

		// start another inputLoop() with the clockCycle boolean set to true
		return dbg.inputLoop(inputter, true)
	}

	// get bank information before we execute the next instruction. we
	// use this when formatting the last result from the CPU. this has
	// to happen before we call the VCS.Step() function
	dbg.lastBank = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())

	// not using the err variable because we'll clobber it before we
	// get to check the result of VCS.Step()
	var stepErr error

	switch dbg.quantum {
	case QuantumInstruction:
		stepErr = dbg.vcs.Step(callbackInstruction)
	case QuantumVideo:
		stepErr = dbg.vcs.Step(callbackVideo)
	default:
		stepErr = fmt.Errorf("unknown quantum mode")
	}

	// make sure reflection has been updated at the end of the instruction
	if err := dbg.ref.OnInstructionEnd(dbg.lastBank); err != nil {
		return err
	}

	if dbg.unwindLoopRestart != nil {
		return nil
	}

	// update rewind state if the last CPU instruction took place during a new
	// frame event. but not if we're in catchup mode
	if !catchup {
		dbg.Rewind.RecordFrameState()
	}

	// check step error. note that we format and store last CPU
	// execution result whether there was an error or not. in the case
	// of an error the resul a fresh formatting. if there was no error
	// the formatted result is returned by the ExecutedEntry() function.

	if stepErr != nil {
		var err error

		// format last execution result even on error
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.vcs.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}

		// exit input loop if error is a plain error
		if !curated.IsAny(stepErr) {
			return stepErr
		}

		// ...set lastStepError instead and allow emulation to halt
		dbg.lastStepError = true
		dbg.printLine(terminal.StyleError, "%s", stepErr)

		// error has occurred before CPU has completed its instruction
		if !dbg.lastResult.Result.Final {
			dbg.printLine(terminal.StyleError, "CPU halted mid-instruction. next step may be inaccurate.")
			dbg.vcs.CPU.Interrupted = true
		}
	} else if dbg.vcs.CPU.LastResult.Final {
		var err error

		// update entry and store result as last result
		dbg.lastResult, err = dbg.Disasm.ExecutedEntry(dbg.lastBank, dbg.vcs.CPU.LastResult, dbg.vcs.CPU.PC.Value())
		if err != nil {
			return err
		}

		// check validity of instruction result
		err = dbg.vcs.CPU.LastResult.IsValid()
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", dbg.vcs.CPU.LastResult.Defn)
			dbg.printLine(terminal.StyleError, "%s", dbg.vcs.CPU.LastResult)
			return err
		}
	}

	return nil
}

// termRead uses the TermRead() function of the inputter and process the output
// as required by the debugger.
func (dbg *Debugger) termRead(inputter terminal.Input) error {
	// get user input from terminal.Input implementatio
	inputLen, err := inputter.TermRead(dbg.input, dbg.buildPrompt(), dbg.events)

	// check exit video loop and return immediately if required
	if dbg.unwindLoopRestart != nil {
		return nil
	}

	// errors returned by UserRead() functions are very rich. the
	// following block interprets the error carefully and proceeds
	// appropriately
	if err != nil {
		if !curated.IsAny(err) {
			// if the error originated from outside of gopher2600 then
			// it is probably serious or unexpected
			switch err {
			case io.EOF:
				// treat EOF errors the same as terminal.UserAbort
				err = curated.Errorf(terminal.UserAbort)
			default:
				// the error is probably serious. exit input loop with err
				return err
			}
		}

		// we now know the we have an project specific error
		if curated.Is(err, terminal.UserInterrupt) {
			// user interrupts are triggered by the user (in a terminal
			// environment, usually by pressing ctrl-c)
			dbg.handleInterrupt(inputter)
		} else if curated.Is(err, terminal.UserAbort) {
			// like UserInterrupt but with no confirmation stage
			dbg.running = false
			return nil
		} else {
			// all other errors are passed upwards to the calling function
			return err
		}
	}

	// sometimes UserRead can return zero bytes read, we need to filter
	// this out before we try any parsing
	//
	// parsing may cause a number of inputLoop flags to changes. for example:
	// continueEmulation or runUntilHalt.
	if inputLen > 0 {
		err = dbg.parseInput(string(dbg.input[:inputLen-1]), inputter.IsInteractive(), false)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
		}
	}

	return nil
}

// interrupt errors that are sent back to the debugger need some special care
// depending on the current state:
//
// * if script recording is active then recording is ended
// * for non-interactive input set running flag to false immediately
// * otherwise, prompt use for confirmation that the debugger should quit.
func (dbg *Debugger) handleInterrupt(inputter terminal.Input) {
	if dbg.scriptScribe.IsActive() {
		// script recording is in progress so we insert SCRIPT END into the
		// input stream
		dbg.input = []byte("SCRIPT END")
		return
	}

	if !inputter.IsInteractive() {
		// terminal is not interactive so we set running to false which will
		// quit the debugger as soon as possible
		dbg.running = false
	}

	// terminal is interactive so we ask for quit confirmation

	confirm := make([]byte, 1)
	_, err := inputter.TermRead(confirm,
		terminal.Prompt{
			Content: "really quit (y/n) ",
			Type:    terminal.PromptTypeConfirm},
		dbg.events)

	if err != nil {
		// another UserInterrupt has occurred. we treat a second UserInterrupt
		// as thought 'y' was pressed
		if curated.Is(err, terminal.UserInterrupt) {
			confirm[0] = 'y'
		} else {
			dbg.printLine(terminal.StyleError, err.Error())
		}
	}

	// check if confirmation has been confirmed
	if confirm[0] == 'y' || confirm[0] == 'Y' {
		dbg.running = false
	}
}
