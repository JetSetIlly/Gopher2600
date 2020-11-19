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
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
)

// inputLoop has two modes, defined by the videoCycle argument. when videoCycle
// is true then user will be prompted every video cycle; when false the user
// is prompted every cpu cycle.
func (dbg *Debugger) inputLoop(inputter terminal.Input, videoCycle bool) error {
	dbg.isVideoCycleInputLoop = videoCycle

	// to speed things a bit we only check for input events every
	// "inputCtDelay" iterations.
	const inputCtDelay = 50
	inputCt := 0

	for dbg.running {
		var err error
		var checkTerm bool

		// check for events every inputCtDelay iteratins
		inputCt++
		if inputCt%inputCtDelay == 0 {
			inputCt = 0

			checkTerm = inputter.TermReadCheck()

			err = dbg.checkEvents()
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			}

			// check exit video loop
			if dbg.inputLoopRestart {
				return nil
			}
		}

		// if debugger is no longer running after checking interrupts and
		// events then break for loop
		if !dbg.running {
			break // for loop
		}

		// return immediately if this inputLoop() is a videoCycle, the current
		// quantum mode has been changed to quantumCPU and the emulation has
		// been asked to continue with (eg. with STEP)
		//
		// this is important in a very specific situation:
		// a) the emulation has been in video quantum mode
		// b) it is mid-way between CPU quantums
		// c) the debugger has been changed to cpu quantum mode
		//
		// if we don't do this then debugging output will be wrong and confusing.
		if videoCycle && dbg.continueEmulation && dbg.quantum == QuantumCPU {
			return nil
		}

		// check trace and output in context of last CPU result
		trace := dbg.traces.check()
		if trace != "" {
			if dbg.commandOnTrace != nil {
				err := dbg.processTokenGroup(dbg.commandOnTrace)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf(" <trace> %s", trace))
		}

		var stepTrapMessage string
		var breakMessage string
		var trapMessage string
		var watchMessage string

		// check for breakpoints and traps. for video cycle input loops we only
		// do this if the instruction has affected flow.
		if !videoCycle || (dbg.VCS.CPU.LastResult.Defn != nil &&
			(dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Flow ||
				dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Subroutine ||
				dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Interrupt)) {
			breakMessage = dbg.breakpoints.check(breakMessage)
			trapMessage = dbg.traps.check(trapMessage)
			watchMessage = dbg.watches.check(watchMessage)
			stepTrapMessage = dbg.stepTraps.check("")
		}

		// check for halt conditions
		haltEmulation := stepTrapMessage != "" || breakMessage != "" ||
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
				err := dbg.processTokenGroup(dbg.commandOnHalt)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// pause TV/GUI when emulation has halted
			err = dbg.tv.Pause(true)
			if err != nil {
				return err
			}
			err = dbg.scr.SetFeature(gui.ReqState, gui.StatePaused)
			if err != nil {
				return err
			}

			// take note of current machine state if the emulation was in a running
			// state and is halting just now
			if dbg.continueEmulation && inputter.IsInteractive() && !videoCycle {
				dbg.Rewind.ExecutionState()
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
			err = dbg.readTerminal(inputter)
			if err != nil {
				if curated.Is(err, script.ScriptEnd) {
					dbg.printLine(terminal.StyleFeedback, err.Error())
					return nil
				}
				return err
			}

			// check exit video loop
			if dbg.inputLoopRestart {
				return nil
			}

			// unpause emulation if we're continuing emulation
			if dbg.runUntilHalt {
				// unpause TV/GUI if there are no step traps. unpausing a TV/GUI when
				// stepping by scanline, for example, looks ugly
				if dbg.stepTraps.isEmpty() {
					err = dbg.tv.Pause(false)
					if err != nil {
						return err
					}
					err = dbg.scr.SetFeature(gui.ReqState, gui.StateRunning)
					if err != nil {
						return err
					}
				}

				// update comparison point before execution continues
				if !videoCycle {
					dbg.Rewind.SetComparison()
				}
			}
		}

		if checkTerm {
			err := dbg.readTerminal(inputter)
			if err != nil {
				return err
			}

			// check exit video loop
			if dbg.inputLoopRestart {
				return nil
			}
		}

		if dbg.continueEmulation {
			// input loops with the videoCycle flag must not execute another
			// call to vcs.Step()
			if videoCycle {
				return nil
			}

			err = dbg.contEmulation(inputter)
			if err != nil {
				return err
			}

			// make sure videocyle info is current
			dbg.isVideoCycleInputLoop = videoCycle

			// check exit video loop
			if dbg.inputLoopRestart {
				return nil
			}

			// command on step is process every time emulation has continued one step
			if dbg.commandOnStep != nil {
				err := dbg.processTokenGroup(dbg.commandOnStep)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}
		}
	}

	return nil
}

func (dbg *Debugger) readTerminal(inputter terminal.Input) error {
	// get user input
	inputLen, err := inputter.TermRead(dbg.input, dbg.buildPrompt(), dbg.events)

	// check exit video loop
	if dbg.inputLoopRestart {
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
	// this out before we try any
	if inputLen > 0 {
		// parse user input, taking note of whether the emulation should
		// continue
		err = dbg.parseInput(string(dbg.input[:inputLen-1]), inputter.IsInteractive(), false)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
		}
	}

	return nil
}

