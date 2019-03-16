package debugger

import (
	"fmt"
	"gopher2600/debugger/console"
	"gopher2600/debugger/input"
	"gopher2600/errors"
	"gopher2600/gui"
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
	cmdBall          = "BALL"
	cmdBreak         = "BREAK"
	cmdCPU           = "CPU"
	cmdCapture       = "CAPTURE"
	cmdCartridge     = "CARTRIDGE"
	cmdClear         = "CLEAR"
	cmdDebuggerState = "DEBUGGERSTATE"
	cmdDisassemble   = "DISASSEMBLE"
	cmdDisplay       = "DISPLAY"
	cmdDrop          = "DROP"
	cmdGrep          = "GREP"
	cmdHelp          = "HELP"
	cmdHexLoad       = "HEXLOAD"
	cmdInsert        = "INSERT"
	cmdLast          = "LAST"
	cmdList          = "LIST"
	cmdMemMap        = "MEMMAP"
	cmdMissile       = "MISSILE"
	cmdMouse         = "MOUSE"
	cmdOnHalt        = "ONHALT"
	cmdOnStep        = "ONSTEP"
	cmdPeek          = "PEEK"
	cmdPlayer        = "PLAYER"
	cmdPlayfield     = "PLAYFIELD"
	cmdPoke          = "POKE"
	cmdQuit          = "QUIT"
	cmdRAM           = "RAM"
	cmdRIOT          = "RIOT"
	cmdReset         = "RESET"
	cmdRun           = "RUN"
	cmdScript        = "SCRIPT"
	cmdStep          = "STEP"
	cmdStepMode      = "STEPMODE"
	cmdStick0        = "STICK0"
	cmdStick1        = "STICK1"
	cmdSymbol        = "SYMBOL"
	cmdTIA           = "TIA"
	cmdTV            = "TV"
	cmdTerse         = "TERSE"
	cmdTrap          = "TRAP"
	cmdVerbose       = "VERBOSE"
	cmdVerbosity     = "VERBOSITY"
	cmdWatch         = "WATCH"
)

// notes
// o KeywordStep can take a valid target
// o KeywordDisplay SCALE takes an additional argument but OFF and DEBUG do
// 	not. the %* is a compromise

// break/trap/watch values are parsed in parseTargets() function
// TODO: find some way to create valid templates using information from
// other sources

var commandTemplate = input.CommandTemplate{
	cmdBall:          "",
	cmdBreak:         "%*",
	cmdCPU:           "",
	cmdCapture:       "[END|%F]",
	cmdCartridge:     "",
	cmdClear:         "[BREAKS|TRAPS|WATCHES]",
	cmdDebuggerState: "",
	cmdDisassemble:   "",
	cmdDisplay:       "[|OFF|DEBUG|SCALE|DEBUGCOLORS] %*", // see notes
	cmdDrop:          "[BREAK|TRAP|WATCH] %V",
	cmdGrep:          "%S %*",
	cmdHexLoad:       "%*",
	cmdInsert:        "%F",
	cmdLast:          "[|DEFN]",
	cmdList:          "[BREAKS|TRAPS|WATCHES]",
	cmdMemMap:        "",
	cmdMissile:       "",
	cmdMouse:         "[|X|Y]",
	cmdOnHalt:        "[|OFF|RESTORE] %*",
	cmdOnStep:        "[|OFF|RESTORE] %*",
	cmdPeek:          "%*",
	cmdPlayer:        "",
	cmdPlayfield:     "",
	cmdPoke:          "%*",
	cmdQuit:          "",
	cmdRAM:           "",
	cmdRIOT:          "",
	cmdReset:         "",
	cmdRun:           "",
	cmdScript:        "%F",
	cmdStep:          "[|CPU|VIDEO|SCANLINE]", // see notes
	cmdStepMode:      "[|CPU|VIDEO]",
	cmdStick0:        "[LEFT|RIGHT|UP|DOWN|FIRE|CENTRE|NOFIRE]",
	cmdStick1:        "[LEFT|RIGHT|UP|DOWN|FIRE|CENTRE|NOFIRE]",
	cmdSymbol:        "%S [|ALL]",
	cmdTIA:           "[|FUTURE|HMOVE]",
	cmdTV:            "[|SPEC]",
	cmdTerse:         "",
	cmdTrap:          "%*",
	cmdVerbose:       "",
	cmdVerbosity:     "",
	cmdWatch:         "[READ|WRITE|] %V %*",
}

// DebuggerCommands is the tree of valid commands
var DebuggerCommands input.Commands

