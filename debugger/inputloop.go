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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"gopher2600/debugger/terminal"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/instructions"
	"io"
)

// videoCycle() to be used with vcs.Step()
//
// compare the use of this function with videoCycleWithInput() function
// defined inside the inputLoop() function
func (dbg *Debugger) videoCycle() error {
	// because we call this callback mid-instruction, the program counter
	// maybe in its non-final state - we don't want to break or trap in those
	// instances when the final effect of the instruction changes the program
	// counter to some other value (ie. a flow, subroutine or interrupt
	// instruction)
	if !dbg.vcs.CPU.LastResult.Final &&
		dbg.vcs.CPU.LastResult.Defn != nil {
		if dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Flow ||
			dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Subroutine ||
			dbg.vcs.CPU.LastResult.Defn.Effect == instructions.Interrupt {
			return nil
		}
	}

	dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
	dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
	dbg.watchMessages = dbg.watches.check(dbg.watchMessages)

	return dbg.reflect.Check()
}

// inputLoop has two modes, defined by the videoCycle argument. when videoCycle
// is true then user will be prompted every video cycle; when false the user
// is prompted every cpu cycle.
func (dbg *Debugger) inputLoop(inputter terminal.Input, videoCycle bool) error {
	var err error

	// videoCycleWithInput() to be used with vcs.Step() instead of videoCycle()
	// when in video-step mode
	//
	// (we're defining the function here because we want the inputter instance
	// to be enclosed)
	//
	// compare the use of this function with Debugger.videoCycle() function
	// defined elsewhere
	videoCycleWithInput := func() error {
		dbg.videoCycle()
		if dbg.commandOnStep != "" {
			_, err := dbg.parseInput(dbg.commandOnStep, false, true)
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			}
		}
		return dbg.inputLoop(inputter, true)
	}

	for dbg.running {
		// check for events
		err = dbg.checkEvents()
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
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
		if videoCycle && dbg.continueEmulation && dbg.quantum == quantumCPU {
			return nil
		}

		// check for breakpoints and traps
		dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
		dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
		dbg.watchMessages = dbg.watches.check(dbg.watchMessages)
		stepTrapMessage := dbg.stepTraps.check("")

		// check for halt conditions
		dbg.haltEmulation = stepTrapMessage != "" ||
			dbg.breakMessages != "" ||
			dbg.trapMessages != "" ||
			dbg.watchMessages != "" ||
			dbg.lastStepError

		// expand halt to include step-once/many flag
		dbg.haltEmulation = dbg.haltEmulation || !dbg.runUntilHalt

		// step traps are cleared once they have been encountered
		if stepTrapMessage != "" {
			dbg.stepTraps.clear()
		}

		// print and reset accumulated break/trap/watch messages
		dbg.printLine(terminal.StyleFeedback, dbg.breakMessages)
		dbg.printLine(terminal.StyleFeedback, dbg.trapMessages)
		dbg.printLine(terminal.StyleFeedback, dbg.watchMessages)

		// clear accumulated break/trap/watch messages
		dbg.breakMessages = ""
		dbg.trapMessages = ""
		dbg.watchMessages = ""

		// reset last step error
		dbg.lastStepError = false

		// something has happened to cause the emulation to halt
		if dbg.haltEmulation {
			// input has halted. print on halt command if it is defined
			if dbg.commandOnHalt != "" {
				_, err = dbg.parseInput(dbg.commandOnHalt, false, true)
				if err != nil {
					dbg.printLine(terminal.StyleError, "%s", err)
				}
			}

			// pause tv when emulation has halted
			err = dbg.scr.SetFeature(gui.ReqSetPause, true)
			if err != nil {
				return err
			}

			// reset run until halt flag - it will be set again if the parsed command requires it
			// (eg. the RUN command)
			dbg.runUntilHalt = false

			// get user input
			inputLen, err := inputter.TermRead(dbg.input, dbg.buildPrompt(videoCycle), dbg.events)

			// errors returned by UserRead() functions are very rich. the
			// following block interprets the error carefully and proceeds
			// appropriately
			if err != nil {
				if !errors.IsAny(err) {
					// if the error originated from outside of gopher2600 then
					// it is probably serious or unexpected
					switch err {
					case io.EOF:
						// treat EOF events the same as UserInterrupt events
						err = errors.New(errors.UserInterrupt)
					default:
						// the error is probably serious. exit input loop with
						// err
						return err
					}
				}

				// we now know the we have an Atari Error so we can safely
				// switch on the internal Errno
				switch err.(errors.AtariError).Message {

				// user interrupts are triggered by the user (in a terminal
				// environment, usually by pressing ctrl-c)
				case errors.UserInterrupt:
					dbg.handleInterrupt(inputter, inputLen)

				// like UserInterrupt but with no confirmation stage
				case errors.UserQuit:
					dbg.running = false

				// a script that is being run will usually end with a ScriptEnd
				// error. in these instances we can say simply say so (using
				// the error message) with a feedback style (not an error
				// style)
				case errors.ScriptEnd:
					if !videoCycle {
						dbg.printLine(terminal.StyleFeedback, err.Error())
					}
					return nil

				// a GUI event has triggered an error
				case errors.GUIEventError:
					dbg.printLine(terminal.StyleError, err.Error())

				// all other errors are passed upwards to the calling function
				default:
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

			// prepare for next loop
			dbg.haltEmulation = false

			// if continueEmulation is set at the end of the haltEmulation
			// block, then unpause GUI
			if dbg.continueEmulation {
				err = dbg.scr.SetFeature(gui.ReqSetPause, false)
				if err != nil {
					return err
				}
			}
		}

		if dbg.continueEmulation {
			// if this non-video-cycle input loop then
			if !videoCycle {
				switch dbg.quantum {
				case quantumCPU:
					err = dbg.vcs.Step(dbg.videoCycle)
				case quantumVideo:
					err = dbg.vcs.Step(videoCycleWithInput)
				default:
					err = errors.New(errors.DebuggerError, "unknown quantum mode")
				}

				if err != nil {
					// exit input loop only if error is not an AtariError...
					if !errors.IsAny(err) {
						return err
					}

					// ...set lastStepError instead and allow emulation to halt
					dbg.lastStepError = true
					dbg.printLine(terminal.StyleError, "%s", err)

				} else {
					// check validity of instruction result
					if dbg.vcs.CPU.LastResult.Final {
						err := dbg.vcs.CPU.LastResult.IsValid()
						if err != nil {
							dbg.printLine(terminal.StyleError, "%s", dbg.vcs.CPU.LastResult.Defn)
							dbg.printLine(terminal.StyleError, "%s", dbg.vcs.CPU.LastResult)
							return errors.New(errors.DebuggerError, err)
						}
					}
				}

				if dbg.commandOnStep != "" {
					_, err := dbg.parseInput(dbg.commandOnStep, false, true)
					if err != nil {
						dbg.printLine(terminal.StyleError, "%s", err)
					}
				}
			} else {
				return nil
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
// - otherwise, prompt use for confirmation that the debugger should quit
func (dbg *Debugger) handleInterrupt(inputter terminal.Input, inputLen int) {
	// if script input is being capture by a scriptScribe then
	// we the user interrupt event as a SCRIPT END
	// command.
	if dbg.scriptScribe.IsActive() {
		dbg.input = []byte("SCRIPT END")
		inputLen = 11

	} else if !inputter.IsInteractive() {
		dbg.running = false

	} else {
		// a scriptScribe is not active nor is this a script
		// input loop. ask the user if they really want to quit
		confirm := make([]byte, 1)
		_, err := inputter.TermRead(confirm,
			terminal.Prompt{
				Content: "really quit (y/n) ",
				Style:   terminal.StylePromptConfirm},
			dbg.events)

		if err != nil {
			// another UserInterrupt has occurred. we treat
			// UserInterrupt as thought 'y' was pressed
			if errors.Is(err, errors.UserInterrupt) {
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
