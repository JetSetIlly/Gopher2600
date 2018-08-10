package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"gopher2600/television/sdltv"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
)

const defaultOnHalt = "CPU; TV"
const defaultOnStep = "LAST"

// Debugger is the basic debugging frontend for the emulation
type Debugger struct {
	vcs    *hardware.VCS
	disasm disassembly.Disassembly

	// control of debug/input loop - accessed from within a sub-go routine so
	// both protected with a sync.mutex
	// 	o running - whether the debugger is to continue with the debugging loop
	// 	o runUntilHalt - repeat execution loop until a halt condition is encountered
	runLock      sync.Mutex
	running      bool
	runUntilHalt bool

	// halt conditions
	// note that the UI probably allows the user to halt (eg. ctrl-c)
	breakpoints *breakpoints
	traps       *traps

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
	//
	// *CRITICAL SECTION* *NOT REQUIRED* go routine not started yet
	dbg.running = true

	// register ctrl-c handler -- note that this controls the debugging
	// inputLoop() below. when that loop calls debugger.ui.UserRead(), that
	// function may have its own loop that may delay the side-effects of our
	// handler below. compare plain terminal and colour terminal
	// implementations. the latter terminal responds to ctrl-c in its input
	// loop and curtails the input loop immediately, debugger.inputLoop() to
	// react to the changes caused by our signal handler.
	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt)
	go func() {
		loop := true
		for loop {
			<-ctrlC

			// *CRITICAL SECTION* after signal received
			dbg.runLock.Lock()
			if dbg.runUntilHalt {
				dbg.runUntilHalt = false
			} else {
				// TODO: interrupt os.stdin.Read() in plain terminal, so that
				// the user doesn't have to press return after a ctrl-c press
				dbg.running = false
				loop = false
			}
			dbg.runLock.Unlock()
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

// videoCycleCallback() and noVideoCycleCallback() are wrapper functions to be
// used when calling vcs.Step() -- video stepping uses the former and cpu
// stepping uses the latter

func (dbg *Debugger) videoCycleCallback(result *result.Instruction) error {
	dbg.lastResult = result
	if dbg.commandOnStep != "" {
		_, err := dbg.parseInput(dbg.commandOnStep)
		if err != nil {
			dbg.print(ui.Error, "%s", err)
		}
	}
	return dbg.inputLoop(false)
}

func (dbg *Debugger) noVideoCycleCallback(result *result.Instruction) error {
	return nil
}

// inputLoop has two modes, defined by the mainLoop argument. when inputLoop is
// not a "mainLoop", the function will only loop for the duration of one cpu
// step. this is used to implement video-stepping.
func (dbg *Debugger) inputLoop(mainLoop bool) error {
	var err error

	for {
		// *CRITICAL SECTION*
		dbg.runLock.Lock()
		if !dbg.running {
			dbg.runLock.Unlock()
			break // for loop
		}
		dbg.runLock.Unlock()

		// return immediately if we're in a mid-cycle input loop and we don't want
		// to be
		//
		// the extra condition (dbg.inputLoopNext) is to prevent execution
		// continuing if we call "stepmode cpu" in the middle of a cpu-cycle
		if !mainLoop && !dbg.inputloopVideoClock && dbg.inputloopNext {
			return nil
		}

		// check for breakpoints and traps. check() functions echo all the
		// conditions that match
		if dbg.inputloopNext {
			bpCheck := dbg.breakpoints.check()
			trCheck := dbg.traps.check()
			dbg.inputloopHalt = bpCheck || trCheck || dbg.lastStepError
		}

		// reset last step error
		dbg.lastStepError = false

		// *CRITICAL SECTION*
		dbg.runLock.Lock()

		// if haltCommand mode and if run state is correct that print haltCommand
		// command(s)
		if dbg.commandOnHalt != "" {
			if (dbg.inputloopNext && !dbg.runUntilHalt) || dbg.inputloopHalt {
				// note this is parsing input, not reading input. we're passing the
				// parse function a prepared command sequence.
				_, _ = dbg.parseInput(dbg.commandOnHalt)
			}
		}

		// expand breakpoint to include step-once/many flag
		dbg.inputloopHalt = dbg.inputloopHalt || !dbg.runUntilHalt

		dbg.runLock.Unlock()
		// *END CRITICAL SECTION*

		if dbg.inputloopHalt {
			// pause tv when emulation has halted
			err = dbg.vcs.TV.SetPause(true)
			if err != nil {
				return err
			}

			// *CRITICAL SECTION*
			dbg.runLock.Lock()
			dbg.runUntilHalt = false
			dbg.runLock.Unlock()

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

					// *CRITICAL SECTION*
					dbg.runLock.Lock()
					dbg.running = false
					dbg.runLock.Unlock()

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
			//  o forget about current break state
			//  o prepare for matching on next breakpoint
			dbg.inputloopHalt = false
			dbg.breakpoints.prepareBreakpoints()

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
					_, dbg.lastResult, err = dbg.vcs.Step(dbg.noVideoCycleCallback)
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

// parseCommand scans user input for valid commands and acts upon it. commands
// that cause the emulation to move forward (RUN, STEP) return true for the
// first return value. other commands return false and act upon the command
// immediately. note that the empty string is the same as the STEP command
func (dbg *Debugger) parseCommand(userInput string) (bool, error) {
	// TODO: categorise commands into script-safe and non-script-safe

	// tokenise input
	tokens := input.TokeniseInput(userInput)

	// check validity of input -- this allows us to catch errors early and in
	// many cases to ignore the "success" flag when calling tokens.item()
	if err := DebuggerCommands.ValidateInput(tokens); err != nil {
		switch err := err.(type) {
		case errors.GopherError:
			switch err.Errno {
			case errors.InputEmpty:
				// user pressed return
				return true, nil
			}
		}
		return false, err
	}

	// most commands do not cause the emulator to step forward
	stepNext := false

	tokens.Reset()
	command, _ := tokens.Get()
	command = strings.ToUpper(command)
	switch command {
	default:
		return false, fmt.Errorf("%s is not yet implemented", command)

		// control of the debugger
	case KeywordHelp:
		keyword, present := tokens.Get()
		if present {
			s := strings.ToUpper(keyword)
			txt, prs := Help[s]
			if prs == false {
				dbg.print(ui.Help, "no help for %s", s)
			} else {
				dbg.print(ui.Help, txt)
			}
		} else {
			for k := range DebuggerCommands {
				dbg.print(ui.Help, k)
			}
		}

	case KeywordInsert:
		cart, _ := tokens.Get()
		err := dbg.loadCartridge(cart)
		if err != nil {
			return false, err
		}
		dbg.print(ui.Feedback, "machine reset with new cartridge (%s)", cart)

	case KeywordScript:
		script, _ := tokens.Get()
		err := dbg.RunScript(script, false)
		if err != nil {
			return false, err
		}

	case KeywordDisassemble:
		dbg.print(ui.CPUStep, dbg.disasm.Dump())

	case KeywordSymbol:
		symbol, _ := tokens.Get()
		address, err := dbg.disasm.Symtable.SearchLocation(symbol)
		if err != nil {
			switch err := err.(type) {
			case errors.GopherError:
				if err.Errno == errors.SymbolUnknown {
					dbg.print(ui.Feedback, "%s -> not found", symbol)
					return false, nil
				}
			}
			return false, err
		}
		dbg.print(ui.Feedback, "%s -> %#04x", symbol, address)

	case KeywordBreak:
		err := dbg.breakpoints.parseBreakpoint(tokens)
		if err != nil {
			return false, fmt.Errorf("error on break: %s", err)
		}

	case KeywordTrap:
		err := dbg.traps.parseTrap(tokens)
		if err != nil {
			return false, fmt.Errorf("error on trap: %s", err)
		}

	case KeywordList:
		list, _ := tokens.Get()
		switch strings.ToUpper(list) {
		case "BREAKS":
			dbg.breakpoints.list()
		case "TRAPS":
			dbg.traps.list()
		}

	case KeywordClear:
		clear, _ := tokens.Get()
		switch strings.ToUpper(clear) {
		case "BREAKS":
			dbg.breakpoints.clear()
			dbg.print(ui.Feedback, "breakpoints cleared")
		case "TRAPS":
			dbg.traps.clear()
			dbg.print(ui.Feedback, "traps cleared")
		}

	case KeywordOnHalt:
		if tokens.Remaining() == 0 {
			dbg.commandOnHalt = dbg.commandOnHaltStored
		} else {
			option, _ := tokens.Peek()
			if strings.ToUpper(option) == "OFF" {
				dbg.commandOnHalt = ""
				dbg.print(ui.Feedback, "no auto-command on halt")
				return false, nil
			}

			// use remaininder of command line to form the ONHALT command sequence
			dbg.commandOnHalt = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnHalt = strings.Replace(dbg.commandOnHalt, ",", ";", -1)

			// store the new command so we can reuse it
			dbg.commandOnHaltStored = dbg.commandOnHalt
		}

		dbg.print(ui.Feedback, "auto-command on halt: %s", dbg.commandOnHalt)

		// run the new onhalt command(s)
		_, err := dbg.parseInput(dbg.commandOnHalt)
		return false, err

	case KeywordOnStep:
		if tokens.Remaining() == 0 {
			dbg.commandOnStep = dbg.commandOnStepStored
		} else {
			option, _ := tokens.Peek()
			if strings.ToUpper(option) == "OFF" {
				dbg.commandOnStep = ""
				dbg.print(ui.Feedback, "no auto-command on step")
				return false, nil
			}

			// use remaininder of command line to form the ONSTEP command sequence
			dbg.commandOnStep = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnStep = strings.Replace(dbg.commandOnStep, ",", ";", -1)

			// store the new command so we can reuse it
			dbg.commandOnStepStored = dbg.commandOnStep
		}

		dbg.print(ui.Feedback, "auto-command on step: %s", dbg.commandOnStep)

		// run the new onstep command(s)
		_, err := dbg.parseInput(dbg.commandOnStep)
		return false, err

	case KeywordLast:
		if dbg.lastResult != nil {
			var printTag ui.PrintProfile
			if dbg.lastResult.Final {
				printTag = ui.CPUStep
			} else {
				printTag = ui.VideoStep
			}
			dbg.print(printTag, "%s", dbg.lastResult.GetString(dbg.disasm.Symtable, result.StyleFull))
		}

	case KeywordMemMap:
		dbg.print(ui.MachineInfo, "%v", dbg.vcs.Mem.MemoryMap())

	case KeywordQuit:
		// *CRITICAL SECTION*
		dbg.runLock.Lock()
		dbg.running = false
		dbg.runLock.Unlock()

	case KeywordReset:
		err := dbg.vcs.Reset()
		if err != nil {
			return false, err
		}
		dbg.print(ui.Feedback, "machine reset")

	case KeywordRun:
		// *CRITICAL SECTION*
		dbg.runLock.Lock()
		dbg.runUntilHalt = true
		dbg.runLock.Unlock()

		stepNext = true

	case KeywordStep:
		stepNext = true

	case KeywordStepMode:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "CPU":
			dbg.inputloopVideoClock = false
		case "VIDEO":
			dbg.inputloopVideoClock = true
		default:
			return false, fmt.Errorf("unknown step mode (%s)", mode)
		}
		dbg.print(ui.Feedback, "step mode: %s", mode)

	case KeywordTerse:
		dbg.machineInfoVerbose = false
		dbg.print(ui.Feedback, "verbosity: terse")

	case KeywordVerbose:
		dbg.machineInfoVerbose = true
		dbg.print(ui.Feedback, "verbosity: verbose")

	case KeywordVerbosity:
		if dbg.machineInfoVerbose {
			dbg.print(ui.Feedback, "verbosity: verbose")
		} else {
			dbg.print(ui.Feedback, "verbosity: terse")
		}

	case KeywordDebuggerState:
		_, err := dbg.parseInput("VERBOSITY; STEPMODE; ONHALT")
		if err != nil {
			return false, err
		}

	// information about the machine (chips)

	case KeywordCPU:
		dbg.printMachineInfo(dbg.vcs.MC)

	case KeywordPeek:
		a, present := tokens.Get()
		for present {
			var addr interface{}
			var msg string

			addr, err := strconv.ParseUint(a, 0, 16)
			if err != nil {
				// argument is not a number so argument must be a string
				addr = strings.ToUpper(a)
				msg = addr.(string)
			} else {
				// convert number to type suitable for Peek command
				addr = uint16(addr.(uint64))
				msg = fmt.Sprintf("%#04x", addr)
			}

			// peform peek
			val, mappedAddress, areaName, addressLabel, err := dbg.vcs.Mem.Peek(addr)
			if err != nil {
				dbg.print(ui.Error, "%s", err)
				continue
			}

			// format results
			if uint64(mappedAddress) != addr {
				msg = fmt.Sprintf("%s = %#04x", msg, mappedAddress)
			}
			msg = fmt.Sprintf("%s -> 0x%02x :: %s", msg, val, areaName)
			if addressLabel != "" {
				msg = fmt.Sprintf("%s [%s]", msg, addressLabel)
			}
			dbg.print(ui.MachineInfo, msg)

			a, present = tokens.Get()
		}

	case KeywordRAM:
		dbg.printMachineInfo(dbg.vcs.Mem.PIA)

	case KeywordRIOT:
		dbg.printMachineInfo(dbg.vcs.RIOT)

	case KeywordTIA:
		dbg.printMachineInfo(dbg.vcs.TIA)

	case KeywordTV:
		dbg.printMachineInfo(dbg.vcs.TV)

	// information about the machine (sprites, playfield)
	case KeywordPlayer:
		// TODO: argument to print either player 0 or player 1
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Player0)
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Player1)

	case KeywordMissile:
		// TODO: argument to print either missile 0 or missile 1
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile0)
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile1)

	case KeywordBall:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Ball)

	case KeywordPlayfield:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Playfield)

	// tv control

	case KeywordDisplay:
		visibility := true
		action, present := tokens.Get()
		if present {
			switch strings.ToUpper(action) {
			case "OFF":
				visibility = false
			}
		}
		err := dbg.vcs.TV.SetVisibility(visibility)
		if err != nil {
			return false, err
		}
	}

	return stepNext, nil
}
