package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/monitor"
	"gopher2600/debugger/ui"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"os"
	"os/signal"
	"strings"
)

const defaultOnHalt = "CPU; TV"
const defaultOnStep = "LAST"

// Debugger is the basic debugging frontend for the emulation
type Debugger struct {
	vcs    *hardware.VCS
	disasm *disassembly.Disassembly
	tv     gui.GUI

	// control of debug/input loop:
	// 	o running - whether the debugger is to continue with the debugging loop
	// 	o runUntilHalt - repeat execution loop until a halt condition is encountered
	running      bool
	runUntilHalt bool

	// interface to the vcs memory with additional debugging functions
	// -- access to vcs memory from the debugger (eg. peeking and poking) is
	// most fruitfully performed through this structure
	dbgmem *memoryDebug

	// system monitor is a very low level mechanism for monitoring the state of
	// the cpu and of memory. it is checked every video cycle and interesting
	// changes noted and recorded.
	sysmon *monitor.SystemMonitor

	// halt conditions
	breakpoints *breakpoints
	traps       *traps
	watches     *watches

	// note that the UI probably allows the user to manually break or trap at
	// will, with for example, ctrl-c

	// we accumulate break, trap and watch messsages until we can service them
	// if the strings are empty then no break/trap/watch event has occurred
	breakMessages string
	trapMessages  string
	watchMessages string

	// any error from previous emulation step
	lastStepError bool

	// single-fire step traps. these are used for the STEP command, allowing
	// things like "STEP FRAME".
	// -- note that the hardware.VCS type has the StepFrames() function, we're
	// not using that here because this solution is more general and flexible
	stepTraps *traps

	// commandOnHalt says whether an sequence of commands should run automatically
	// when emulation halts. commandOnHaltPrev is the stored command sequence
	// used when ONHALT is called with no arguments
	// halt is a breakpoint or user intervention (ie. ctrl-c)
	commandOnHalt       string
	commandOnHaltStored string

	// similarly, commandOnStep is the sequence of commands to run afer every
	// cpu/video cycle
	commandOnStep       string
	commandOnStepStored string

	// machineInfoVerbose controls the verbosity of commands that echo machine state
	machineInfoVerbose bool

	// input loop fields. we're storing these here because inputLoop can be
	// called from within another input loop (via a video step callback) and we
	// want these properties to persist (when a video step input loop has
	// completed and we're back into the main input loop)
	inputloopHalt       bool // whether to halt the current execution loop
	inputloopNext       bool // execute a step once user input has returned a result
	inputloopVideoClock bool // step mode

	// the last result from vcs.Step() - could be a complete result or an
	// intermediate result when video-stepping
	lastResult *result.Instruction

	// user interface
	ui       ui.UserInterface
	uiSilent bool // controls whether UI is to remain silent

	// buffer for user input
	input []byte

	// channel for communicating with the debugging loop from other areas of
	// the emulation, paritcularly from other goroutines. it is used to:
	//  a) receive events from some other part of the emulation. for example,
	//  the SDL guiloop() goroutine
	//  b) receive ctrl-c events when the emulation is running (note that
	//  ctrl-c handling is handled differently under different circumstances)
	interruptChannel chan func()
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger() (*Debugger, error) {
	var err error

	dbg := new(Debugger)

	dbg.ui = new(ui.PlainTerminal)

	// prepare hardware
	dbg.tv, err = sdl.NewGUI("NTSC", 2.0)
	if err != nil {
		return nil, fmt.Errorf("error preparing television: %s", err)
	}
	dbg.tv.SetFeature(gui.ReqSetAllowDebugging, true)

	// create a new VCS instance
	dbg.vcs, err = hardware.NewVCS(dbg.tv)
	if err != nil {
		return nil, fmt.Errorf("error preparing VCS: %s", err)
	}

	// create instance of disassembly -- the same base structure is used
	// for disassemblies subseuquent to the first one.
	dbg.disasm = &disassembly.Disassembly{}

	// set up debugging interface to memory
	dbg.dbgmem = &memoryDebug{mem: dbg.vcs.Mem, symtable: &dbg.disasm.Symtable}

	// set up system monitor
	dbg.sysmon = &monitor.SystemMonitor{Mem: dbg.vcs.Mem, MC: dbg.vcs.MC, Rec: dbg.vcs.TV}

	// set up breakpoints/traps
	dbg.breakpoints = newBreakpoints(dbg)
	dbg.traps = newTraps(dbg)
	dbg.watches = newWatches(dbg)
	dbg.stepTraps = newTraps(dbg)

	// default ONHALT command sequence
	dbg.commandOnHaltStored = defaultOnHalt

	// default ONSTEP command sequnce
	dbg.commandOnStep = defaultOnStep
	dbg.commandOnStepStored = dbg.commandOnStep

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// make synchronisation channel
	dbg.interruptChannel = make(chan func(), 2)

	// set up callbacks for the TV interface
	// -- requires interruptChannel to have been set up
	err = dbg.setupTVCallbacks()
	if err != nil {
		return nil, err
	}

	return dbg, nil
}

// Start the main debugger sequence
func (dbg *Debugger) Start(iface ui.UserInterface, filename string, initScript string) error {
	// prepare user interface
	if iface != nil {
		dbg.ui = iface
	}

	err := dbg.ui.Initialise()
	if err != nil {
		return err
	}
	defer dbg.ui.CleanUp()

	dbg.ui.RegisterTabCompleter(input.NewTabCompletion(DebuggerCommands))

	err = dbg.loadCartridge(filename)
	if err != nil {
		return err
	}

	dbg.running = true

	// register a ctrl-c handler.
	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt)
	go func() {
		for {
			<-ctrlC

			dbg.interruptChannel <- func() {
				if dbg.runUntilHalt {
					// stop emulation at the next step
					dbg.runUntilHalt = false
				} else {
					// runUntilHalt is false which means that the emulation is
					// not running. at this point, an input loop is probably
					// running. note that ctrl-c signals do not always reach
					// this far into the program.  for instance, the colorterm
					// implementation of UserRead() puts the terminal into raw
					// mode and so must handle ctrl-c events differently.
					dbg.running = false
				}
			}
		}
	}()

	// run initialisation script
	if initScript != "" {
		spt, err := dbg.loadScript(initScript)
		if err != nil {
			dbg.print(ui.Error, "error running debugger initialisation script: %s\n", err)
		}

		err = dbg.inputLoop(spt, true)
		if err != nil {
			return err
		}
	}

	// prepare and run main input loop. inputLoop will not return until
	// debugger is to exit
	err = dbg.inputLoop(dbg.ui, true)
	if err != nil {
		return err
	}
	return nil
}

