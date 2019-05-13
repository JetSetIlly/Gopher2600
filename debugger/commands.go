package debugger

import (
	"fmt"
	"gopher2600/debugger/commandline"
	"gopher2600/debugger/console"
	"gopher2600/debugger/script"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/peripherals"
	"gopher2600/symbols"
	"os"
	"sort"
	"strconv"
	"strings"
)

// debugger keywords. not a useful data structure but we can use these to form
// the more useful DebuggerCommands and Help structures
const (
	cmdBall          = "BALL"
	cmdBreak         = "BREAK"
	cmdCPU           = "CPU"
	cmdCartridge     = "CARTRIDGE"
	cmdClear         = "CLEAR"
	cmdDebuggerState = "DEBUGGERSTATE"
	cmdDigest        = "DIGEST"
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

var commandTemplate = []string{
	cmdBall,
	cmdBreak + " [%*]",
	cmdCPU,
	cmdCartridge + " (ANALYSIS)",
	cmdClear + " [BREAKS|TRAPS|WATCHES]",
	cmdDebuggerState,
	cmdDigest + " (RESET)",
	cmdDisassembly,
	cmdDisplay + " (OFF|DEBUG|SCALE [%I]|DEBUGCOLORS)", // see notes
	cmdDrop + " [BREAK|TRAP|WATCH] %V",
	cmdGrep + " %S",
	cmdHelp + " %*",
	cmdHexLoad + " %V %V %*",
	cmdInsert + " %F",
	cmdLast + " (DEFN)",
	cmdList + " [BREAKS|TRAPS|WATCHES]",
	cmdMemMap,
	cmdMissile + "(0|1)",
	cmdOnHalt + " (OFF|RESTORE|%*)",
	cmdOnStep + " (OFF|RESTORE|%*)",
	cmdPeek + " [%V|%S] %*",
	cmdPlayer + "(0|1)",
	cmdPlayfield,
	cmdPoke + " [%V|%S] %V",
	cmdQuit,
	cmdRAM,
	cmdRIOT,
	cmdReset,
	cmdRun,
	cmdScript + " [WRITE [%S]|END|%F]",
	cmdStep + " (CPU|VIDEO|SCANLINE)", // see notes
	cmdStepMode + " (CPU|VIDEO)",
	cmdStick + "[0|1] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",
	cmdSymbol + " [%S (ALL|MIRRORS)|LIST (LOCATIONS|READ|WRITE)]",
	cmdTIA,
	cmdTV + " (SPEC)",
	cmdTerse,
	cmdTrap + " [%*]",
	cmdVerbose,
	cmdVerbosity,
	cmdWatch + " (READ|WRITE) %V (%V)",
}

// list of commands that should not be executed when recording/playing scripts
var scriptUnsafeTemplate = []string{
	cmdScript + " [WRITE [%S]]",
	cmdRun,
}

var debuggerCommands *commandline.Commands
var scriptUnsafeCommands *commandline.Commands
var debuggerCommandsIdx *commandline.Index

func init() {
	var err error

	// parse command template
	debuggerCommands, err = commandline.ParseCommandTemplate(commandTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(100)
	}
	sort.Stable(debuggerCommands)

	scriptUnsafeCommands, err = commandline.ParseCommandTemplate(scriptUnsafeTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(100)
	}
	sort.Stable(scriptUnsafeCommands)

	debuggerCommandsIdx = commandline.CreateIndex(debuggerCommands)
}

type parseCommandResult int

const (
	doNothing parseCommandResult = iota
	emptyInput
	stepContinue
	setDefaultStep
	scriptRecordStarted
	scriptRecordEnded
)

// parseCommand/enactCommand scans user input for a valid command and acts upon
// it. commands that cause the emulation to move forward (RUN, STEP) return
// true for the first return value. other commands return false and act upon
// the command immediately. note that the empty string is the same as the STEP
// command

func (dbg *Debugger) parseCommand(userInput *string, interactive bool) (parseCommandResult, error) {
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
	err := debuggerCommands.ValidateTokens(tokens)
	if err != nil {
		return doNothing, err
	}

	// the absolute best thing about the ValidateTokens() function is that we
	// don't need to worrying too much about the success of tokens.Get() in the
	// enactCommand() function below:
	//
	//   arg, _ := tokens.Get()
	//
	// is an acceptable pattern even when an argument is required. the
	// ValidateTokens() function has already caught invalid attempts.
	//
	// default values can be handled thus:
	//
	//  arg, ok := tokens.Get()
	//  if ok {
	//    switch arg {
	//		...
	//	  }
	//  } else {
	//	  // default action
	//    ...
	//  }
	//
	// unfortunately, there is no way currently to handle the case where the
	// command templates don't match expectation in the code below. the code
	// won't break but some error messages may be misleading but hopefully, it
	// will be obvious something went wrong.

	// test to see if command is allowed when recording/playing a script
	if dbg.scriptScribe.IsActive() || !interactive {
		tokens.Reset()

		err := scriptUnsafeCommands.ValidateTokens(tokens)

		// fail when the tokens DO match the scriptUnsafe template (ie. when
		// there is no err from the validate function)
		if err == nil {
			return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("'%s' is unsafe to use in scripts", tokens.String()))
		}
	}

	return dbg.enactCommand(tokens, interactive)
}

