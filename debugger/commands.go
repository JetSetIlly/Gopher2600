package debugger

import (
	"fmt"
	"gopher2600/debugger/commandline"
	"gopher2600/debugger/console"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/result"
	"gopher2600/symbols"
	"gopher2600/television"
	"os"
	"sort"
	"strconv"
	"strings"
)

// TODO: categorise commands into script-safe and non-script-safe

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
	cmdDisassembly   = "DISASSEMBLY"
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
	cmdStick         = "STICK"
	cmdSymbol        = "SYMBOL"
	cmdTIA           = "TIA"
	cmdTV            = "TV"
	cmdTerse         = "TERSE"
	cmdTrap          = "TRAP"
	cmdVerbose       = "VERBOSE"
	cmdVerbosity     = "VERBOSITY"
	cmdWatch         = "WATCH"
)

var expCommandTemplate = []string{
	cmdBall,
	cmdBreak + " [%*]",
	cmdCPU,
	cmdCapture + " [END|%F]",
	cmdCartridge,
	cmdClear + " [BREAKS|TRAPS|WATCHES]",
	cmdDebuggerState,
	cmdDisassembly + "(STATE)",
	cmdDisplay + " (OFF|DEBUG|SCALE [%I]|DEBUGCOLORS)", // see notes
	cmdDrop + " [BREAK|TRAP|WATCH] %V",
	cmdGrep + " %V",
	cmdHelp + " %*",
	cmdHexLoad + " %V %*",
	cmdInsert + " %F",
	cmdLast + " (DEFN)",
	cmdList + " [BREAKS|TRAPS|WATCHES]",
	cmdMemMap,
	cmdMissile + "(0|1)",
	cmdOnHalt + " (OFF|RESTORE|%*)",
	cmdOnStep + " (OFF|RESTORE|%*)",
	cmdPeek + " %V %*",
	cmdPlayer + "(0|1)",
	cmdPlayfield,
	cmdPoke + " %V %*",
	cmdQuit,
	cmdRAM,
	cmdRIOT,
	cmdReset,
	cmdRun,
	cmdScript + " %F",
	cmdStep + " (CPU|VIDEO|SCANLINE)", // see notes
	cmdStepMode + " (CPU|VIDEO)",
	cmdStick + "[0|1] [LEFT|RIGHT|UP|DOWN|FIRE|CENTRE|NOFIRE]",
	cmdSymbol + " %V (ALL)",
	cmdTIA + " (FUTURE|HMOVE)",
	cmdTV + " (SPEC)",
	cmdTerse,
	cmdTrap + " [%*]",
	cmdVerbose,
	cmdVerbosity,
	cmdWatch + " (READ|WRITE) [%V]",
}

var debuggerCommands *commandline.Commands
var debuggerCommandsIdx *commandline.Index