// loadCartridge makes sure that the cartridge loaded into vcs memory and the
// available disassembly/symbols are in sync.
//
// NEVER call vcs.AttachCartridge except through this funtion
//
// this is the glue that hold the cartridge and disassembly packages
// together
func (dbg *Debugger) loadCartridge(cartridgeFilename string) error {
	err := dbg.vcs.AttachCartridge(cartridgeFilename)
	if err != nil {
		return err
	}

	symtable, err := symbols.ReadSymbolsFile(cartridgeFilename)
	if err != nil {
		dbg.print(ui.Error, "%s", err)
		symtable = symbols.StandardSymbolTable()
	}

	err = dbg.disasm.ParseMemory(dbg.vcs.Mem.Cart, symtable)
	if err != nil {
		return err
	}

	err = dbg.vcs.TV.Reset()
	if err != nil {
		return err
	}

	return nil
}

// videoCycle() to be used with vcs.Step()
func (dbg *Debugger) videoCycle(result *result.Instruction) error {
	// because we call this callback mid-instruction, the programme counter
	// maybe in it's non-final state - we don't want to break or trap in these
	// instances if the final effect of the instruction changes the programme
	// counter to some other value
	if result.Defn != nil {
		if (result.Defn.Effect == definitions.Flow ||
			result.Defn.Effect == definitions.Subroutine ||
			result.Defn.Effect == definitions.Interrupt) &&
			!result.Final {
			return nil
		}
	}

	dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
	dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
	dbg.watchMessages = dbg.watches.check(dbg.watchMessages)

	return dbg.sysmon.Check()
}

func (dbg *Debugger) checkForInterrupts() {
	// check interrupt channel and run any functions we find in there
	select {
	case f := <-dbg.interruptChannel:
		f()
	default:
		// pro-tip: default case required otherwise the select will block
		// indefinately.
	}
}

