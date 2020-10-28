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
	// vcsStep is to be called every video cycle when the quantum mode
	// is set to CPU
	vcsStep := func() error {
		if dbg.reflect == nil {
			return nil
		}
		return dbg.reflect.Check(dbg.lastBank)
	}

	// vcsStepVideo is to be called every video cycle when the quantum mode
	// is set to Video
	vcsStepVideo := func() error {
		var err error

		// format last CPU execution result for vcs step. this is in addition
		// to the FormatResult() call in the main dbg.running loop below.
		dbg.lastResult, err = dbg.Disasm.FormatResult(dbg.lastBank, dbg.VCS.CPU.LastResult, disassembly.EntryLevelExecuted)
		if err != nil {
			return err
		}

		// update debugger the same way for video quantum as for cpu quantum
		err = vcsStep()
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

	// to speed things a bit we only check for input events every
	// "inputCtDelay" iterations.
	const inputCtDelay = 50
	inputCt := 0

	for dbg.running {
		var err error
		var checkTerm bool

		inputCt++
		if inputCt%inputCtDelay == 0 {
			inputCt = 0
			// check for events
			checkTerm, err = dbg.checkEvents(inputter)
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
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

		// check for breakpoints and traps
		if !videoCycle ||
			(dbg.VCS.CPU.LastResult.Final &&
				dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Flow ||
				dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Subroutine ||
				dbg.VCS.CPU.LastResult.Defn.Effect == instructions.Interrupt) ||
			(!dbg.VCS.CPU.LastResult.Final &&
				dbg.VCS.CPU.LastResult.Defn.Effect != instructions.Flow &&
				dbg.VCS.CPU.LastResult.Defn.Effect != instructions.Subroutine &&
				dbg.VCS.CPU.LastResult.Defn.Effect != instructions.Interrupt) {
			dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
			dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
			dbg.watchMessages = dbg.watches.check(dbg.watchMessages)
			stepTrapMessage = dbg.stepTraps.check("")
		}

		// check for halt conditions
		haltEmulation := stepTrapMessage != "" ||
			dbg.breakMessages != "" ||
			dbg.trapMessages != "" ||
			dbg.watchMessages != "" ||
			dbg.lastStepError || dbg.haltImmediately

		// expand halt to include step-once/many flag
		haltEmulation = haltEmulation || !dbg.runUntilHalt

		// take note of current machine state if the emulation was in a running
		// state and is halting just now
		if haltEmulation && dbg.continueEmulation {
			// but not for scripts
			if inputter.IsInteractive() {
				if !videoCycle {
					dbg.Rewind.CurrentState()
				}
			}
		}

		// reset last step error
		dbg.lastStepError = false

		// if emulation is to be halted or if we need to check the terminal
		if haltEmulation || checkTerm {
			// always clear steptraps. if the emulation has halted for any
			// reason then any existing step trap is stale.
			dbg.stepTraps.clear()

			// some things we don't want to if this is only a momentary halt
			if haltEmulation {
				// print and reset accumulated break/trap/watch messages
				dbg.printLine(terminal.StyleFeedback, dbg.breakMessages)
				dbg.printLine(terminal.StyleFeedback, dbg.trapMessages)
				dbg.printLine(terminal.StyleFeedback, dbg.watchMessages)
				dbg.breakMessages = ""
				dbg.trapMessages = ""
				dbg.watchMessages = ""

				// input has halted. print on halt command if it is defined
				if dbg.commandOnHalt != nil {
					err := dbg.processTokenGroup(dbg.commandOnHalt)
					if err != nil {
						dbg.printLine(terminal.StyleError, "%s", err)
					}
				}

				// pause tv when emulation has halted
				err = dbg.tv.Pause(true)
				if err != nil {
					return err
				}
				err = dbg.scr.ReqFeature(gui.ReqPause, true)
				if err != nil {
					return err
				}
			}

			// reset run until halt flag - it will be set again if the parsed
			// command requires it (eg. the RUN command)
			dbg.runUntilHalt = false

			// reset haltImmediately flag - it will be set again with the next
			// HALT command
			dbg.haltImmediately = false

			// get user input
			inputLen, err := inputter.TermRead(dbg.input, dbg.buildPrompt(), dbg.events)

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
				} else if curated.Is(err, script.ScriptEnd) {
					// a script that is being run will usually end with a ScriptEnd
					// error. in these instances we can say simply say so (using
					// the error message) with a feedback style (not an error
					// style)
					if !videoCycle {
						dbg.printLine(terminal.StyleFeedback, err.Error())
					}
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
				dbg.continueEmulation, err = dbg.parseInput(string(dbg.input[:inputLen-1]), inputter.IsInteractive(), false)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// if we stopped only to check the terminal then set continue and
			// runUntilHalt conditions
			if checkTerm {
				dbg.continueEmulation = true
				dbg.runUntilHalt = true
			}

			// unpause emulation if we're continuing emulation and this
			// *wasn't* a checkTerm pause. we don't want to send an unpause
			// request if we only entered HELP in the terminal, for example.
			if !checkTerm && dbg.runUntilHalt {
				// unpause GUI if there are no step traps. unpausing a GUI when
				// stepping by scanline, for example, looks ugly
				if dbg.stepTraps.isEmpty() {
					err = dbg.tv.Pause(false)
					if err != nil {
						return err
					}
					err = dbg.scr.ReqFeature(gui.ReqPause, false)
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

		if dbg.continueEmulation {
			// if this non-video-cycle input loop then
			if videoCycle {
				return nil
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
				stepErr = dbg.VCS.Step(vcsStep)
			case QuantumVideo:
				stepErr = dbg.VCS.Step(vcsStepVideo)
			default:
				stepErr = fmt.Errorf("unknown quantum mode")
			}

			// update rewind state if the last CPU instruction took place during a new
			// frame event
			if !videoCycle {
				dbg.Rewind.Check()
			}

			// check step error. note that we format and store last CPU
			// execution result whether there was an error or not. in the case
			// of an error the resul a fresh formatting. if there was no error
			// the formatted result is returned by the ExecutedEntry() function.

			if stepErr != nil {
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

					// if this is not a video cycle loop and the error has occurred in the middle of
					// a CPU cycle (ie. CPU result is not final) then issue an additional warning
					// and update CPU interrupt state
					//
					// !!TODO: errors that occur mid-CPU cycle to continue inside video-cycle loop
					// * this will need quite a bit of work with uncertain benefits
					//
					if !videoCycle && !dbg.lastResult.Result.Final {
						dbg.printLine(terminal.StyleError, "CPU halted mid-instruction. next step may be inaccurate.")
						dbg.VCS.CPU.Interrupted = true
					}
				}
			} else {
				// update entry and store result as last result
				dbg.lastResult, err = dbg.Disasm.ExecutedEntry(dbg.lastBank, dbg.VCS.CPU.LastResult, dbg.VCS.CPU.PC.Value())
				if err != nil {
					return err
				}

				// check validity of instruction result
				if dbg.VCS.CPU.LastResult.Final {
					err = dbg.VCS.CPU.LastResult.IsValid()
					if err != nil {
						dbg.printLine(terminal.StyleError, "%s", dbg.VCS.CPU.LastResult.Defn)
						dbg.printLine(terminal.StyleError, "%s", dbg.VCS.CPU.LastResult)
						return err
					}
				}
			}

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
