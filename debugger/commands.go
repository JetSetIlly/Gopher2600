package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"gopher2600/television"
	"strconv"
	"strings"
)

// debugger keywords. not a useful data structure but we can use these to form
// the more useful DebuggerCommands and Help structures
const (
	KeywordHelp          = "HELP"
	KeywordInsert        = "INSERT"
	KeywordSymbol        = "SYMBOL"
	KeywordBreak         = "BREAK"
	KeywordTrap          = "TRAP"
	KeywordList          = "LIST"
	KeywordClear         = "CLEAR"
	KeywordDrop          = "DROP"
	KeywordOnHalt        = "ONHALT"
	KeywordOnStep        = "ONSTEP"
	KeywordLast          = "LAST"
	KeywordMemMap        = "MEMMAP"
	KeywordQuit          = "QUIT"
	KeywordReset         = "RESET"
	KeywordRun           = "RUN"
	KeywordStep          = "STEP"
	KeywordStepMode      = "STEPMODE"
	KeywordTerse         = "TERSE"
	KeywordVerbose       = "VERBOSE"
	KeywordVerbosity     = "VERBOSITY"
	KeywordDebuggerState = "DEBUGGERSTATE"
	KeywordCPU           = "CPU"
	KeywordPeek          = "PEEK"
	KeywordRAM           = "RAM"
	KeywordRIOT          = "RIOT"
	KeywordTIA           = "TIA"
	KeywordTV            = "TV"
	KeywordPlayer        = "PLAYER"
	KeywordMissile       = "MISSILE"
	KeywordBall          = "BALL"
	KeywordPlayfield     = "PLAYFIELD"
	KeywordDisplay       = "DISPLAY"
	KeywordMouse         = "MOUSE"
	KeywordScript        = "SCRIPT"
	KeywordDisassemble   = "DISASSEMBLE"
)

// Help contains the help text for the debugger's top level commands
var Help = map[string]string{
	KeywordHelp:          "Lists commands and provides help for individual debugger commands",
	KeywordInsert:        "Insert cartridge into emulation (from file)",
	KeywordSymbol:        "Search for the address label symbol in disassembly. returns address",
	KeywordBreak:         "Cause emulator to halt when conditions are met",
	KeywordTrap:          "Cause emulator to halt when specified machine component is touched",
	KeywordList:          "List current entries for BREAKS and TRAPS",
	KeywordClear:         "Clear all entries in BREAKS and TRAPS",
	KeywordDrop:          "Drop a specific BREAK or TRAP conditin, using the number of the condition reported by LIST",
	KeywordOnHalt:        "Commands to run whenever emulation is halted (separate commands with comma)",
	KeywordOnStep:        "Commands to run whenever emulation steps forward an cpu/video cycle (separate commands with comma)",
	KeywordLast:          "Prints the result of the last cpu/video cycle",
	KeywordMemMap:        "Display high-levl VCS memory map",
	KeywordQuit:          "Exits the emulator",
	KeywordReset:         "Reset the emulation to its initial state",
	KeywordRun:           "Run emulator until next halt state",
	KeywordStep:          "Step forward emulator one step (see STEPMODE command)",
	KeywordStepMode:      "Change method of stepping: CPU or VIDEO",
	KeywordTerse:         "Use terse format when displaying machine information",
	KeywordVerbose:       "Use verbose format when displaying machine information",
	KeywordVerbosity:     "Display which format is used when displaying machine information (see TERSE and VERBOSE commands)",
	KeywordDebuggerState: "Display summary of debugger options",
	KeywordCPU:           "Display the current state of the CPU",
	KeywordPeek:          "Inspect an individual memory address",
	KeywordRAM:           "Display the current contents of PIA RAM",
	KeywordRIOT:          "Display the current state of the RIOT",
	KeywordTIA:           "Display current state of the TIA",
	KeywordTV:            "Display the current TV state",
	KeywordPlayer:        "Display the current state of the Player 0/1 sprite",
	KeywordMissile:       "Display the current state of the Missile 0/1 sprite",
	KeywordBall:          "Display the current state of the Ball sprite",
	KeywordPlayfield:     "Display the current playfield data",
	KeywordDisplay:       "Display the TV image",
	KeywordMouse:         "Return the coordinates of the last mouse press",
	KeywordScript:        "Run commands from specified file",
	KeywordDisassemble:   "Print the full cartridge disassembly",
}

