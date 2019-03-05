package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"gopher2600/television"
	"os"
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
	KeywordWatch         = "WATCH"
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
	KeywordCartridge     = "CARTRIDGE"
	KeywordCPU           = "CPU"
	KeywordPeek          = "PEEK"
	KeywordPoke          = "POKE"
	KeywordHexLoad       = "HEXLOAD"
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
	KeywordGrep          = "GREP"
)

// Help contains the help text for the debugger's top level commands
var Help = map[string]string{
	KeywordHelp:          "Lists commands and provides help for individual debugger commands",
	KeywordInsert:        "Insert cartridge into emulation (from file)",
	KeywordSymbol:        "Search for the address label symbol in disassembly. returns address",
	KeywordBreak:         "Cause emulator to halt when conditions are met",
	KeywordTrap:          "Cause emulator to halt when specified machine component is touched",
	KeywordWatch:         "Watch a memory address for activity",
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
	KeywordCartridge:     "Display information about the current cartridge",
	KeywordCPU:           "Display the current state of the CPU",
	KeywordPeek:          "Inspect an individual memory address",
	KeywordPoke:          "Modify an individual memory address",
	KeywordHexLoad:       "Modify a sequence of memory addresses. Starting address must be numeric.",
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
	KeywordGrep:          "Simple string search (case insensitive) of the disassembly",
}

var commandTemplate = input.CommandTemplate{
	KeywordInsert: "%F",
	KeywordSymbol: "%S [|ALL]",

	// break/trap/watch values are parsed in parseTargets() function
	// TODO: find some way to create valid templates using information from
	// other sources
	KeywordBreak: "%*",
	KeywordTrap:  "%*",

	KeywordWatch:         "[READ|WRITE|] %V %*",
	KeywordList:          "[BREAKS|TRAPS|WATCHES]",
	KeywordClear:         "[BREAKS|TRAPS|WATCHES]",
	KeywordDrop:          "[BREAK|TRAP|WATCH] %V",
	KeywordOnHalt:        "[|OFF|ECHO] %*",
	KeywordOnStep:        "[|OFF|ECHO] %*",
	KeywordLast:          "[|DEFN]",
	KeywordMemMap:        "",
	KeywordQuit:          "",
	KeywordReset:         "",
	KeywordRun:           "",
	KeywordStep:          "[|CPU|VIDEO|SCANLINE]", // see notes
	KeywordStepMode:      "[|CPU|VIDEO]",
	KeywordTerse:         "",
	KeywordVerbose:       "",
	KeywordVerbosity:     "",
	KeywordDebuggerState: "",
	KeywordCartridge:     "",
	KeywordCPU:           "",
	KeywordPeek:          "%*",
	KeywordPoke:          "%*",
	KeywordHexLoad:       "%*",
	KeywordRAM:           "",
	KeywordRIOT:          "",
	KeywordTIA:           "[|FUTURE|HMOVE]",
	KeywordTV:            "[|SPEC]",
	KeywordPlayer:        "",
	KeywordMissile:       "",
	KeywordBall:          "",
	KeywordPlayfield:     "",
	KeywordDisplay:       "[|OFF|DEBUG|SCALE|DEBUGCOLORS] %*", // see notes
	KeywordMouse:         "[|X|Y]",
	KeywordScript:        "%F",
	KeywordDisassemble:   "",
	KeywordGrep:          "%S %*",
}