// inputLoop has two modes, defined by the mainLoop argument. when inputLoop is
// not a "mainLoop", the function will only loop for the duration of one cpu
// step. this is used to implement video-stepping.
//
// inputter is an instance of type UserInput. this will usually be dbg.ui but
// it could equally be an instance of debuggingScript.
func (dbg *Debugger) inputLoop(inputter ui.UserInput, mainLoop bool) error {
	var err error

	// videoCycleWithInput() to be used with vcs.Step() instead of videoCycle()
	// when in video-step mode
	videoCycleWithInput := func(result *result.Instruction) error {
		dbg.videoCycle(result)
		dbg.lastResult = result
		if dbg.commandOnStep != "" {
			_, err := dbg.parseInput(dbg.commandOnStep)
			if err != nil {
				dbg.print(ui.Error, "%s", err)
			}
		}
		return dbg.inputLoop(inputter, false)
	}

	for {
		dbg.checkForInterrupts()
		if !dbg.running {
			break // for loop
		}

		// this extra test is to prevent the video input loop from continuing
		// if step mode has been switched to cpu - the input loop will unravel
		// and execution will continue in the main inputLoop
		if !mainLoop && !dbg.inputloopVideoClock && dbg.inputloopNext {
			return nil
		}

		// check for step-traps
		stepTrapMessage := dbg.stepTraps.check("")
		if stepTrapMessage != "" {
			dbg.stepTraps.clear()
		}

		// check for breakpoints and traps
		dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
		dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
		dbg.watchMessages = dbg.watches.check(dbg.watchMessages)

		// check for halt conditions
		dbg.inputloopHalt = stepTrapMessage != "" || dbg.breakMessages != "" || dbg.trapMessages != "" || dbg.watchMessages != "" || dbg.lastStepError

		// reset last step error
		dbg.lastStepError = false

		// if commandOnHalt is defined and if run state is correct then run
		// commandOnHalt command(s)
		if dbg.commandOnHalt != "" {
			if (dbg.inputloopNext && !dbg.runUntilHalt) || dbg.inputloopHalt {
				_, err = dbg.parseInput(dbg.commandOnHalt)
				if err != nil {
					dbg.print(ui.Error, "%s", err)
				}
			}
		}

		// print and reset accumulated break and trap messages
		dbg.print(ui.Feedback, dbg.breakMessages)
		dbg.print(ui.Feedback, dbg.trapMessages)
		dbg.print(ui.Feedback, dbg.watchMessages)
		dbg.breakMessages = ""
		dbg.trapMessages = ""
		dbg.watchMessages = ""

		// expand inputloopHalt to include step-once/many flag
		dbg.inputloopHalt = dbg.inputloopHalt || !dbg.runUntilHalt

		// enter halt state
		if dbg.inputloopHalt {
			// pause tv when emulation has halted
			err = dbg.tv.SetFeature(gui.ReqSetPause, true)
			if err != nil {
				return err
			}

			dbg.runUntilHalt = false

			// decide which PC value to use
			var disasmPC uint16
			if dbg.lastResult == nil || dbg.lastResult.Final {
				disasmPC = dbg.vcs.MC.PC.ToUint16()
			} else {
				disasmPC = dbg.lastResult.Address
			}

			// build prompt
			var prompt string
			if r, ok := dbg.disasm.Get(dbg.disasm.Cart.Bank, disasmPC); ok {
				prompt = strings.Trim(r.GetString(dbg.disasm.Symtable, result.StyleBrief), " ")
				prompt = fmt.Sprintf("[ %s ] > ", prompt)
			} else {
				// incomplete disassembly, prepare witchspace prompt
				// TODO: implement "just in time" disassembly
				prompt = fmt.Sprintf("[witchspace (%d, %#04x)] > ", dbg.disasm.Cart.Bank, disasmPC)
			}

			// - additional annotation if we're not showing the prompt in the main loop
			if !mainLoop && !dbg.lastResult.Final {
				prompt = fmt.Sprintf("+ %s", prompt)
			}

			// get user input
			n, err := inputter.UserRead(dbg.input, prompt, dbg.interruptChannel)
			if err != nil {
				switch err := err.(type) {

				case errors.FormattedError:
					switch err.Errno {
					case errors.UserInterrupt:
						dbg.running = false
						fallthrough
					case errors.ScriptEnd:
						if mainLoop {
							dbg.print(ui.Feedback, err.Error())
						}
						return nil
					}
				}

				return err
			}

			dbg.checkForInterrupts()
			if !dbg.running {
				break // for loop
			}

			// parse user input
			dbg.inputloopNext, err = dbg.parseInput(string(dbg.input[:n-1]))
			if err != nil {
				dbg.print(ui.Error, "%s", err)
			}

			// prepare for next loop
			dbg.inputloopHalt = false

			// make sure tv is unpaused if emulation is about to resume
			if dbg.inputloopNext {
				err = dbg.tv.SetFeature(gui.ReqSetPause, false)
				if err != nil {
					return err
				}
			}
		}

		// move emulation on one step if user has requested/implied it
		if dbg.inputloopNext {
			if mainLoop {
				if dbg.inputloopVideoClock {
					_, dbg.lastResult, err = dbg.vcs.Step(videoCycleWithInput)
				} else {
					_, dbg.lastResult, err = dbg.vcs.Step(dbg.videoCycle)
				}

				if err != nil {
					switch err := err.(type) {
					case errors.FormattedError:
						// do not exit input loop when error is a gopher error
						// set lastStepError instead and allow emulation to
						// halt
						dbg.lastStepError = true

						// print gopher error message
						dbg.print(ui.Error, "%s", err)
					default:
						return err
					}
				} else {
					// check validity of instruction result
					if dbg.lastResult.Final {
						err := dbg.lastResult.IsValid()
						if err != nil {
							dbg.print(ui.Error, "%s", dbg.lastResult.Defn)
							dbg.print(ui.Error, "%s", dbg.lastResult)
							panic(err)
						}
					}
				}

				if dbg.commandOnStep != "" {
					_, err := dbg.parseInput(dbg.commandOnStep)
					if err != nil {
						dbg.print(ui.Error, "%s", err)
					}
				}
			} else {
				return nil
			}
		}
	}

	return nil
}

// parseInput splits the input into individual commands. each command is then
// passed to parseCommand for final processing
func (dbg *Debugger) parseInput(input string) (bool, error) {
	var cont bool
	var err error

	input = strings.TrimSpace(input)

	// ignore comments
	if strings.HasPrefix(input, "#") {
		return false, nil
	}

	commands := strings.Split(input, ";")
	for i := 0; i < len(commands); i++ {
		cont, err = dbg.parseCommand(commands[i])
		if err != nil {
			return false, err
		}
	}

	return cont, nil
}
