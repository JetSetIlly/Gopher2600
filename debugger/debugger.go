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
	disasm disassembly.Disassembly

	// control of debug/input loop:
	// 	o running - whether the debugger is to continue with the debugging loop
	// 	o runUntilHalt - repeat execution loop until a halt condition is encountered
	running      bool
	runUntilHalt bool

	// halt conditions
	// note that the UI probably allows the user to halt (eg. ctrl-c)
	breakpoints *breakpoints
	traps       *traps

	// we accumulate break and trap messsages until we can service them
	breakMessages string
	trapMessages  string

	// any error from previous emulation step
	lastStepError bool

	// commandOnHalt says whether an sequence of commands should run automatically
	// when emulation halts. commandOnHaltPrev is the stored command sequence
	// used when ONHALT is called with no arguments
	// halt is a breakpoint or user intervention (ie. ctrl-c)
	commandOnHalt       string
	commandOnHaltStored string

	// similarly, commandOnStep is the sequence of commands to run afer ever
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
	// the emulation, paritcularly from other goroutines. for instance, we use
	// the syncChannel to implement ctrl-c handling, even though all the code
	// to do it is right here in this very file. we do this to avoid having to
	// use sync.Mutex. marking critical sections with a mutex is fine and would
	// definitiely work. However it is, frankly, a pain, messy and feels wrong.
	syncChannel chan func()
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger() (*Debugger, error) {
	var err error

	dbg := new(Debugger)

	dbg.ui = new(ui.PlainTerminal)

	// prepare hardware
	tv, err := sdltv.NewSDLTV("NTSC", sdltv.IdealScale)
	if err != nil {
		return nil, fmt.Errorf("error preparing television: %s", err)
	}

	dbg.vcs, err = hardware.NewVCS(tv)
	if err != nil {
		return nil, fmt.Errorf("error preparing VCS: %s", err)
	}

	// set up breakpoints/traps
	dbg.breakpoints = newBreakpoints(dbg)
	dbg.traps = newTraps(dbg)

	// default ONHALT command squence
	dbg.commandOnHaltStored = defaultOnHalt

	// default ONSTEP command sequnce
	dbg.commandOnStep = defaultOnStep
	dbg.commandOnStepStored = dbg.commandOnStep

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// make synchronisation channel
	dbg.syncChannel = make(chan func(), 2)

	// register tv callbacks
	err = tv.RegisterCallback(television.ReqOnMouseButtonRight, dbg.syncChannel, func() {
		// this callback function may be running inside a different goroutine
		// so care must be taken not to cause a deadlock
		hp, _ := dbg.vcs.TV.RequestTVInfo(television.ReqLastMouseX)
		sl, _ := dbg.vcs.TV.RequestTVInfo(television.ReqLastMouseY)

		dbg.parseCommand(fmt.Sprintf("%s sl %s & hp %s", KeywordBreak, sl, hp))

		// if the emulation is running the new break should cause it to halt
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
		loop := true
		for loop {
			<-ctrlC

			dbg.syncChannel <- func() {
				if dbg.runUntilHalt {
					dbg.runUntilHalt = false
				} else {
					// TODO: interrupt os.stdin.Read() in plain terminal, so that
					// the user doesn't have to press return after a ctrl-c press
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
// available disassembly/symbols are in sync. *never call vcs.AttachCartridge
// except through this funtion*
func (dbg *Debugger) loadCartridge(cartridgeFilename string) error {
	err := dbg.vcs.AttachCartridge(cartridgeFilename)
	if err != nil {
		return err
	}

	symtable, err := symbols.ReadSymbolsFile(cartridgeFilename)
	if err != nil {
		dbg.print(ui.Error, "%s", err)
		symtable, err = symbols.StandardSymbolTable()
		if err != nil {
			return err
		}
	}

	err = dbg.disasm.ParseMemory(dbg.vcs.Mem, symtable)
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
	dbg.breakandtrapCallback(result)
	dbg.lastResult = result
	if dbg.commandOnStep != "" {
		_, err := dbg.parseInput(dbg.commandOnStep)
		if err != nil {
			dbg.print(ui.Error, "%s", err)
		}
	}
	return dbg.inputLoop(false)
}

func (dbg *Debugger) breakandtrapCallback(result *result.Instruction) error {
	// because we call this callback mid-instruction, the programme counter
	// maybe in it's non-final state - we don't want to break or trap in these
	// instances if the final effect of the instruction changes the programme
	// counter to some other value
	if (result.Defn.Effect == definitions.Flow || result.Defn.Effect == definitions.Subroutine) && !result.Final {
		return nil
	}

	dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
	dbg.trapMessages = dbg.traps.check(dbg.trapMessages)

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

		// check syncChannel and run any functions we find in there
		// TODO: not sure if this is the best part of the loop to put this
		// check. it works for now.
		select {
		case f := <-dbg.syncChannel:
			f()
		default:
		}

		// check for breakpoints and traps
		dbg.breakMessages = dbg.breakpoints.check(dbg.breakMessages)
		dbg.trapMessages = dbg.traps.check(dbg.trapMessages)
		dbg.inputloopHalt = dbg.breakMessages != "" || dbg.trapMessages != "" || dbg.lastStepError

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
		dbg.breakMessages = ""
		dbg.trapMessages = ""

		// expand inputloopHalt to include step-once/many flag
		dbg.inputloopHalt = dbg.inputloopHalt || !dbg.runUntilHalt

		if dbg.inputloopHalt {
			// pause tv when emulation has halted
			err = dbg.vcs.TV.SetPause(true)
			if err != nil {
				return err
			}

			dbg.runUntilHalt = false

			// build prompt
			// - different prompt depending on whether a valid disassembly is available
			var prompt string
			if p, ok := dbg.disasm.Program[dbg.vcs.MC.PC.ToUint16()]; ok {
				prompt = strings.Trim(p.GetString(dbg.disasm.Symtable, result.StyleBrief), " ")
				prompt = fmt.Sprintf("[ %s ] > ", prompt)
			} else {
				prompt = fmt.Sprintf("[ %#04x ] > ", dbg.vcs.MC.PC.ToUint16())
			}
			// - additional annotation if we're not showing the prompt in the main loop
			if !mainLoop && !dbg.lastResult.Final {
				prompt = fmt.Sprintf("+ %s", prompt)
			}

			// get user input
			n, err := dbg.ui.UserRead(dbg.input, prompt)
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
				err = dbg.vcs.TV.SetPause(false)
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
					_, dbg.lastResult, err = dbg.vcs.Step(dbg.breakandtrapCallback)
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

	commands := strings.Split(input, ";")
	for i := 0; i < len(commands); i++ {
		cont, err = dbg.parseCommand(commands[i])
		if err != nil {
			return false, err
		}
	}

	return cont, nil
}
