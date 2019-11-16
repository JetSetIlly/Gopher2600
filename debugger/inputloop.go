package debugger

import (
	"gopher2600/debugger/console"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/definitions"
	"os"
	"syscall"
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
		if dbg.vcs.CPU.LastResult.Defn.Effect == definitions.Flow ||
			dbg.vcs.CPU.LastResult.Defn.Effect == definitions.Subroutine ||
			dbg.vcs.CPU.LastResult.Defn.Effect == definitions.Interrupt {
			return nil
		}

		// display information about any CPU bugs that may have been triggered
		if dbg.reportCPUBugs && dbg.vcs.CPU.LastResult.Bug != "" {
			dbg.print(console.StyleInstrument, dbg.vcs.CPU.LastResult.Bug)
		}
	}

	dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
	dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
	dbg.watchMessages = dbg.watches.check(dbg.watchMessages)

	return dbg.relfectMonitor.Check()
}

// inputLoop has two modes, defined by the videoCycle argument. when videoCycle
// is true then user will be prompted every video cycle; when false the user
// is prompted every cpu cycle.
func (dbg *Debugger) inputLoop(inputter console.UserInput, videoCycle bool) error {
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
				dbg.print(console.StyleError, "%s", err)
			}
		}
		return dbg.inputLoop(inputter, true)
	}

	for dbg.running {
		// check for gui events and keyboard interrupts
		err = dbg.checkInterruptsAndEvents()
		if err != nil {
			dbg.print(console.StyleError, "%s", err)
		}

		// if debugger is no longer running after checking interrupts and
		// events then break for loop
		if !dbg.running {
			break // for loop
		}

		// this extra test is to prevent the video input loop from continuing
		// when step granularity has been switched to every cpu instruction - the
		// input loop will unravel and execution will continue in the main
		// inputLoop
		if videoCycle && !dbg.inputEveryVideoCycle && dbg.continueEmulation {
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
		dbg.print(console.StyleFeedback, dbg.breakMessages)
		dbg.print(console.StyleFeedback, dbg.trapMessages)
		dbg.print(console.StyleFeedback, dbg.watchMessages)

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
					dbg.print(console.StyleError, "%s", err)
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
			inputLen, err := inputter.UserRead(dbg.input, dbg.buildPrompt(videoCycle), dbg.guiChan, dbg.guiEventHandler)

			// errors returned by UserRead() functions are very rich. the
			// following block interprets the error carefully and proceeds
			// appropriately
			if err != nil {
				// if the error originated from outside of the emulation code
				// then it is probably serious or unexpected. we give up and
				// return it to the calling function
				if !errors.IsAny(err) {
					return err
				}

				// we now know the we have an Atari Error so we can safely
				// switch on the internal Errno
				switch err.(errors.AtariError).Errno {

				// user interrupts are triggered by the user (in a terminal
				// environment, usually by pressing ctrl-c)
				case errors.UserInterrupt:

					// if script input is being capture by a scriptScribe then
					// we the user interrupt event as a SCRIPT END
					// command.
					if dbg.scriptScribe.IsActive() {
						dbg.input = []byte("SCRIPT END")
						inputLen = 11

					} else if !inputter.IsInteractive() {
						// if the input loop is processing a non-interactive
						// session (a script) then we run the EXIT command
						// immediately, without asking the user
						dbg.input = []byte("EXIT")
						inputLen = 5

					} else {
						// a scriptScribe is not active nor is this a script
						// input loop. ask the user if they really want to quit
						confirm := make([]byte, 1)
						_, err := inputter.UserRead(confirm,
							console.Prompt{
								Content: "really quit (y/n) ",
								Style:   console.StylePromptConfirm},
							nil, nil)

						if err != nil {
							// another UserInterrupt has occurred. we treat
							// UserInterrupt as thought 'y' was pressed
							if errors.Is(err, errors.UserInterrupt) {
								confirm[0] = 'y'
							} else {
								dbg.print(console.StyleError, err.Error())
							}
						}

						// check if confirmation has been confirmed and run
						// EXIT command
						if confirm[0] == 'y' || confirm[0] == 'Y' {
							dbg.input = []byte("EXIT")
							inputLen = 5
						}
					}

				// user has asked to suspend the debuggin process (in a UNIX
				// terminal environment this is usually done with ctrl-z)
				case errors.UserSuspend:
					p, err := os.FindProcess(os.Getppid())
					if err != nil {
						dbg.print(console.StyleError, "debugger doesn't seem to have a parent process")
					} else {
						// send TSTP signal to parent proces
						p.Signal(syscall.SIGTSTP)
					}

				// a script that is being run will usually end with a ScriptEnd
				// error. in these instances we can say simply say so (using
				// the error message) with a feedback style (not an error
				// style)
				case errors.ScriptEnd:
					if !videoCycle {
						dbg.print(console.StyleFeedback, err.Error())
					}
					return nil

				// a GUI event has triggered an error
				case errors.GUIEventError:
					dbg.print(console.StyleError, err.Error())

				// all other errors are passed upwards to the calling function
				default:
					return err
				}
			}

			// parse user input, taking note of whether the emulation should
			// continue
			dbg.continueEmulation, err = dbg.parseInput(string(dbg.input[:inputLen-1]), inputter.IsInteractive(), false)
			if err != nil {
				dbg.print(console.StyleError, "%s", err)
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
				if dbg.inputEveryVideoCycle {
					err = dbg.vcs.Step(videoCycleWithInput)
				} else {
					err = dbg.vcs.Step(dbg.videoCycle)
				}

				if err != nil {
					// exit input loop only if error is not an AtariError...
					if !errors.IsAny(err) {
						return err
					}

					// ...set lastStepError instead and allow emulation to halt
					dbg.lastStepError = true
					dbg.print(console.StyleError, "%s", err)

				} else {
					// check validity of instruction result
					if dbg.vcs.CPU.LastResult.Final {
						err := dbg.vcs.CPU.LastResult.IsValid()
						if err != nil {
							dbg.print(console.StyleError, "%s", dbg.vcs.CPU.LastResult.Defn)
							dbg.print(console.StyleError, "%s", dbg.vcs.CPU.LastResult)
							return errors.New(errors.DebuggerError, err)
						}
					}
				}

				if dbg.commandOnStep != "" {
					_, err := dbg.parseInput(dbg.commandOnStep, false, true)
					if err != nil {
						dbg.print(console.StyleError, "%s", err)
					}
				}
			} else {
				return nil
			}
		}
	}

	return nil
}