func init() {
	var err error

	// parse command template
	debuggerCommands, err = commandline.ParseCommandTemplate(expCommandTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(100)
	}
	sort.Stable(debuggerCommands)

	debuggerCommandsIdx = commandline.CreateIndex(debuggerCommands)
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

// parseCommand/enactCommand scans user input for a valid command and acts upon
// it. commands that cause the emulation to move forward (RUN, STEP) return
// true for the first return value. other commands return false and act upon
// the command immediately. note that the empty string is the same as the STEP
// command

func (dbg *Debugger) parseCommand(userInput *string) (parseCommandResult, error) {
	// tokenise input
	tokens := commandline.TokeniseInput(*userInput)

	// normalise user input -- we don't use the results in this
	// function but we do use it futher-up. eg. when capturing user input to a
	// script
	*userInput = tokens.String()

	// if there are no tokens in the input then return emptyInput directive
	if tokens.Remaining() == 0 {
		return emptyInput, nil
	}

	// check validity of tokenised input
	//
	// the absolute best thing about this is that we don't need to worrying too
	// much about the success of tokens.Get() in the command implementations
	// below:
	//
	//   tok, _ := tokens.Get()
	//
	// is an acceptable pattern. default values can be handled thus:
	//
	//  tok, ok := tokens.Get()
	//  if ok {
	//    switch tok {
	//		...
	//	  }
	//  } else {
	//	  // default action
	//    ...
	//  }
	//
	err := debuggerCommands.ValidateTokens(tokens)
	if err != nil {
		return doNothing, err
	}

	return dbg.enactCommand(tokens)
}

func (dbg *Debugger) enactCommand(tokens *commandline.Tokens) (parseCommandResult, error) {
	// check first token. if this token makes sense then we will consume the
	// rest of the tokens appropriately
	tokens.Reset()
	command, _ := tokens.Get()

	// take uppercase value of the first token. it's useful to take the
	// uppercase value but we have to be careful when we do it because
	command = strings.ToUpper(command)

	switch command {
	default:
		return doNothing, fmt.Errorf("%s is not yet implemented", command)

	case cmdHelp:
		keyword, present := tokens.Get()
		if present {
			keyword = strings.ToUpper(keyword)
			helpTxt, prs := Help[keyword]
			if prs == false {
				dbg.print(console.Help, "no help for %s", keyword)
			} else {
				helpTxt = fmt.Sprintf("%s\n\n  Usage: %s", helpTxt, (*debuggerCommandsIdx)[keyword].String())
				dbg.print(console.Help, helpTxt)
			}
		} else {
			dbg.print(console.Help, debuggerCommands.String())
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

	case cmdDisassembly:
		option, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(option) {
			case "STATE":
				dbg.print(console.Feedback, dbg.disasm.String())
			}
		} else {
			dbg.disasm.Dump(os.Stdout)
		}

	case cmdGrep:
		search, _ := tokens.Get()
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
			option, ok := tokens.Get()
			if ok {
				switch strings.ToUpper(option) {
				case "DEFN":
					dbg.print(console.Feedback, "%s", dbg.lastResult.Defn)
				}
			} else {
				var printTag console.PrintProfile
				if dbg.lastResult.Final {
					printTag = console.CPUStep
				} else {
					printTag = console.VideoStep
				}
				dbg.print(printTag, "%s", dbg.lastResult.GetString(dbg.disasm.Symtable, result.StyleFull))
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
		err = dbg.gui.Reset()
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
		_, err := dbg.parseInput("VERBOSITY; STEPMODE; ONHALT; ONSTEP", false)
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
				info, err := dbg.gui.GetState(television.ReqTVSpec)
				if err != nil {
					return doNothing, err
				}
				dbg.print(console.MachineInfo, info.(string))
			default:
				return doNothing, fmt.Errorf("unknown request (%s)", option)
			}
		} else {
			dbg.printMachineInfo(dbg.gui)
		}

	// information about the machine (sprites, playfield)
	case cmdPlayer:
		plyr := -1

		tok, _ := tokens.Get()
		switch tok {
		case "0":
			plyr = 0
		case "1":
			plyr = 1
		default:
			tokens.Unget()
		}

		if dbg.machineInfoVerbose {
			// arrange the two player's information side by side in order to
			// save space and to allow for easy comparison

			switch plyr {
			case 0:
				p0 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Player0), "\n")
				dbg.print(console.MachineInfo, strings.Join(p0, "\n"))

			case 1:
				p1 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Player1), "\n")
				dbg.print(console.MachineInfo, strings.Join(p1, "\n"))

			default:
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
			}
		} else {
			switch plyr {
			case 0:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Player0)

			case 1:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Player1)

			default:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Player0)
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Player1)
			}
		}

	case cmdMissile:
		mssl := -1

		tok, _ := tokens.Get()
		switch tok {
		case "0":
			mssl = 0
		case "1":
			mssl = 1
		default:
			tokens.Unget()
		}

		if dbg.machineInfoVerbose {
			switch mssl {
			case 0:
				m0 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile0), "\n")
				dbg.print(console.MachineInfo, strings.Join(m0, "\n"))

			case 1:
				m1 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile0), "\n")
				dbg.print(console.MachineInfo, strings.Join(m1, "\n"))

			default:
				// arrange the two missile's information side by side in order to
				// save space and to allow for easy comparison

				m0 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile0), "\n")
				m1 := strings.Split(dbg.getMachineInfo(dbg.vcs.TIA.Video.Missile1), "\n")

				ml := 0
				for i := range m0 {
					if len(m0[i]) > ml {
						ml = len(m0[i])
					}
				}

				s := strings.Builder{}
				for i := range m0 {
					if m0[i] != "" {
						s.WriteString(fmt.Sprintf("%s %s | %s\n", m0[i], strings.Repeat(" ", ml-len(m0[i])), m1[i]))
					}
				}
				dbg.print(console.MachineInfo, s.String())
			}
		} else {
			switch mssl {
			case 0:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile0)

			case 1:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile1)

			default:
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile0)
				dbg.printMachineInfo(dbg.vcs.TIA.Video.Missile1)
			}
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
				err = dbg.gui.SetFeature(gui.ReqSetVisibility, false)
				if err != nil {
					return doNothing, err
				}
			case "DEBUG":
				err = dbg.gui.SetFeature(gui.ReqToggleMasking)
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

				err = dbg.gui.SetFeature(gui.ReqSetScale, float32(scale))
				return doNothing, err
			case "DEBUGCOLORS":
				err = dbg.gui.SetFeature(gui.ReqToggleAltColors)
				if err != nil {
					return doNothing, err
				}
			case "METASIGNALS":
				err = dbg.gui.SetFeature(gui.ReqToggleShowSystemState)
				if err != nil {
					return doNothing, err
				}
			default:
				return doNothing, fmt.Errorf("unknown display action (%s)", action)
			}
		} else {
			err = dbg.gui.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdStick:
		stick, _ := tokens.Get()
		action, _ := tokens.Get()

		stickN, _ := strconv.Atoi(stick)

		err := dbg.vcs.Controller.HandleStick(stickN, action)
		if err != nil {
			return doNothing, err
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
