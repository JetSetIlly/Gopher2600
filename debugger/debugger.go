package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"gopher2600/television"
	"gopher2600/television/sdltv"
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

	// control of debug/input loop:
	// 	o running - whether the debugger is to continue with the debugging loop
	// 	o runUntilHalt - repeat execution loop until a halt condition is encountered
	running      bool
	runUntilHalt bool

	// interface to the vcs memory with additional debugging functions
	// -- access to vcs memory from the debugger is most fruitfully performed
	// through this structure
	dbgmem *memoryDebug

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
	// the emulation, paritcularly from other goroutines.
	// -- it is used primarily to communicate with SDL gui goroutine
	// -- we also use the dbgChannel to implement ctrl-c handling. even though
	// all the code to do it is right here in this very file, we do this to
	// avoid having to use sync.Mutex.
	// -- we could improve this by allowing the function to take arguments and
	// to return a value but there's no real use-case for this at the moment
	// but it's something to consider (see colorterminal/input.go for possible
	// reason)
	dbgChannel chan func()
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger() (*Debugger, error) {
	var err error

	dbg := new(Debugger)

	dbg.ui = new(ui.PlainTerminal)

	// prepare hardware
	tv, err := sdltv.NewSDLTV("NTSC", 2.0)
	if err != nil {
		return nil, fmt.Errorf("error preparing television: %s", err)
	}

	dbg.vcs, err = hardware.NewVCS(tv)
	if err != nil {
		return nil, fmt.Errorf("error preparing VCS: %s", err)
	}

	// create instance of disassembly -- the same base structure is used
	// for disassemblies subseuquent to the first one.
	dbg.disasm = new(disassembly.Disassembly)

	// set up debugging interface to memory
	dbg.dbgmem = newMemoryDebug(dbg)

	// set up breakpoints/traps
	dbg.breakpoints = newBreakpoints(dbg)
	dbg.traps = newTraps(dbg)
	dbg.watches = newWatches(dbg)
	dbg.stepTraps = newTraps(dbg)

	// default ONHALT command squence
	dbg.commandOnHaltStored = defaultOnHalt

	// default ONSTEP command sequnce
	dbg.commandOnStep = defaultOnStep
	dbg.commandOnStepStored = dbg.commandOnStep

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// make synchronisation channel
	dbg.dbgChannel = make(chan func(), 2)

	// register tv callbacks
	// -- add break on right mouse button
	err = tv.RegisterCallback(television.ReqOnMouseButtonRight, dbg.dbgChannel, func() {
		// this callback function may be running inside a different goroutine
		// so care must be taken not to cause a deadlock
		hp, _ := dbg.vcs.TV.GetMetaState(television.ReqLastMouseHorizPos)
		sl, _ := dbg.vcs.TV.GetMetaState(television.ReqLastMouseScanline)

		dbg.print(ui.Feedback, "mouse break on sl->%s and hp->%s", sl, hp)
		dbg.parseCommand(fmt.Sprintf("%s sl %s & hp %s", KeywordBreak, sl, hp))
	})
	if err != nil {
		return nil, err
	}

	return dbg, nil
}