var commandTemplate = input.CommandTemplate{
	KeywordInsert:        "%F",
	KeywordSymbol:        "%V [|ALL]",
	KeywordBreak:         "%*",
	KeywordTrap:          "%*",
	KeywordList:          "[BREAKS|TRAPS]",
	KeywordClear:         "[BREAKS|TRAPS]",
	KeywordDrop:          "[BREAK|TRAP] %V",
	KeywordOnHalt:        "[|OFF|ECHO] %*",
	KeywordOnStep:        "[|OFF|ECHO] %*",
	KeywordLast:          "[|DEFN]",
	KeywordMemMap:        "",
	KeywordQuit:          "",
	KeywordReset:         "",
	KeywordRun:           "",
	KeywordStep:          "[|CPU|VIDEO]",
	KeywordStepMode:      "[|CPU|VIDEO]",
	KeywordTerse:         "",
	KeywordVerbose:       "",
	KeywordVerbosity:     "",
	KeywordDebuggerState: "",
	KeywordCPU:           "",
	KeywordPeek:          "%*",
	KeywordRAM:           "",
	KeywordRIOT:          "",
	KeywordTIA:           "",
	KeywordTV:            "[|SPEC]",
	KeywordPlayer:        "",
	KeywordMissile:       "",
	KeywordBall:          "",
	KeywordPlayfield:     "",
	KeywordDisplay:       "[|OFF]",
	KeywordMouse:         "[|X|Y]",
	KeywordScript:        "%F",
	KeywordDisassemble:   "",
}

// DebuggerCommands is the tree of valid commands
var DebuggerCommands input.Commands