func init() {
	var err error

	// parse command template
	DebuggerCommands, err = input.CompileCommandTemplate(commandTemplate, cmdHelp)
	if err != nil {
		panic(fmt.Errorf("error compiling command template: %s", err))
	}
}

type parseCommandResult int

const (
	doNothing parseCommandResult = iota
	emptyInput
	stepContinue
	setDefaultStep
	captureStarted
	captureEnded
)

// parseCommand scans user input for valid commands and acts upon it. commands
// that cause the emulation to move forward (RUN, STEP) return true for the
// first return value. other commands return false and act upon the command
// immediately. note that the empty string is the same as the STEP command
//
// TODO: categorise commands into script-safe and non-script-safe
func (dbg *Debugger) parseCommand(userInput *string) (parseCommandResult, error) {
	// tokenise input
	tokens := input.TokeniseInput(*userInput)

	// check validity of input
	err := DebuggerCommands.ValidateInput(tokens)
	if err != nil {
		return doNothing, err
	}

	// if there are no tokens in the input then return emptyInput directive
	if tokens.Remaining() == 0 {
		return emptyInput, nil
	}

	// normalise user input
	*userInput = tokens.String()

	tokens.Reset()
	command, _ := tokens.Get()
	command = strings.ToUpper(command)
	switch command {
	default:
		return doNothing, fmt.Errorf("%s is not yet implemented", command)

	case cmdHelp:
		keyword, present := tokens.Get()
		if present {
			s := strings.ToUpper(keyword)
			txt, prs := Help[s]
			if prs == false {
				dbg.print(console.Help, "no help for %s", s)
			} else {
				dbg.print(console.Help, txt)
			}
		} else {
			for k := range DebuggerCommands {
				dbg.print(console.Help, k)
			}
		}

	case cmdInsert:
		cart, _ := tokens.Get()
		err := dbg.loadCartridge(cart)
		if err != nil {
			return doNothing, err
		}
		dbg.print(console.Feedback, "machine reset with new cartridge (%s)", cart)

	case cmdScript:
		script, _ := tokens.Get()

		spt, err := dbg.loadScript(script)
		if err != nil {
			dbg.print(console.Error, "error running debugger initialisation script: %s\n", err)
			return doNothing, err
		}

		err = dbg.inputLoop(spt, false)
		if err != nil {
			return doNothing, err
		}

	case cmdDisassemble:
		dbg.disasm.Dump(os.Stdout)

	case cmdGrep:
		search := tokens.Remainder()
		output := strings.Builder{}
		dbg.disasm.Grep(search, &output, false, 3)
		if output.Len() == 0 {
			dbg.print(console.Error, "%s not found in disassembly", search)
		} else {
			dbg.print(console.Feedback, output.String())
		}

	case cmdSymbol:
		// TODO: change this so that it uses debugger.memory front-end
		symbol, _ := tokens.Get()
		table, symbol, address, err := dbg.disasm.Symtable.SearchSymbol(symbol, symbols.UnspecifiedSymTable)
		if err != nil {
			switch err := err.(type) {
			case errors.FormattedError:
				if err.Errno == errors.SymbolUnknown {
					dbg.print(console.Feedback, "%s -> not found", symbol)
					return doNothing, nil
				}
			}
			return doNothing, err
		}

		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "ALL":
				dbg.print(console.Feedback, "%s -> %#04x", symbol, address)

				// find all instances of symbol address in memory space
				// assumption: the address returned by SearchSymbol is the
				// first address in the complete list
				for m := address + 1; m < dbg.vcs.Mem.Cart.Origin(); m++ {
					if dbg.vcs.Mem.MapAddress(m, table == symbols.ReadSymTable) == address {
						dbg.print(console.Feedback, "%s -> %#04x", symbol, m)
					}
				}
			default:
				return doNothing, fmt.Errorf("unknown option for SYMBOL command (%s)", option)
			}
		} else {
			dbg.print(console.Feedback, "%s -> %#04x", symbol, address)
		}

	case cmdBreak:
		err := dbg.breakpoints.parseBreakpoint(tokens)
		if err != nil {
			return doNothing, fmt.Errorf("error on break: %s", err)
		}

	case cmdTrap:
		err := dbg.traps.parseTrap(tokens)
		if err != nil {
			return doNothing, fmt.Errorf("error on trap: %s", err)
		}

	case cmdWatch:
		err := dbg.watches.parseWatch(tokens, dbg.dbgmem)
		if err != nil {
			return doNothing, fmt.Errorf("error on watch: %s", err)
		}

	case cmdList:
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
			return doNothing, fmt.Errorf("unknown list option (%s)", list)
		}

	case cmdClear:
		clear, _ := tokens.Get()
		clear = strings.ToUpper(clear)
		switch clear {
		case "BREAKS":
			dbg.breakpoints.clear()
			dbg.print(console.Feedback, "breakpoints cleared")
		case "TRAPS":
			dbg.traps.clear()
			dbg.print(console.Feedback, "traps cleared")
		case "WATCHES":
			dbg.watches.clear()
			dbg.print(console.Feedback, "watches cleared")
		default:
			return doNothing, fmt.Errorf("unknown clear option (%s)", clear)
		}

	case cmdDrop:
		drop, _ := tokens.Get()

		s, _ := tokens.Get()
		num, err := strconv.Atoi(s)
		if err != nil {
			return doNothing, fmt.Errorf("drop attribute must be a decimal number (%s)", s)
		}

		drop = strings.ToUpper(drop)
		switch drop {
		case "BREAK":
			err := dbg.breakpoints.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.Feedback, "breakpoint #%d dropped", num)
		case "TRAP":
			err := dbg.traps.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.Feedback, "trap #%d dropped", num)
		case "WATCH":
			err := dbg.watches.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.Feedback, "watch #%d dropped", num)
		default:
			return doNothing, fmt.Errorf("unknown drop option (%s)", drop)
		}

	case cmdOnHalt:
		if tokens.Remaining() == 0 {
			dbg.print(console.Feedback, "auto-command on halt: %s", dbg.commandOnHalt)
			return doNothing, nil
		}

		option, _ := tokens.Peek()
		switch strings.ToUpper(option) {
		case "OFF":
			dbg.commandOnHalt = ""
		case "RESTORE":
			dbg.commandOnHalt = dbg.commandOnHaltStored
		default:
			// use remaininder of command line to form the ONHALT command sequence
			dbg.commandOnHalt = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnHalt = strings.Replace(dbg.commandOnHalt, ",", ";", -1)

			// store the new command so we can reuse it
			// TODO: normalise case of specified command sequence
			dbg.commandOnHaltStored = dbg.commandOnHalt
		}

		// display the new/restored onhalt command(s)
		if dbg.commandOnHalt == "" {
			dbg.print(console.Feedback, "auto-command on halt: OFF")
		} else {
			dbg.print(console.Feedback, "auto-command on halt: %s", dbg.commandOnHalt)
		}

		// run the new/restored onhalt command(s)
		_, err := dbg.parseInput(dbg.commandOnHalt, false)
		return doNothing, err

	case cmdOnStep:
		if tokens.Remaining() == 0 {
			dbg.print(console.Feedback, "auto-command on step: %s", dbg.commandOnStep)
			return doNothing, nil
		}

		option, _ := tokens.Peek()
		switch strings.ToUpper(option) {
		case "OFF":
			dbg.commandOnStep = ""
		case "RESTORE":
			dbg.commandOnStep = dbg.commandOnStepStored
		default:
			// use remaininder of command line to form the ONSTEP command sequence
			dbg.commandOnStep = tokens.Remainder()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			dbg.commandOnStep = strings.Replace(dbg.commandOnStep, ",", ";", -1)

			// store the new command so we can reuse it
			// TODO: normalise case of specified command sequence
			dbg.commandOnStepStored = dbg.commandOnStep
		}

		// display the new/restored onstep command(s)
		if dbg.commandOnStep == "" {
			dbg.print(console.Feedback, "auto-command on step: OFF")
		} else {
			dbg.print(console.Feedback, "auto-command on step: %s", dbg.commandOnStep)
		}

		// run the new/restored onstep command(s)
		_, err := dbg.parseInput(dbg.commandOnStep, false)
		return doNothing, err

	case cmdLast:
		if dbg.lastResult != nil {
			option, _ := tokens.Get()
			option = strings.ToUpper(option)
			switch option {
			case "DEFN":
				dbg.print(console.Feedback, "%s", dbg.lastResult.Defn)
			case "":
				var printTag console.PrintProfile
				if dbg.lastResult.Final {
					printTag = console.CPUStep
				} else {
					printTag = console.VideoStep
				}
				dbg.print(printTag, "%s", dbg.lastResult.GetString(dbg.disasm.Symtable, result.StyleFull))
			default:
				return doNothing, fmt.Errorf("unknown last request option (%s)", option)
			}
		}

	case cmdMemMap:
		dbg.print(console.MachineInfo, "%v", dbg.vcs.Mem.MemoryMap())

	case cmdQuit:
		dbg.running = false

	case cmdReset:
		err := dbg.vcs.Reset()
		if err != nil {
			return doNothing, err
		}
		err = dbg.tv.Reset()
		if err != nil {
			return doNothing, err
		}
		dbg.print(console.Feedback, "machine reset")

	case cmdRun:
		dbg.runUntilHalt = true
		return stepContinue, nil

	case cmdStep:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "":
		case "CPU":
			dbg.inputEveryVideoCycle = false
		case "VIDEO":
			dbg.inputEveryVideoCycle = true
		default:
			// try to parse trap
			tokens.Unget()
			err := dbg.stepTraps.parseTrap(tokens)
			if err != nil {
				return doNothing, fmt.Errorf("unknown step mode (%s)", mode)
			}
			dbg.runUntilHalt = true
		}

		return setDefaultStep, nil

	case cmdStepMode:
		mode, present := tokens.Get()
		if present {
			mode = strings.ToUpper(mode)
			switch mode {
			case "CPU":
				dbg.inputEveryVideoCycle = false
			case "VIDEO":
				dbg.inputEveryVideoCycle = true
			default:
				return doNothing, fmt.Errorf("unknown step mode (%s)", mode)
			}
		}
		if dbg.inputEveryVideoCycle {
			mode = "VIDEO"
		} else {
			mode = "CPU"
		}
		dbg.print(console.Feedback, "step mode: %s", mode)

	case cmdTerse:
		dbg.machineInfoVerbose = false
		dbg.print(console.Feedback, "verbosity: terse")

	case cmdVerbose:
		dbg.machineInfoVerbose = true
		dbg.print(console.Feedback, "verbosity: verbose")

	case cmdVerbosity:
		if dbg.machineInfoVerbose {
			dbg.print(console.Feedback, "verbosity: verbose")
		} else {
			dbg.print(console.Feedback, "verbosity: terse")
		}

	case cmdDebuggerState:
		_, err := dbg.parseInput("VERBOSITY; STEPMODE; ONHALT ECHO; ONSTEP ECHO", false)
		if err != nil {
			return doNothing, err
		}

	case cmdCartridge:
		dbg.printMachineInfo(dbg.vcs.Mem.Cart)

	case cmdCPU:
		dbg.printMachineInfo(dbg.vcs.MC)

	case cmdPeek:
		// get first address token
		a, present := tokens.Get()
		if !present {
			dbg.print(console.Error, "peek address required")
			return doNothing, nil
		}

		for present {
			// perform peek
			val, mappedAddress, areaName, addressLabel, err := dbg.dbgmem.peek(a)
			if err != nil {
				dbg.print(console.Error, "%s", err)
			} else {
				// format results
				msg := fmt.Sprintf("%#04x -> %#02x :: %s", mappedAddress, val, areaName)
				if addressLabel != "" {
					msg = fmt.Sprintf("%s [%s]", msg, addressLabel)
				}
				dbg.print(console.MachineInfo, msg)
			}

			// loop through all addresses
			a, present = tokens.Get()
		}

	case cmdPoke:
		// get address token
		a, present := tokens.Get()
		if !present {
			dbg.print(console.Error, "poke address required")
			return doNothing, nil
		}

		addr, err := dbg.dbgmem.mapAddress(a, true)
		if err != nil {
			dbg.print(console.Error, "invalid poke address (%v)", a)
			return doNothing, nil
		}

		// get value token
		a, present = tokens.Get()
		if !present {
			dbg.print(console.Error, "poke value required")
			return doNothing, nil
		}

		val, err := strconv.ParseUint(a, 0, 8)
		if err != nil {
			dbg.print(console.Error, "poke value must be numeric (%s)", a)
			return doNothing, nil
		}

		// perform single poke
		err = dbg.dbgmem.poke(addr, uint8(val))
		if err != nil {
			dbg.print(console.Error, "%s", err)
		} else {
			dbg.print(console.MachineInfo, fmt.Sprintf("%#04x -> %#02x", addr, uint16(val)))
		}

	case cmdHexLoad:
		// get address token
		a, present := tokens.Get()
		if !present {
			dbg.print(console.Error, "hexload address required")
			return doNothing, nil
		}

		addr, err := dbg.dbgmem.mapAddress(a, true)
		if err != nil {
			dbg.print(console.Error, "invalid hexload address (%s)", a)
			return doNothing, nil
		}

		// get (first) value token
		a, present = tokens.Get()
		if !present {
			dbg.print(console.Error, "at least one hexload value required")
			return doNothing, nil
		}

		for present {
			val, err := strconv.ParseUint(a, 0, 8)
			if err != nil {
				dbg.print(console.Error, "hexload value must be numeric (%s)", a)
				a, present = tokens.Get()
				continue // for loop
			}

			// perform individual poke
			err = dbg.dbgmem.poke(uint16(addr), uint8(val))
			if err != nil {
				dbg.print(console.Error, "%s", err)
			} else {
				dbg.print(console.MachineInfo, fmt.Sprintf("%#04x -> %#02x", addr, uint16(val)))
			}

			// loop through all values
			a, present = tokens.Get()
			addr++
		}

	case cmdRAM:
		dbg.printMachineInfo(dbg.vcs.Mem.PIA)

	case cmdRIOT:
		dbg.printMachineInfo(dbg.vcs.RIOT)

	case cmdTIA:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "FUTURE":
				dbg.printMachineInfo(dbg.vcs.TIA.Video.OnFutureColorClock)
			case "HMOVE":
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Hmove.MachineInfoInternal())
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Video.Player0.MachineInfoInternal())
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Video.Player1.MachineInfoInternal())
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Video.Missile0.MachineInfoInternal())
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Video.Missile1.MachineInfoInternal())
				dbg.print(console.MachineInfoInternal, dbg.vcs.TIA.Video.Ball.MachineInfoInternal())
			default:
				return doNothing, fmt.Errorf("unknown request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA)
		}

	case cmdTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				info, err := dbg.tv.GetState(television.ReqTVSpec)
				if err != nil {
					return doNothing, err
				}
				dbg.print(console.MachineInfo, info.(string))
			default:
				return doNothing, fmt.Errorf("unknown request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.tv)
		}

	// information about the machine (sprites, playfield)
	case cmdPlayer:
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
			dbg.print(console.MachineInfo, s.String())
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Player0)
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Player1)
		}

	case cmdMissile:
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
			dbg.print(console.MachineInfo, s.String())
		} else {
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile0)
			dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile1)
		}

	case cmdBall:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Ball)

	case cmdPlayfield:
		dbg.printMachineInfo(dbg.vcs.TIA.Video.Playfield)

	case cmdDisplay:
		var err error

		action, present := tokens.Get()
		if present {
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				err = dbg.tv.SetFeature(gui.ReqSetVisibility, false)
				if err != nil {
					return doNothing, err
				}
			case "DEBUG":
				err = dbg.tv.SetFeature(gui.ReqToggleMasking)
				if err != nil {
					return doNothing, err
				}
			case "SCALE":
				scl, present := tokens.Get()
				if !present {
					return doNothing, fmt.Errorf("value required for %s %s", command, action)
				}

				scale, err := strconv.ParseFloat(scl, 32)
				if err != nil {
					return doNothing, fmt.Errorf("%s %s value not valid (%s)", command, action, scl)
				}

				err = dbg.tv.SetFeature(gui.ReqSetScale, float32(scale))
				return doNothing, err
			case "DEBUGCOLORS":
				err = dbg.tv.SetFeature(gui.ReqToggleAltColors)
				if err != nil {
					return doNothing, err
				}
			case "METASIGNALS":
				err = dbg.tv.SetFeature(gui.ReqToggleShowSystemState)
				if err != nil {
					return doNothing, err
				}
			default:
				return doNothing, fmt.Errorf("unknown display action (%s)", action)
			}
		} else {
			err = dbg.tv.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdMouse:
		req := gui.ReqLastMouse

		coord, present := tokens.Get()

		if present {
			coord = strings.ToUpper(coord)
			switch coord {
			case "X":
				req = gui.ReqLastMouseHorizPos
			case "Y":
				req = gui.ReqLastMouseScanline
			default:
				return doNothing, fmt.Errorf("unknown mouse option (%s)", coord)
			}
		}

		info, err := dbg.tv.GetMetaState(req)
		if err != nil {
			return doNothing, err
		}
		dbg.print(console.MachineInfo, info.(string))

	case cmdStick0:
		action, present := tokens.Get()
		if present {
			err := dbg.vcs.Controller.HandleStick(0, action)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdStick1:
		action, present := tokens.Get()
		if present {
			err := dbg.vcs.Controller.HandleStick(1, action)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdCapture:
		tok, _ := tokens.Get()

		if strings.ToUpper(tok) == "END" {
			if dbg.capture == nil {
				return doNothing, fmt.Errorf("no script capture currently taking place")
			}
			err := dbg.capture.end()
			dbg.capture = nil
			return captureEnded, err
		}

		var err error

		dbg.capture, err = dbg.startCaptureScript(tok)
		return captureStarted, err
	}

	return doNothing, nil
}