func (dbg *Debugger) contEmulation(inputter terminal.Input) error {
	quantumCPU := func() error {
		if dbg.reflect == nil {
			return nil
		}
		return dbg.reflect.Check(dbg.lastBank)
	}

	quantumVideo := func() error {
		var err error

		if dbg.inputLoopRestart {
			return nil
		}

		// format last CPU execution result for vcs step. this is in addition
		// to the FormatResult() call in the main dbg.running loop below.
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.VCS.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}

		// update debugger the same way for video quantum as for cpu quantum
		err = quantumCPU()
		if err != nil {
			return err
		}

		// for video quantums we need to run any OnStep commands before
		// starting a new inputLoop
		if dbg.commandOnStep != nil {
			err := dbg.processTokenGroup(dbg.commandOnStep)
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			}
		}

		// start another inputLoop() with the videoCycle boolean set to true
		return dbg.inputLoop(inputter, true)
	}

	// get bank information before we execute the next instruction. we
	// use this when formatting the last result from the CPU. this has
	// to happen before we call the VCS.Step() function
	dbg.lastBank = dbg.VCS.Mem.Cart.GetBank(dbg.VCS.CPU.PC.Address())

	// not using the err variable because we'll clobber it before we
	// get to check the result of VCS.Step()
	var stepErr error

	switch dbg.quantum {
	case QuantumCPU:
		stepErr = dbg.VCS.Step(quantumCPU)
	case QuantumVideo:
		stepErr = dbg.VCS.Step(quantumVideo)
	default:
		stepErr = fmt.Errorf("unknown quantum mode")
	}

	if dbg.inputLoopRestart {
		return nil
	}

	// update rewind state if the last CPU instruction took place during a new
	// frame event
	dbg.Rewind.Check()

	// check step error. note that we format and store last CPU
	// execution result whether there was an error or not. in the case
	// of an error the resul a fresh formatting. if there was no error
	// the formatted result is returned by the ExecutedEntry() function.

	if stepErr != nil {
		var err error

		// format last execution result even on error
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.VCS.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}

		// the supercharger ROM will eventually start execution from the PC
		// address given in the supercharger file. when "fast-loading"
		// supercharger bin files however, we need a way of doing this without
		// the ROM. the TapeLoaded error allows us to do this.
		if onTapeLoaded, ok := stepErr.(supercharger.FastLoaded); ok {
			// CPU execution has been interrupted. update state of CPU
			dbg.VCS.CPU.Interrupted = true

			// the interrupted CPU means it never got a chance to
			// finalise the result. we force that here by simply
			// setting the Final flag to true.
			dbg.VCS.CPU.LastResult.Final = true

			// we've already obtained the disassembled lastResult so we
			// need to change the final flag there too
			dbg.lastResult.Result.Final = true

			// call function to complete tape loading procedure
			err = onTapeLoaded(dbg.VCS.CPU, dbg.VCS.Mem.RAM, dbg.VCS.RIOT.Timer)
			if err != nil {
				return err
			}

			// (re)disassemble memory on TapeLoaded error signal
			err = dbg.Disasm.FromMemory(nil, nil)
			if err != nil {
				return err
			}
		} else {
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
				dbg.VCS.CPU.Interrupted = true
			}
		}
	} else if dbg.VCS.CPU.LastResult.Final {
		var err error

		// update entry and store result as last result
		dbg.lastResult, err = dbg.Disasm.ExecutedEntry(dbg.lastBank, dbg.VCS.CPU.LastResult, dbg.VCS.CPU.PC.Value())
		if err != nil {
			return err
		}

		// check validity of instruction result
		err = dbg.VCS.CPU.LastResult.IsValid()
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", dbg.VCS.CPU.LastResult.Defn)
			dbg.printLine(terminal.StyleError, "%s", dbg.VCS.CPU.LastResult)
			return err
		}
	}

	return nil
}

// interrupt errors that are sent back to the debugger need some special care
// depending on the current state.
//
// - if script recording is active then recording is ended
// - for non-interactive input set running flag to false immediately
// - otherwise, prompt use for confirmation that the debugger should quit.
func (dbg *Debugger) handleInterrupt(inputter terminal.Input) {
	// if script input is being capture by a scriptScribe then
	// we the user interrupt event as a SCRIPT END
	// command.
	if dbg.scriptScribe.IsActive() {
		dbg.input = []byte("SCRIPT END")
	} else if !inputter.IsInteractive() {
		dbg.running = false
	} else {
		// a scriptScribe is not active nor is this a script
		// input loop. ask the user if they really want to quit
		confirm := make([]byte, 1)
		_, err := inputter.TermRead(confirm,
			terminal.Prompt{
				Content: "really quit (y/n) ",
				Type:    terminal.PromptTypeConfirm},
			dbg.events)

		if err != nil {
			// another UserInterrupt has occurred. we treat
			// UserInterrupt as thought 'y' was pressed
			if curated.Is(err, terminal.UserInterrupt) {
				confirm[0] = 'y'
			} else {
				dbg.printLine(terminal.StyleError, err.Error())
			}
		}

		// check if confirmation has been confirmed and run
		// QUIT command
		if confirm[0] == 'y' || confirm[0] == 'Y' {
			dbg.running = false
		}
	}
}