func init() {
	var err error

	// parse command template
	DebuggerCommands, err = input.CompileCommandTemplate(commandTemplate, KeywordHelp)
	if err != nil {
		panic(fmt.Errorf("error compiling command template: %s", err))
	}
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
		table, symbol, address, err := dbg.disasm.Symtable.SearchSymbol(symbol, symbols.UnspecifiedSymTable)
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

		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "ALL":
				dbg.print(ui.Feedback, "%s -> %#04x", symbol, address)

				// find all instances of symbol address in memory space
				// assumption: the address returned by SearchSymbol is the
				// first address in the complete list
				for m := address + 1; m < dbg.vcs.Mem.Cart.Origin(); m++ {
					if dbg.vcs.Mem.MapAddress(m, table == symbols.ReadSymTable) == address {
						dbg.print(ui.Feedback, "%s -> %#04x", symbol, m)
					}
				}
			default:
				return false, fmt.Errorf("unknown option for SYMBOL command (%s)", option)
			}
		} else {
			dbg.print(ui.Feedback, "%s -> %#04x", symbol, address)
		}

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
		list = strings.ToUpper(list)
		switch list {
		case "BREAKS":
			dbg.breakpoints.list()
		case "TRAPS":
			dbg.traps.list()
		default:
			return false, fmt.Errorf("unknown list option (%s)", list)
		}

	case KeywordClear:
		clear, _ := tokens.Get()
		clear = strings.ToUpper(clear)
		switch clear {
		case "BREAKS":
			dbg.breakpoints.clear()
			dbg.print(ui.Feedback, "breakpoints cleared")
		case "TRAPS":
			dbg.traps.clear()
			dbg.print(ui.Feedback, "traps cleared")
		default:
			return false, fmt.Errorf("unknown clear option (%s)", clear)
		}

	case KeywordDrop:
		drop, _ := tokens.Get()

		s, _ := tokens.Get()
		num, err := strconv.Atoi(s)
		if err != nil {
			return false, fmt.Errorf("drop attribute must be a decimal number (%s)", s)
		}

		drop = strings.ToUpper(drop)
		switch drop {
		case "BREAK":
			err := dbg.breakpoints.drop(num)
			if err != nil {
				return false, err
			}
			dbg.print(ui.Feedback, "breakpoint #%d dropped", num)
		case "TRAP":
			err := dbg.traps.drop(num)
			if err != nil {
				return false, err
			}
			dbg.print(ui.Feedback, "trap #%d dropped", num)
		default:
			return false, fmt.Errorf("unknown drop option (%s)", drop)
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
			if strings.ToUpper(option) == "ECHO" {
				dbg.print(ui.Feedback, "auto-command on halt: %s", dbg.commandOnHalt)
				return false, nil
			}

			// use remaininder of command line to form the ONHALT command sequence
			dbg.commandOnHalt = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnHalt = strings.Replace(dbg.commandOnHalt, ",", ";", -1)

			// store the new command so we can reuse it
			// TODO: normalise case of specified command sequence
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
			if strings.ToUpper(option) == "ECHO" {
				dbg.print(ui.Feedback, "auto-command on step: %s", dbg.commandOnStep)
				return false, nil
			}

			// use remaininder of command line to form the ONSTEP command sequence
			dbg.commandOnStep = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnStep = strings.Replace(dbg.commandOnStep, ",", ";", -1)

			// store the new command so we can reuse it
			// TODO: normalise case of specified command sequence
			dbg.commandOnStepStored = dbg.commandOnStep
		}

		dbg.print(ui.Feedback, "auto-command on step: %s", dbg.commandOnStep)

		// run the new onstep command(s)
		_, err := dbg.parseInput(dbg.commandOnStep)
		return false, err

	case KeywordLast:
		if dbg.lastResult != nil {
			option, _ := tokens.Get()
			option = strings.ToUpper(option)
			switch option {
			case "DEFN":
				dbg.print(ui.Feedback, "%s", dbg.lastResult.Defn)
			case "":
				var printTag ui.PrintProfile
				if dbg.lastResult.Final {
					printTag = ui.CPUStep
				} else {
					printTag = ui.VideoStep
				}
				dbg.print(printTag, "%s", dbg.lastResult.GetString(dbg.disasm.Symtable, result.StyleFull))
			default:
				return false, fmt.Errorf("unknown last request option (%s)", option)
			}
		}

	case KeywordMemMap:
		dbg.print(ui.MachineInfo, "%v", dbg.vcs.Mem.MemoryMap())

	case KeywordQuit:
		dbg.running = false

	case KeywordReset:
		err := dbg.vcs.Reset()
		if err != nil {
			return false, err
		}
		dbg.print(ui.Feedback, "machine reset")

	case KeywordRun:
		dbg.runUntilHalt = true
		stepNext = true

	case KeywordStep:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "":
			stepNext = true
		case "CPU":
			dbg.inputloopVideoClock = false
			stepNext = true
		case "VIDEO":
			dbg.inputloopVideoClock = true
			stepNext = true
		default:
			return false, fmt.Errorf("unknown step mode (%s)", mode)
		}

	case KeywordStepMode:
		mode, present := tokens.Get()
		if present {
			mode = strings.ToUpper(mode)
			switch mode {
			case "CPU":
				dbg.inputloopVideoClock = false
			case "VIDEO":
				dbg.inputloopVideoClock = true
			default:
				return false, fmt.Errorf("unknown step mode (%s)", mode)
			}
		}
		if dbg.inputloopVideoClock {
			mode = "VIDEO"
		} else {
			mode = "CPU"
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
		_, err := dbg.parseInput("VERBOSITY; STEPMODE; ONHALT ECHO; ONSTEP ECHO")
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
			} else {
				// format results
				if uint64(mappedAddress) != addr {
					msg = fmt.Sprintf("%s = %#04x", msg, mappedAddress)
				}
				msg = fmt.Sprintf("%s -> 0x%02x :: %s", msg, val, areaName)
				if addressLabel != "" {
					msg = fmt.Sprintf("%s [%s]", msg, addressLabel)
				}
				dbg.print(ui.MachineInfo, msg)
			}

			a, present = tokens.Get()
		}

	case KeywordRAM:
		dbg.printMachineInfo(dbg.vcs.Mem.PIA)

	case KeywordRIOT:
		dbg.printMachineInfo(dbg.vcs.RIOT)

	case KeywordTIA:
		dbg.printMachineInfo(dbg.vcs.TIA)

	case KeywordTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				info, err := dbg.vcs.TV.RequestTVInfo(television.ReqTVSpec)
				if err != nil {
					return false, err
				}
				dbg.print(ui.MachineInfo, info)
			default:
				return false, fmt.Errorf("unknown info request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.vcs.TV)
		}

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
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				visibility = false
			default:
				return false, fmt.Errorf("unknown display action (%s)", action)
			}
		}
		err := dbg.vcs.TV.SetVisibility(visibility)
		if err != nil {
			return false, err
		}

	case KeywordMouse:
		req := television.ReqLastMouse

		coord, present := tokens.Get()

		if present {
			coord = strings.ToUpper(coord)
			switch coord {
			case "X":
				req = television.ReqLastMouseX
			case "Y":
				req = television.ReqLastMouseY
			default:
				return false, fmt.Errorf("unknown mouse option (%s)", coord)
			}
		}

		info, err := dbg.vcs.TV.RequestTVInfo(req)
		if err != nil {
			return false, err
		}
		dbg.print(ui.MachineInfo, info)
	}

	return stepNext, nil
}