// notes
// o KeywordStep can take a valid target
// o KeywordDisplay SCALE takes an additional argument but OFF and DEBUG do
// 	not. the %* is a compromise

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
		dbg.disasm.Dump(os.Stdout)

	case KeywordGrep:
		search := tokens.Remainder()
		output := strings.Builder{}
		dbg.disasm.Grep(search, &output, false)
		if output.Len() == 0 {
			dbg.print(ui.Error, "%s not found in disassembly", search)
		} else {
			dbg.print(ui.Feedback, output.String())
		}

	case KeywordSymbol:
		// TODO: change this so that it uses debugger.memory front-end
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

	case KeywordWatch:
		err := dbg.watches.parseWatch(tokens, dbg.dbgmem)
		if err != nil {
			return false, fmt.Errorf("error on watch: %s", err)
		}

	case KeywordList:
		list, _ := tokens.Get()
		list = strings.ToUpper(list)
		switch list {
		case "BREAKS":
			dbg.breakpoints.list()
		case "TRAPS":
			dbg.traps.list()
		case "WATCHES":
			dbg.watches.list()
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
		case "WATCHES":
			dbg.watches.clear()
			dbg.print(ui.Feedback, "watches cleared")
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
		case "WATCH":
			err := dbg.watches.drop(num)
			if err != nil {
				return false, err
			}
			dbg.print(ui.Feedback, "watch #%d dropped", num)
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
		err = dbg.vcs.TV.Reset()
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
			// try to parse trap
			tokens.Unget()
			err := dbg.stepTraps.parseTrap(tokens)
			if err != nil {
				return false, fmt.Errorf("unknown step mode (%s)", mode)
			}
			dbg.runUntilHalt = true
			stepNext = true
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
	case KeywordCartridge:
		dbg.printMachineInfo(dbg.vcs.Mem.Cart)

	case KeywordCPU:
		dbg.printMachineInfo(dbg.vcs.MC)

	case KeywordPeek:
		// get first address token
		a, present := tokens.Get()
		if !present {
			dbg.print(ui.Error, "peek address required")
			return false, nil
		}

		for present {
			// perform peek
			val, mappedAddress, areaName, addressLabel, err := dbg.dbgmem.peek(a)
			if err != nil {
				dbg.print(ui.Error, "%s", err)
			} else {
				// format results
				msg := fmt.Sprintf("%#04x -> %#02x :: %s", mappedAddress, val, areaName)
				if addressLabel != "" {
					msg = fmt.Sprintf("%s [%s]", msg, addressLabel)
				}
				dbg.print(ui.MachineInfo, msg)
			}

			// loop through all addresses
			a, present = tokens.Get()
		}

	case KeywordPoke:
		// get address token
		a, present := tokens.Get()
		if !present {
			dbg.print(ui.Error, "poke address required")
			return false, nil
		}

		addr, err := dbg.dbgmem.mapAddress(a, true)
		if err != nil {
			dbg.print(ui.Error, "invalid poke address (%v)", a)
			return false, nil
		}

		// get value token
		a, present = tokens.Get()
		if !present {
			dbg.print(ui.Error, "poke value required")
			return false, nil
		}

		val, err := strconv.ParseUint(a, 0, 8)
		if err != nil {
			dbg.print(ui.Error, "poke value must be numeric (%s)", a)
			return false, nil
		}

		// perform single poke
		err = dbg.dbgmem.poke(addr, uint8(val))
		if err != nil {
			dbg.print(ui.Error, "%s", err)
		} else {
			dbg.print(ui.MachineInfo, fmt.Sprintf("%#04x -> %#02x", addr, uint16(val)))
		}

	case KeywordHexLoad:
		// get address token
		a, present := tokens.Get()
		if !present {
			dbg.print(ui.Error, "hexload address required")
			return false, nil
		}

		addr, err := dbg.dbgmem.mapAddress(a, true)
		if err != nil {
			dbg.print(ui.Error, "invalid hexload address (%s)", a)
			return false, nil
		}

		// get (first) value token
		a, present = tokens.Get()
		if !present {
			dbg.print(ui.Error, "at least one hexload value required")
			return false, nil
		}

		for present {
			val, err := strconv.ParseUint(a, 0, 8)
			if err != nil {
				dbg.print(ui.Error, "hexload value must be numeric (%s)", a)
				a, present = tokens.Get()
				continue // for loop
			}

			// perform individual poke
			err = dbg.dbgmem.poke(uint16(addr), uint8(val))
			if err != nil {
				dbg.print(ui.Error, "%s", err)
			} else {
				dbg.print(ui.MachineInfo, fmt.Sprintf("%#04x -> %#02x", addr, uint16(val)))
			}

			// loop through all values
			a, present = tokens.Get()
			addr++
		}

	case KeywordRAM:
		dbg.printMachineInfo(dbg.vcs.Mem.PIA)

	case KeywordRIOT:
		dbg.printMachineInfo(dbg.vcs.RIOT)

	case KeywordTIA:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "FUTURE":
				dbg.printMachineInfo(dbg.vcs.TIA.Video.OnFutureColorClock)
			case "HMOVE":
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Hmove.MachineInfoInternal())
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Video.Player0.MachineInfoInternal())
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Video.Player1.MachineInfoInternal())
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Video.Missile0.MachineInfoInternal())
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Video.Missile1.MachineInfoInternal())
				dbg.print(ui.MachineInfoInternal, dbg.vcs.TIA.Video.Ball.MachineInfoInternal())
			default:
				return false, fmt.Errorf("unknown request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA)
		}

	case KeywordTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				info, err := dbg.vcs.TV.GetMetaState(television.ReqTVSpec)
				if err != nil {
					return false, err
				}
				dbg.print(ui.MachineInfo, info)
			default:
				return false, fmt.Errorf("unknown request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.vcs.TV)
		}

	// information about the machine (sprites, playfield)
	case KeywordPlayer:
		// TODO: argument to print either player 0 or player 1

		if dbg.machineInfoVerbose {
			// arrange the two player's information side by side in order to
			// save space and to allow for easy comparison

			p0 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Player0), "\n")
			p1 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Player1), "\n")

			ml := 0
			for i := range p0 {
				if len(p0[i]) > ml {
					ml = len(p0[i])
				}
			}

			s := strings.Builder{}
			for i := range p0 {
				if p0[i] != "" {
					s.WriteString(fmt.Sprintf("%s %s | %s\n", p0[i], strings.Repeat(" ", ml-len(p0[i])), p1[i]))
				}
			}
			dbg.print(ui.MachineInfo, s.String())
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Player0)
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Player1)
		}

	case KeywordMissile:
		// TODO: argument to print either missile 0 or missile 1

		if dbg.machineInfoVerbose {
			// arrange the two missile's information side by side in order to
			// save space and to allow for easy comparison

			p0 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile0), "\n")
			p1 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile1), "\n")

			ml := 0
			for i := range p0 {
				if len(p0[i]) > ml {
					ml = len(p0[i])
				}
			}

			s := strings.Builder{}
			for i := range p0 {
				if p0[i] != "" {
					s.WriteString(fmt.Sprintf("%s %s | %s\n", p0[i], strings.Repeat(" ", ml-len(p0[i])), p1[i]))
				}
			}
			dbg.print(ui.MachineInfo, s.String())
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile0)
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile1)
		}

	case KeywordBall:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Ball)

	case KeywordPlayfield:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Playfield)

	// tv control

	case KeywordDisplay:
		var err error

		action, present := tokens.Get()
		if present {
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				err = dbg.vcs.TV.SetFeature(television.ReqSetVisibility, false)
				if err != nil {
					return false, err
				}
			case "DEBUG":
				err = dbg.vcs.TV.SetFeature(television.ReqToggleDebug)
				if err != nil {
					return false, err
				}
			case "SCALE":
				scl, present := tokens.Get()
				if !present {
					return false, fmt.Errorf("value required for %s %s", command, action)
				}

				scale, err := strconv.ParseFloat(scl, 32)
				if err != nil {
					return false, fmt.Errorf("%s %s value not valid (%s)", command, action, scl)
				}

				err = dbg.vcs.TV.SetFeature(television.ReqSetScale, float32(scale))
				return false, err
			case "DEBUGCOLORS":
				dbg.vcs.TIA.UseDebugColors = !dbg.vcs.TIA.UseDebugColors
				if dbg.vcs.TIA.UseDebugColors {
					dbg.print(ui.Feedback, "using debug colors in display")
				} else {
					dbg.print(ui.Feedback, "using program colors in display")
				}
			default:
				return false, fmt.Errorf("unknown display action (%s)", action)
			}
		} else {
			err = dbg.vcs.TV.SetFeature(television.ReqSetVisibility, true)
			if err != nil {
				return false, err
			}
		}

	case KeywordMouse:
		req := television.ReqLastMouse

		coord, present := tokens.Get()

		if present {
			coord = strings.ToUpper(coord)
			switch coord {
			case "X":
				req = television.ReqLastMouseHorizPos
			case "Y":
				req = television.ReqLastMouseScanline
			default:
				return false, fmt.Errorf("unknown mouse option (%s)", coord)
			}
		}

		info, err := dbg.vcs.TV.GetMetaState(req)
		if err != nil {
			return false, err
		}
		dbg.print(ui.MachineInfo, info)
	}

	return stepNext, nil
}