// Start the main debugger sequence
func (dbg *Debugger) Start(interf ui.UserInterface, filename string, initScript string) error {
	// prepare user interface
	if interf != nil {
		dbg.ui = interf
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

	// make sure we've indicated that the debugger is running before we start
	// the ctrl-c handler. it'll return immediately if we don't
	dbg.running = true

	// register ctrl-c handler
	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt)
	go func() {
		for {
			<-ctrlC

			dbg.dbgChannel <- func() {
				if dbg.runUntilHalt {
					dbg.runUntilHalt = false
				} else {
					dbg.running = false
				}
			}
		}
	}()

	// run initialisation script
	if initScript != "" {
		err = dbg.RunScript(initScript, true)
		if err != nil {
			dbg.print(ui.Error, "* error running debugger initialisation script: %s\n", err)
		}
	}

	// prepare and run main input loop. inputLoop will not return until
	// debugger is to exit
	err = dbg.inputLoop(true)
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

// videoCycleCallback() and breakandtrapCallback() are wrapper functions to be
// used when calling vcs.Step(). stepmode CPU uses breakandtrapCallback(),
// whereas stepmode VIDEO uses videoCycleCallback() which in turn uses
// breakandtrapCallback()

func (dbg *Debugger) videoCycleCallback(result *result.Instruction) error {
	dbg.breakAndTrapCallback(result)
	dbg.lastResult = result
	if dbg.commandOnStep != "" {
		_, err := dbg.parseInput(dbg.commandOnStep)
		if err != nil {
			dbg.print(ui.Error, "%s", err)
		}
	}
	return dbg.inputLoop(false)
}

func (dbg *Debugger) breakAndTrapCallback(result *result.Instruction) error {
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

	return nil
}

// inputLoop has two modes, defined by the mainLoop argument. when inputLoop is
// not a "mainLoop", the function will only loop for the duration of one cpu
// step. this is used to implement video-stepping.
func (dbg *Debugger) inputLoop(mainLoop bool) error {
	var err error

	for {
		if !dbg.running {
			break // for loop
		}

		// this extra test is to prevent the video input loop from continuing
		// if step mode has been switched to cpu - the input loop will unravel
		// and execution will continue in the main inputLoop
		if !mainLoop && !dbg.inputloopVideoClock && dbg.inputloopNext {
			return nil
		}

		// check dbgChannel and run any functions we find in there -- note that
		// we also monitor the dbgChannel in the UserInterface.UserRead()
		// function. if we didn't then this inputLoop would not react to
		// messages on the channel until it reaches this point again.
		select {
		case f := <-dbg.dbgChannel:
			f()
		default:
			// go novice note: we need a default case otherwise the select blocks
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
				_, _ = dbg.parseInput(dbg.commandOnHalt)
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
			err = dbg.vcs.TV.SetFeature(television.ReqSetPause, true)
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
			// - different prompt depending on whether a valid disassembly is available
			var prompt string
			if disasmRes, ok := dbg.disasm.Program[dbg.disasm.Cart.Bank][disasmPC]; ok {
				prompt = strings.Trim(disasmRes.GetString(dbg.disasm.Symtable, result.StyleBrief), " ")
				prompt = fmt.Sprintf("[ %s ] > ", prompt)
			} else {
				// we should have a valid entry from the disassembly, if we
				// don't then say so and prepare a suitable prompt
				dbg.print(ui.Error, "something went wrong with the disassembly (no instruction at this address)")
				prompt = fmt.Sprintf("[ *witchspace* (%d, %#04x)] > ", dbg.disasm.Cart.Bank, disasmPC)
			}

			// - additional annotation if we're not showing the prompt in the main loop
			if !mainLoop && !dbg.lastResult.Final {
				prompt = fmt.Sprintf("+ %s", prompt)
			}

			// get user input
			n, err := dbg.ui.UserRead(dbg.input, prompt, dbg.dbgChannel)
			if err != nil {
				switch err.(type) {
				case *ui.UserInterrupt:
					dbg.print(ui.Feedback, err.Error())
					dbg.running = false
					return nil
				default:
					return err
				}
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
				err = dbg.vcs.TV.SetFeature(television.ReqSetPause, false)
				if err != nil {
					return err
				}
			}
		}

		// move emulation on one step if user has requested/implied it
		if dbg.inputloopNext {
			if mainLoop {
				if dbg.inputloopVideoClock {
					_, dbg.lastResult, err = dbg.vcs.Step(dbg.videoCycleCallback)
				} else {
					_, dbg.lastResult, err = dbg.vcs.Step(dbg.breakAndTrapCallback)
				}

				if err != nil {
					switch err := err.(type) {
					case errors.GopherError:
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