func (dbg *Debugger) enactCommand(tokens *commandline.Tokens, interactive bool) (parseCommandResult, error) {
	// check first token. if this token makes sense then we will consume the
	// rest of the tokens appropriately
	tokens.Reset()
	command, _ := tokens.Get()

	// take uppercase value of the first token. it's useful to take the
	// uppercase value but we have to be careful when we do it because
	command = strings.ToUpper(command)

	switch command {
	default:
		return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("%s is not yet implemented", command))

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
		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "WRITE":
			var err error
			saveFile, _ := tokens.Get()
			err = dbg.scriptScribe.StartSession(saveFile)
			if err != nil {
				return doNothing, err
			}
			return scriptRecordStarted, nil

		case "END":
			dbg.scriptScribe.Rollback()
			err := dbg.scriptScribe.EndSession()
			return scriptRecordEnded, err

		default:
			// run a script
			plb, err := script.StartPlayback(option)
			if err != nil {
				return doNothing, err
			}

			if dbg.scriptScribe.IsActive() {
				// if we're currently recording a script we want to write this
				// command to the new script file...

				if err != nil {
					return doNothing, err
				}

				// ... but indicate that we'll be entering a new script and so
				// don't want to repeat the commands from that script
				dbg.scriptScribe.StartPlayback()

				defer func() {
					dbg.scriptScribe.EndPlayback()
				}()

				// TODO: provide a recording option to allow insertion of
				// the actual script commands rather than the call to the
				// script itself
			}

			err = dbg.inputLoop(plb, false)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdDisassembly:
		dbg.disasm.Dump(dbg.printStyle(console.Feedback))

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
		tok, _ := tokens.Get()
		switch strings.ToUpper(tok) {
		case "LIST":
			option, present := tokens.Get()
			if present {
				switch strings.ToUpper(option) {
				default:
					// already caught by command line ValidateTokens()

				case "LOCATIONS":
					dbg.disasm.Symtable.ListLocations(dbg.printStyle(console.Feedback))

				case "READ":
					dbg.disasm.Symtable.ListReadSymbols(dbg.printStyle(console.Feedback))

				case "WRITE":
					dbg.disasm.Symtable.ListWriteSymbols(dbg.printStyle(console.Feedback))
				}
			} else {
				dbg.disasm.Symtable.ListSymbols(dbg.printStyle(console.Feedback))
			}

		default:
			symbol := tok
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
				switch strings.ToUpper(option) {
				default:
					// already caught by command line ValidateTokens()

				case "ALL", "MIRRORS":
					dbg.print(console.Feedback, "%s -> %#04x", symbol, address)

					// find all instances of symbol address in memory space
					// assumption: the address returned by SearchSymbol is the
					// first address in the complete list
					for m := address + 1; m < dbg.vcs.Mem.Cart.Origin(); m++ {
						if dbg.vcs.Mem.MapAddress(m, table == symbols.ReadSymTable) == address {
							dbg.print(console.Feedback, "%s -> %#04x", symbol, m)
						}
					}
				}
			} else {
				dbg.print(console.Feedback, "%s -> %#04x", symbol, address)
			}
		}

	case cmdBreak:
		err := dbg.breakpoints.parseBreakpoint(tokens)
		if err != nil {
			return doNothing, errors.NewFormattedError(errors.CommandError, err)
		}

	case cmdTrap:
		err := dbg.traps.parseTrap(tokens)
		if err != nil {
			return doNothing, errors.NewFormattedError(errors.CommandError, err)
		}

	case cmdWatch:
		err := dbg.watches.parseWatch(tokens, dbg.dbgmem)
		if err != nil {
			return doNothing, errors.NewFormattedError(errors.CommandError, err)
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
			// already caught by command line ValidateTokens()
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
			// already caught by command line ValidateTokens()
		}

	case cmdDrop:
		drop, _ := tokens.Get()

		s, _ := tokens.Get()
		num, err := strconv.Atoi(s)
		if err != nil {
			return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("drop attribute must be a number (%s)", s))
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
			// already caught by command line ValidateTokens()
		}

	case cmdOnHalt:
		if tokens.Remaining() == 0 {
			dbg.print(console.Feedback, "auto-command on halt: %s", dbg.commandOnHalt)
			return doNothing, nil
		}

		// TODO: non-interactive check of tokens against scriptUnsafeTemplate

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
		_, err := dbg.parseInput(dbg.commandOnHalt, false, false)
		return doNothing, err

	case cmdOnStep:
		if tokens.Remaining() == 0 {
			dbg.print(console.Feedback, "auto-command on step: %s", dbg.commandOnStep)
			return doNothing, nil
		}

		// TODO: non-interactive check of tokens against scriptUnsafeTemplate

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
		_, err := dbg.parseInput(dbg.commandOnStep, false, false)
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
				var printTag console.Style
				if dbg.lastResult.Final {
					printTag = console.CPUStep
				} else {
					printTag = console.VideoStep
				}
				dbg.print(printTag, "%s", dbg.lastResult.GetString(dbg.disasm.Symtable, result.StyleExecution))
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
		case "SCANLINE":
			dbg.inputEveryVideoCycle = false
			tokens.Unget()
			err := dbg.stepTraps.parseTrap(tokens)
			if err != nil {
				return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("unknown step mode (%s)", mode))
			}
			dbg.runUntilHalt = true
		default:
			// already caught by command line ValidateTokens()
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
				// already caught by command line ValidateTokens()
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
		_, err := dbg.parseInput("VERBOSITY; STEPMODE; ONHALT; ONSTEP", false, false)
		if err != nil {
			return doNothing, err
		}

	case cmdCartridge:
		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "ANALYSIS":
				dbg.print(console.Feedback, dbg.disasm.String())
			}
		} else {
			dbg.printMachineInfo(dbg.vcs.Mem.Cart)
		}

	case cmdCPU:
		dbg.printMachineInfo(dbg.vcs.MC)

	case cmdPeek:
		// get first address token
		a, present := tokens.Get()

		for present {
			// perform peek
			ai, err := dbg.dbgmem.peek(a)
			if err != nil {
				dbg.print(console.Error, "%s", err)
			} else {
				dbg.print(console.MachineInfo, ai.String())
			}

			// loop through all addresses
			a, present = tokens.Get()
		}

	case cmdPoke:
		// get address token
		a, _ := tokens.Get()

		// get value token
		v, _ := tokens.Get()

		val, err := strconv.ParseUint(v, 0, 8)
		if err != nil {
			dbg.print(console.Error, "poke value must be 8bit number (%s)", v)
			return doNothing, nil
		}

		// perform single poke
		ai, err := dbg.dbgmem.poke(a, uint8(val))
		if err != nil {
			dbg.print(console.Error, "%s", err)
		} else {
			dbg.print(console.MachineInfo, ai.String())
		}

	case cmdHexLoad:
		// get address token
		a, _ := tokens.Get()

		addr, err := strconv.ParseUint(a, 0, 16)
		if err != nil {
			dbg.print(console.Error, "hexload address must be 16bit number (%s)", a)
			return doNothing, nil
		}

		// get (first) value token
		v, present := tokens.Get()

		for present {
			val, err := strconv.ParseUint(v, 0, 8)
			if err != nil {
				dbg.print(console.Error, "hexload value must be 8bit number (%s)", addr)
				v, present = tokens.Get()
				continue // for loop (without advancing address)
			}

			// perform individual poke
			ai, err := dbg.dbgmem.poke(uint16(addr), uint8(val))
			if err != nil {
				dbg.print(console.Error, "%s", err)
			} else {
				dbg.print(console.MachineInfo, ai.String())
			}

			// loop through all values
			v, present = tokens.Get()
			addr++
		}

	case cmdRAM:
		dbg.printMachineInfo(dbg.vcs.Mem.PIA)

	case cmdRIOT:
		dbg.printMachineInfo(dbg.vcs.RIOT)

	case cmdTIA:
		dbg.printMachineInfo(dbg.vcs.TIA)

	case cmdTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				spec := dbg.gui.GetSpec()
				dbg.print(console.MachineInfo, spec.ID)
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printMachineInfo(dbg.gui)
		}

	// information about the machine (sprites, playfield)
	case cmdPlayer:
		plyr := -1

		arg, _ := tokens.Get()
		switch arg {
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

		arg, _ := tokens.Get()
		switch arg {
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
					return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("value required for %s %s", command, action))
				}

				scale, err := strconv.ParseFloat(scl, 32)
				if err != nil {
					return doNothing, errors.NewFormattedError(errors.CommandError, fmt.Sprintf("%s %s value not valid (%s)", command, action, scl))
				}

				err = dbg.gui.SetFeature(gui.ReqSetScale, float32(scale))
				return doNothing, err
			case "DEBUGCOLORS":
				err = dbg.gui.SetFeature(gui.ReqToggleAltColors)
				if err != nil {
					return doNothing, err
				}
			case "METASIGNALS":
				err = dbg.gui.SetFeature(gui.ReqToggleShowMetaPixels)
				if err != nil {
					return doNothing, err
				}
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			err = dbg.gui.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdStick:
		var err error

		stick, _ := tokens.Get()
		action, _ := tokens.Get()

		var event peripherals.Event
		switch strings.ToUpper(action) {
		case "UP":
			event = peripherals.Up
		case "DOWN":
			event = peripherals.Down
		case "LEFT":
			event = peripherals.Left
		case "RIGHT":
			event = peripherals.Right
		case "NOUP":
			event = peripherals.NoUp
		case "NODOWN":
			event = peripherals.NoDown
		case "NOLEFT":
			event = peripherals.NoLeft
		case "NORIGHT":
			event = peripherals.NoRight
		case "FIRE":
			event = peripherals.Fire
		case "NOFIRE":
			event = peripherals.NoFire
		}

		n, _ := strconv.Atoi(stick)
		switch n {
		case 0:
			err = dbg.vcs.Ports.Player0.Handle(event)
		case 1:
			err = dbg.vcs.Ports.Player1.Handle(event)
		}

		if err != nil {
			return doNothing, err
		}

	case cmdDigest:
		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "RESET":
				dbg.digest.ResetDigest()
			}
		} else {
			dbg.print(console.Feedback, dbg.digest.String())
		}
	}

	return doNothing, nil
}
