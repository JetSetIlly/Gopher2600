package debugger

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/debugger/commandline"
	"gopher2600/debugger/console"
	"gopher2600/debugger/script"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/memorymap"
	"gopher2600/hardware/peripherals"
	"gopher2600/symbols"
	"os"
	"sort"
	"strconv"
	"strings"
)

// debugger keywords
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
	cmdHexLoad       = "HEXLOAD"
	cmdInsert        = "INSERT"
	cmdLast          = "LAST"
	cmdList          = "LIST"
	cmdMemMap        = "MEMMAP"
	cmdReflect       = "REFLECT"
	cmdMissile       = "MISSILE"
	cmdOnHalt        = "ONHALT"
	cmdOnStep        = "ONSTEP"
	cmdPeek          = "PEEK"
	cmdPanel         = "PANEL"
	cmdPlayer        = "PLAYER"
	cmdPlayfield     = "PLAYFIELD"
	cmdPoke          = "POKE"
	cmdQuit          = "QUIT"
	cmdExit          = "EXIT"
	cmdRAM           = "RAM"
	cmdRIOT          = "RIOT"
	cmdReset         = "RESET"
	cmdRun           = "RUN"
	cmdScript        = "SCRIPT"
	cmdStep          = "STEP"
	cmdGranularity   = "GRANULARITY"
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

const cmdHelp = "HELP"

var commandTemplate = []string{
	cmdBall,
	cmdBreak + " [%S %N|%N] {& %S %N|& %N}",
	cmdCPU + " (SET [PC|A|X|Y|SP] [%N]|BUG (ON|OFF))",
	cmdCartridge + " (ANALYSIS|BANK %N)",
	cmdClear + " [BREAKS|TRAPS|WATCHES|ALL]",
	cmdDebuggerState,
	cmdDigest + " (RESET)",
	cmdDisassembly,
	cmdDisplay + " (ON|OFF|DEBUG (ON|OFF)|SCALE [%P]|ALT (ON|OFF)|OVERLAY (ON|OFF))", // see notes
	cmdDrop + " [BREAK|TRAP|WATCH] %N",
	cmdGrep + " %S",
	cmdHexLoad + " %N %N {%N}",
	cmdInsert + " %F",
	cmdLast + " (DEFN)",
	cmdList + " [BREAKS|TRAPS|WATCHES|ALL]",
	cmdMemMap,
	cmdReflect + " (ON|OFF)",
	cmdMissile + " (0|1)",
	cmdOnHalt + " (OFF|ON|%S {%S})",
	cmdOnStep + " (OFF|ON|%S {%S})",
	cmdPeek + " [%S] {%S}",
	cmdPanel + " (SET [P0PRO|P1PRO|P0AM|P1AM|COL|BW]|TOGGLE [P0|P1|COL])",
	cmdPlayer + " (0|1)",
	cmdPlayfield,
	cmdPoke + " [%S] %N",
	cmdQuit,
	cmdExit,
	cmdRAM + " (CART)",
	cmdRIOT + " (TIMER)",
	cmdReset,
	cmdRun,
	cmdScript + " [WRITE %S|END|%F]",
	cmdStep + " (CPU|VIDEO|%S)",
	cmdGranularity + " (CPU|VIDEO)",
	cmdStick + " [0|1] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",
	cmdSymbol + " [%S (ALL|MIRRORS)|LIST (LOCATIONS|READ|WRITE)]",
	cmdTIA + " (DELAYS)",
	cmdTV + " (SPEC)",
	cmdTerse,
	cmdTrap + " [%S] {%S}",
	cmdVerbose,
	cmdVerbosity,
	cmdWatch + " (READ|WRITE) [%S] (%S)",
}

// list of commands that should not be executed when recording/playing scripts
var scriptUnsafeTemplate = []string{
	cmdScript + " [WRITE [%S]]",
	cmdRun,
}

var debuggerCommands *commandline.Commands
var scriptUnsafeCommands *commandline.Commands
var debuggerCommandsIdx *commandline.Index

// this init() function "compiles" the commandTemplate above into a more
// usuable form. It will cause the program to fail if the template is invalid.
func init() {
	var err error

	// parse command template
	debuggerCommands, err = commandline.ParseCommandTemplateWithOutput(commandTemplate, os.Stdout)
	if err != nil {
		fmt.Println(err)
		os.Exit(100)
	}

	err = debuggerCommands.AddHelp(cmdHelp)
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

	// make sure all tokens have been handled. this should only happen if
	// input has been allowed by ValidateTokens() but has not been
	// explicitely consumed by entactCommand(). a false positive might occur if
	// the token queue has been Peek()ed rather than Get()ed
	if interactive {
		defer func() {
			if !tokens.IsEnd() {
				dbg.print(console.StyleError, fmt.Sprintf("unhandled arguments in user input (%s)", tokens.Remainder()))
			}
		}()
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
			return doNothing, errors.New(errors.CommandError, fmt.Sprintf("'%s' is unsafe to use in scripts", tokens.String()))
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
		return doNothing, errors.New(errors.CommandError, fmt.Sprintf("%s is not yet implemented", command))

	case cmdHelp:
		keyword, present := tokens.Get()
		if present {
			keyword = strings.ToUpper(keyword)

			helpTxt, ok := Help[keyword]
			if !ok {
				dbg.print(console.StyleHelp, "no help for %s", keyword)
			} else {
				helpTxt = fmt.Sprintf("%s\n\n  Usage: %s", helpTxt, (*debuggerCommandsIdx)[keyword].String())
				dbg.print(console.StyleHelp, helpTxt)
			}
		} else {
			dbg.print(console.StyleHelp, debuggerCommands.String())
		}

	case cmdInsert:
		cart, _ := tokens.Get()
		err := dbg.loadCartridge(cartridgeloader.Loader{Filename: cart})
		if err != nil {
			return doNothing, err
		}
		dbg.print(console.StyleFeedback, "machine reset with new cartridge (%s)", cart)

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
				// command to the new script file but indicate that we'll be
				// entering a new script and so don't want to repeat the
				// commands from that script
				dbg.scriptScribe.StartPlayback()

				defer func() {
					dbg.scriptScribe.EndPlayback()
				}()

				// !!TODO: provide a recording option to allow insertion of
				// the actual script commands rather than the call to the
				// script itself
			}

			err = dbg.inputLoop(plb, false)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdDisassembly:
		dbg.disasm.Dump(dbg.printStyle(console.StyleFeedback))

	case cmdGrep:
		search, _ := tokens.Get()
		output := strings.Builder{}
		dbg.disasm.Grep(search, &output, false, 3)
		if output.Len() == 0 {
			dbg.print(console.StyleError, "%s not found in disassembly", search)
		} else {
			dbg.print(console.StyleFeedback, output.String())
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
					dbg.disasm.Symtable.ListLocations(dbg.printStyle(console.StyleFeedback))

				case "READ":
					dbg.disasm.Symtable.ListReadSymbols(dbg.printStyle(console.StyleFeedback))

				case "WRITE":
					dbg.disasm.Symtable.ListWriteSymbols(dbg.printStyle(console.StyleFeedback))
				}
			} else {
				dbg.disasm.Symtable.ListSymbols(dbg.printStyle(console.StyleFeedback))
			}

		default:
			symbol := tok
			table, symbol, address, err := dbg.disasm.Symtable.SearchSymbol(symbol, symbols.UnspecifiedSymTable)
			if err != nil {
				if errors.Is(err, errors.SymbolUnknown) {
					dbg.print(console.StyleFeedback, "%s -> not found", symbol)
					return doNothing, nil
				}
				return doNothing, err
			}

			option, present := tokens.Get()
			if present {
				switch strings.ToUpper(option) {
				default:
					// already caught by command line ValidateTokens()

				case "ALL", "MIRRORS":
					dbg.print(console.StyleFeedback, "%s -> %#04x", symbol, address)

					// find all instances of symbol address in memory space
					// assumption: the address returned by SearchSymbol is the
					// first address in the complete list
					for m := address + 1; m < memorymap.OriginCart; m++ {
						ma, _ := memorymap.MapAddress(m, table == symbols.ReadSymTable)
						if ma == address {
							dbg.print(console.StyleFeedback, "%s -> %#04x", symbol, m)
						}
					}
				}
			} else {
				dbg.print(console.StyleFeedback, "%s -> %#04x", symbol, address)
			}
		}

	case cmdBreak:
		err := dbg.breakpoints.parseBreakpoint(tokens)
		if err != nil {
			return doNothing, errors.New(errors.CommandError, err)
		}

	case cmdTrap:
		err := dbg.traps.parseTrap(tokens)
		if err != nil {
			return doNothing, errors.New(errors.CommandError, err)
		}

	case cmdWatch:
		err := dbg.watches.parseWatch(tokens, dbg.dbgmem)
		if err != nil {
			return doNothing, errors.New(errors.CommandError, err)
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
		case "ALL":
			// !!TODO: refine output. requires headings
			dbg.breakpoints.list()
			dbg.traps.list()
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
			dbg.print(console.StyleFeedback, "breakpoints cleared")
		case "TRAPS":
			dbg.traps.clear()
			dbg.print(console.StyleFeedback, "traps cleared")
		case "WATCHES":
			dbg.watches.clear()
			dbg.print(console.StyleFeedback, "watches cleared")
		case "ALL":
			dbg.breakpoints.clear()
			dbg.traps.clear()
			dbg.watches.clear()
			dbg.print(console.StyleFeedback, "breakpoints, traps and watches cleared")
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdDrop:
		drop, _ := tokens.Get()

		s, _ := tokens.Get()
		num, err := strconv.Atoi(s)
		if err != nil {
			return doNothing, errors.New(errors.CommandError, fmt.Sprintf("drop attribute must be a number (%s)", s))
		}

		drop = strings.ToUpper(drop)
		switch drop {
		case "BREAK":
			err := dbg.breakpoints.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.StyleFeedback, "breakpoint #%d dropped", num)
		case "TRAP":
			err := dbg.traps.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.StyleFeedback, "trap #%d dropped", num)
		case "WATCH":
			err := dbg.watches.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.print(console.StyleFeedback, "watch #%d dropped", num)
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdOnHalt:
		if tokens.Remaining() == 0 {
			if dbg.commandOnHalt == "" {
				dbg.print(console.StyleFeedback, "auto-command on halt: OFF")
			} else {
				dbg.print(console.StyleFeedback, "auto-command on halt: %s", dbg.commandOnHalt)
			}
			return doNothing, nil
		}

		// !!TODO: non-interactive check of tokens against scriptUnsafeTemplate
		var newOnHalt string

		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			newOnHalt = ""
		case "ON":
			newOnHalt = dbg.commandOnHaltStored
		default:
			// token isn't one we recognise so push it back onto the token queue
			tokens.Unget()

			// use remaininder of command line to form the ONHALT command sequence
			newOnHalt = tokens.Remainder()
			tokens.End()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			newOnHalt = strings.Replace(newOnHalt, ",", ";", -1)
		}

		dbg.commandOnHalt = newOnHalt

		// display the new/restored ONHALT command(s)
		if newOnHalt == "" {
			dbg.print(console.StyleFeedback, "auto-command on halt: OFF")
		} else {
			dbg.print(console.StyleFeedback, "auto-command on halt: %s", dbg.commandOnHalt)

			// store the new command so we can reuse it after an ONHALT OFF
			//
			// !!TODO: normalise case of specified command sequence
			dbg.commandOnHaltStored = newOnHalt
		}

		return doNothing, nil

	case cmdOnStep:
		if tokens.Remaining() == 0 {
			if dbg.commandOnStep == "" {
				dbg.print(console.StyleFeedback, "auto-command on step: OFF")
			} else {
				dbg.print(console.StyleFeedback, "auto-command on step: %s", dbg.commandOnStep)
			}
			return doNothing, nil
		}

		// !!TODO: non-interactive check of tokens against scriptUnsafeTemplate
		var newOnStep string

		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			newOnStep = ""
		case "ON":
			newOnStep = dbg.commandOnStepStored
		default:
			// token isn't one we recognise so push it back onto the token queue
			tokens.Unget()

			// use remaininder of command line to form the ONSTEP command sequence
			newOnStep = tokens.Remainder()
			tokens.End()

			// we can't use semi-colons when specifying the sequence so allow use of
			// commas to act as an alternative
			newOnStep = strings.Replace(newOnStep, ",", ";", -1)
		}

		dbg.commandOnStep = newOnStep

		// display the new/restored ONSTEP command(s)
		if newOnStep == "" {
			dbg.print(console.StyleFeedback, "auto-command on step: OFF")
		} else {
			dbg.print(console.StyleFeedback, "auto-command on step: %s", dbg.commandOnStep)

			// store the new command so we can reuse it after an ONSTEP OFF
			// !!TODO: normalise case of specified command sequence
			dbg.commandOnStepStored = newOnStep
		}

		return doNothing, nil

	case cmdLast:
		option, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(option) {
			case "DEFN":
				dbg.print(console.StyleFeedback, "%s", dbg.vcs.CPU.LastResult.Defn)
			}
		} else {
			var printTag console.Style
			if dbg.vcs.CPU.LastResult.Final {
				printTag = console.StyleCPUStep
			} else {
				printTag = console.StyleVideoStep
			}
			dbg.print(printTag, "%s", dbg.vcs.CPU.LastResult.GetString(dbg.disasm.Symtable, result.StyleExecution))
		}

	case cmdMemMap:
		dbg.print(console.StyleInstrument, "%v", memorymap.Summary())

	case cmdReflect:
		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			err := dbg.scr.SetFeature(gui.ReqSetOverlay, false)
			if err != nil {
				dbg.print(console.StyleError, err.Error())
			}
			dbg.relfectMonitor.Activate(false)
		case "ON":
			err := dbg.scr.SetFeature(gui.ReqSetOverlay, true)
			if err != nil {
				dbg.print(console.StyleError, err.Error())
			}
			dbg.relfectMonitor.Activate(true)
		}
		if dbg.relfectMonitor.IsActive() {
			dbg.print(console.StyleEmulatorInfo, "reflection: ON")
		} else {
			dbg.print(console.StyleEmulatorInfo, "reflection: OFF")
		}

	case cmdExit:
		fallthrough

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
		dbg.print(console.StyleFeedback, "machine reset")

	case cmdRun:
		if !dbg.scr.IsVisible() && dbg.commandOnStep == "" {
			dbg.print(console.StyleEmulatorInfo, "running with no display or terminal output")
		}
		dbg.runUntilHalt = true
		return stepContinue, nil

	case cmdStep:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "":
			// calling step with no argument is the normal case
		case "CPU":
			// changes granularity
			dbg.inputEveryVideoCycle = false
		case "VIDEO":
			// changes granularity
			dbg.inputEveryVideoCycle = true
		default:
			dbg.inputEveryVideoCycle = false
			tokens.Unget()
			err := dbg.stepTraps.parseTrap(tokens)
			if err != nil {
				return doNothing, errors.New(errors.CommandError, fmt.Sprintf("unknown step mode (%s)", mode))
			}
			dbg.runUntilHalt = true
		}

		return stepContinue, nil

	case cmdGranularity:
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
		dbg.print(console.StyleFeedback, "granularity: %s", mode)

	case cmdTerse:

	case cmdVerbose:

	case cmdVerbosity:

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
				dbg.print(console.StyleFeedback, dbg.disasm.String())
			case "BANK":
				bank, _ := tokens.Get()
				n, _ := strconv.Atoi(bank)
				dbg.vcs.Mem.Cart.SetBank(dbg.vcs.CPU.PC.Address(), n)

				err := dbg.vcs.CPU.LoadPCIndirect(addresses.Reset)
				if err != nil {
					return doNothing, err
				}
			}
		} else {
			dbg.printInstrument(dbg.vcs.Mem.Cart)
		}

	case cmdCPU:
		action, present := tokens.Get()
		if present {
			switch strings.ToUpper(action) {
			case "SET":
				target, _ := tokens.Get()
				value, _ := tokens.Get()

				target = strings.ToUpper(target)
				if target == "PC" {
					// program counter can be a 16 bit number
					v, err := strconv.ParseUint(value, 0, 16)
					if err != nil {
						dbg.print(console.StyleError, "value must be a positive 16 number")
					}

					dbg.vcs.CPU.PC.Load(uint16(v))
				} else {
					// 6507 registers are 8 bit
					v, err := strconv.ParseUint(value, 0, 8)
					if err != nil {
						dbg.print(console.StyleError, "value must be a positive 8 number")
					}

					var reg *registers.Register
					switch strings.ToUpper(target) {
					case "A":
						reg = dbg.vcs.CPU.A
					case "X":
						reg = dbg.vcs.CPU.X
					case "Y":
						reg = dbg.vcs.CPU.Y
					case "SP":
						reg = dbg.vcs.CPU.SP
					}

					reg.Load(uint8(v))
				}

			case "BUG":
				option, _ := tokens.Get()

				switch strings.ToUpper(option) {
				case "ON":
					dbg.reportCPUBugs = true
				case "OFF":
					dbg.reportCPUBugs = false
				}

				if dbg.reportCPUBugs {
					dbg.print(console.StyleFeedback, "CPU bug reporting: ON")
				} else {
					dbg.print(console.StyleFeedback, "CPU bug reporting: OFF")
				}

			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printInstrument(dbg.vcs.CPU)
		}

	case cmdPeek:
		// get first address token
		a, present := tokens.Get()

		for present {
			// perform peek
			ai, err := dbg.dbgmem.peek(a)
			if err != nil {
				dbg.print(console.StyleError, "%s", err)
			} else {
				dbg.print(console.StyleInstrument, ai.String())
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
			dbg.print(console.StyleError, "poke value must be 8bit number (%s)", v)
			return doNothing, nil
		}

		// perform single poke
		ai, err := dbg.dbgmem.poke(a, uint8(val))
		if err != nil {
			dbg.print(console.StyleError, "%s", err)
		} else {
			dbg.print(console.StyleInstrument, ai.String())
		}

	case cmdHexLoad:
		// get address token
		a, _ := tokens.Get()

		addr, err := strconv.ParseUint(a, 0, 16)
		if err != nil {
			dbg.print(console.StyleError, "hexload address must be 16bit number (%s)", a)
			return doNothing, nil
		}

		// get (first) value token
		v, present := tokens.Get()

		for present {
			val, err := strconv.ParseUint(v, 0, 8)
			if err != nil {
				dbg.print(console.StyleError, "hexload value must be 8bit number (%s)", addr)
				v, present = tokens.Get()
				continue // for loop (without advancing address)
			}

			// perform individual poke
			ai, err := dbg.dbgmem.poke(uint16(addr), uint8(val))
			if err != nil {
				dbg.print(console.StyleError, "%s", err)
			} else {
				dbg.print(console.StyleInstrument, ai.String())
			}

			// loop through all values
			v, present = tokens.Get()
			addr++
		}

	case cmdRAM:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "CART":
				cartRAM := dbg.vcs.Mem.Cart.RAM()
				if len(cartRAM) > 0 {
					// !!TODO: better presentation of cartridge RAM
					dbg.print(console.StyleInstrument, fmt.Sprintf("%v", dbg.vcs.Mem.Cart.RAM()))
				} else {
					dbg.print(console.StyleFeedback, "cartridge does not contain any additional RAM")
				}

			}
		} else {
			dbg.printInstrument(dbg.vcs.Mem.PIA)
		}

	case cmdRIOT:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "TIMER":
				dbg.printInstrument(dbg.vcs.RIOT.Timer)
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printInstrument(dbg.vcs.RIOT)
		}

	case cmdTIA:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "DELAYS":
				// for convience asking for TIA delays also prints delays for
				// the sprites
				dbg.printInstrument(dbg.vcs.TIA.Video.Player0.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Player1.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Missile0.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Missile1.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Ball.Delay)
			}
		} else {
			dbg.printInstrument(dbg.vcs.TIA)
		}

	case cmdTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				dbg.print(console.StyleInstrument, dbg.tv.GetSpec().ID)
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printInstrument(dbg.tv)
		}

	case cmdPanel:
		mode, _ := tokens.Get()
		switch strings.ToUpper(mode) {
		case "TOGGLE":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0":
				dbg.vcs.Panel.Handle(peripherals.PanelTogglePlayer0Pro)
			case "P1":
				dbg.vcs.Panel.Handle(peripherals.PanelTogglePlayer1Pro)
			case "COL":
				dbg.vcs.Panel.Handle(peripherals.PanelToggleColor)
			}
		case "SET":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0PRO":
				dbg.vcs.Panel.Handle(peripherals.PanelSetPlayer0Pro)
			case "P1PRO":
				dbg.vcs.Panel.Handle(peripherals.PanelSetPlayer1Pro)
			case "P0AM":
				dbg.vcs.Panel.Handle(peripherals.PanelSetPlayer0Am)
			case "P1AM":
				dbg.vcs.Panel.Handle(peripherals.PanelSetPlayer1Am)
			case "COL":
				dbg.vcs.Panel.Handle(peripherals.PanelSetColor)
			case "BW":
				dbg.vcs.Panel.Handle(peripherals.PanelSetBlackAndWhite)
			}
		}
		dbg.printInstrument(dbg.vcs.Panel)

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

		switch plyr {
		case 0:
			dbg.printInstrument(dbg.vcs.TIA.Video.Player0)

		case 1:
			dbg.printInstrument(dbg.vcs.TIA.Video.Player1)

		default:
			dbg.printInstrument(dbg.vcs.TIA.Video.Player0)
			dbg.printInstrument(dbg.vcs.TIA.Video.Player1)
		}

	case cmdMissile:
		miss := -1

		arg, _ := tokens.Get()
		switch arg {
		case "0":
			miss = 0
		case "1":
			miss = 1
		default:
			tokens.Unget()
		}

		switch miss {
		case 0:
			dbg.printInstrument(dbg.vcs.TIA.Video.Missile0)

		case 1:
			dbg.printInstrument(dbg.vcs.TIA.Video.Missile1)

		default:
			dbg.printInstrument(dbg.vcs.TIA.Video.Missile0)
			dbg.printInstrument(dbg.vcs.TIA.Video.Missile1)
		}

	case cmdBall:
		dbg.printInstrument(dbg.vcs.TIA.Video.Ball)

	case cmdPlayfield:
		dbg.printInstrument(dbg.vcs.TIA.Video.Playfield)

	case cmdDisplay:
		var err error

		action, _ := tokens.Get()
		action = strings.ToUpper(action)
		switch action {
		case "ON":
			err = dbg.scr.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return doNothing, err
			}
		case "OFF":
			err = dbg.scr.SetFeature(gui.ReqSetVisibility, false)
			if err != nil {
				return doNothing, err
			}
		case "DEBUG":
			action, _ := tokens.Get()
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				err = dbg.scr.SetFeature(gui.ReqSetMasking, false)
				if err != nil {
					return doNothing, err
				}
			case "ON":
				err = dbg.scr.SetFeature(gui.ReqSetMasking, true)
				if err != nil {
					return doNothing, err
				}
			default:
				err = dbg.scr.SetFeature(gui.ReqToggleMasking)
				if err != nil {
					return doNothing, err
				}
			}
		case "SCALE":
			scl, present := tokens.Get()
			if !present {
				return doNothing, errors.New(errors.CommandError, fmt.Sprintf("value required for %s %s", command, action))
			}

			scale, err := strconv.ParseFloat(scl, 32)
			if err != nil {
				return doNothing, errors.New(errors.CommandError, fmt.Sprintf("%s %s value not valid (%s)", command, action, scl))
			}

			err = dbg.scr.SetFeature(gui.ReqSetScale, float32(scale))
			return doNothing, err
		case "ALT":
			action, _ := tokens.Get()
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				err = dbg.scr.SetFeature(gui.ReqSetAltColors, false)
				if err != nil {
					return doNothing, err
				}
			case "ON":
				err = dbg.scr.SetFeature(gui.ReqSetAltColors, true)
				if err != nil {
					return doNothing, err
				}
			default:
				err = dbg.scr.SetFeature(gui.ReqToggleAltColors)
				if err != nil {
					return doNothing, err
				}
			}
		case "OVERLAY":
			if !dbg.relfectMonitor.IsActive() {
				return doNothing, errors.New(errors.ReflectionNotRunning)
			}

			action, _ := tokens.Get()
			action = strings.ToUpper(action)
			switch action {
			case "OFF":
				err = dbg.scr.SetFeature(gui.ReqSetOverlay, false)
				if err != nil {
					return doNothing, err
				}
			case "ON":
				err = dbg.scr.SetFeature(gui.ReqSetOverlay, true)
				if err != nil {
					return doNothing, err
				}
			default:
				err = dbg.scr.SetFeature(gui.ReqToggleOverlay)
				if err != nil {
					return doNothing, err
				}
			}
		default:
			err = dbg.scr.SetFeature(gui.ReqToggleVisibility)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdStick:
		var err error

		stick, _ := tokens.Get()
		action, _ := tokens.Get()

		var event peripherals.Action
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
			dbg.print(console.StyleFeedback, dbg.digest.String())
		}
	}

	return doNothing, nil
}
