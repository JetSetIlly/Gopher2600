// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"bytes"
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/debugger/script"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/memorymap"
	"gopher2600/hardware/riot/input"
	"gopher2600/patch"
	"gopher2600/symbols"
	"os"
	"sort"
	"strconv"
	"strings"
)

var debuggerCommands *commandline.Commands
var scriptUnsafeCommands *commandline.Commands

// this init() function "compiles" the commandTemplate above into a more
// usuable form. It will cause the program to fail if the template is invalid.
func init() {
	var err error

	// parse command template
	debuggerCommands, err = commandline.ParseCommandTemplate(commandTemplate)
	if err != nil {
		fmt.Println(err)
		os.Exit(100)
	}

	err = debuggerCommands.AddHelp(cmdHelp, helps)
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
}

type parseCommandResult int

const (
	doNothing parseCommandResult = iota
	emptyInput
	stepContinue
	scriptRecordStarted
	scriptRecordEnded
	helpCalled
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
			return doNothing, errors.New(errors.CommandError, fmt.Sprintf("'%s' is unsafe to use in scripts", tokens.String()))
		}
	}

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
			dbg.printLine(terminal.StyleHelp, debuggerCommands.Help(keyword))
		} else {
			dbg.printLine(terminal.StyleHelp, debuggerCommands.HelpOverview())
		}

		return helpCalled, nil

	case cmdQuit:
		if dbg.scriptScribe.IsActive() {
			// we don't want the QUIT command to appear in the script so
			// rollback last entry before we commit it in EndSession()
			dbg.scriptScribe.Rollback()
			dbg.scriptScribe.EndSession()
		} else {
			dbg.running = false
		}

	case cmdReset:
		err := dbg.vcs.Reset()
		if err != nil {
			return doNothing, err
		}
		err = dbg.tv.Reset()
		if err != nil {
			return doNothing, err
		}
		dbg.printLine(terminal.StyleFeedback, "machine reset")

	case cmdRun:
		if !dbg.scr.IsVisible() && dbg.commandOnStep == "" {
			dbg.printLine(terminal.StyleEmulatorInfo, "running with no display or terminal output")
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
			// changes quantum
			dbg.quantum = quantumCPU
		case "VIDEO":
			// changes quantum
			dbg.quantum = quantumVideo
		default:
			// does not change quantum
			tokens.Unget()
			err := dbg.stepTraps.parseTrap(tokens)
			if err != nil {
				return doNothing, errors.New(errors.CommandError, fmt.Sprintf("unknown step mode (%s)", mode))
			}
			dbg.runUntilHalt = true
		}

		return stepContinue, nil

	case cmdQuantum:
		mode, present := tokens.Get()
		if present {
			mode = strings.ToUpper(mode)
			switch mode {
			case "CPU":
				dbg.quantum = quantumCPU
			case "VIDEO":
				dbg.quantum = quantumVideo
			default:
				// already caught by command line ValidateTokens()
			}
		}
		dbg.printLine(terminal.StyleFeedback, "quantum: %s", dbg.quantum)

	case cmdScript:
		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "RECORD":
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
			scr, err := script.RescribeScript(option)
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
			}

			err = dbg.inputLoop(scr, false)
			if err != nil {
				return doNothing, err
			}
		}

	case cmdInsert:
		cart, _ := tokens.Get()
		err := dbg.loadCartridge(cartridgeloader.Loader{Filename: cart})
		if err != nil {
			return doNothing, err
		}
		dbg.printLine(terminal.StyleFeedback, "machine reset with new cartridge (%s)", cart)

	case cmdCartridge:
		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "ANALYSIS":
				dbg.printLine(terminal.StyleFeedback, dbg.disasm.Analysis())
			case "BANK":
				bank, _ := tokens.Get()
				n, _ := strconv.Atoi(bank)
				err := dbg.vcs.Mem.Cart.SetBank(dbg.vcs.CPU.PC.Address(), n)
				if err != nil {
					return doNothing, err
				}

				err = dbg.vcs.CPU.LoadPCIndirect(addresses.Reset)
				if err != nil {
					return doNothing, err
				}
			}
		} else {
			dbg.printInstrument(dbg.vcs.Mem.Cart)
		}

	case cmdPatch:
		f, _ := tokens.Get()
		patched, err := patch.CartridgeMemory(dbg.vcs.Mem.Cart, f)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%v", err)
			if patched {
				dbg.printLine(terminal.StyleEmulatorInfo, "error during patching. cartridge might be unusable.")
			}
			return doNothing, nil
		}
		if patched {
			dbg.printLine(terminal.StyleEmulatorInfo, "cartridge patched")
		}

	case cmdDisassembly:
		bytecode := false
		bank := -1

		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "BYTECODE":
				bytecode = true
			default:
				bank, _ = strconv.Atoi(arg)
			}
		}

		var err error

		s := &bytes.Buffer{}

		if bank == -1 {
			err = dbg.disasm.Write(s, bytecode)
		} else {
			err = dbg.disasm.WriteBank(s, bytecode, bank)
		}

		if err != nil {
			return doNothing, err
		}

		dbg.printLine(terminal.StyleFeedback, s.String())

	case cmdGrep:
		scope := disassembly.GrepAll

		s, _ := tokens.Get()
		switch strings.ToUpper(s) {
		case "MNEMONIC":
			scope = disassembly.GrepMnemonic
		case "OPERAND":
			scope = disassembly.GrepOperand
		default:
			tokens.Unget()
		}

		search, _ := tokens.Get()
		output := strings.Builder{}
		dbg.disasm.Grep(&output, scope, search, false)
		if output.Len() == 0 {
			dbg.printLine(terminal.StyleError, "%s not found in disassembly", search)
		} else {
			dbg.printLine(terminal.StyleFeedback, output.String())
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
					dbg.disasm.Symtable.ListLocations(dbg.printStyle(terminal.StyleFeedback))

				case "READ":
					dbg.disasm.Symtable.ListReadSymbols(dbg.printStyle(terminal.StyleFeedback))

				case "WRITE":
					dbg.disasm.Symtable.ListWriteSymbols(dbg.printStyle(terminal.StyleFeedback))
				}
			} else {
				dbg.disasm.Symtable.ListSymbols(dbg.printStyle(terminal.StyleFeedback))
			}

		default:
			symbol := tok
			table, symbol, address, err := dbg.disasm.Symtable.SearchSymbol(symbol, symbols.UnspecifiedSymTable)
			if err != nil {
				if errors.Is(err, errors.SymbolUnknown) {
					dbg.printLine(terminal.StyleFeedback, "%s -> not found", symbol)
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
					dbg.printLine(terminal.StyleFeedback, "%s -> %#04x", symbol, address)

					// find all instances of symbol address in memory space
					// assumption: the address returned by SearchSymbol is the
					// first address in the complete list
					for m := address + 1; m < memorymap.OriginCart; m++ {
						ai := dbg.dbgmem.mapAddress(m, table == symbols.ReadSymTable)
						if ai.mappedAddress == address {
							dbg.printLine(terminal.StyleFeedback, "%s (%s) -> %#04x", symbol, table, m)
						}
					}
				}
			} else {
				dbg.printLine(terminal.StyleFeedback, "%s (%s) -> %#04x", symbol, table, address)
			}
		}

	case cmdOnHalt:
		if tokens.Remaining() == 0 {
			if dbg.commandOnHalt == "" {
				dbg.printLine(terminal.StyleFeedback, "auto-command on halt: OFF")
			} else {
				dbg.printLine(terminal.StyleFeedback, "auto-command on halt: %s", dbg.commandOnHalt)
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
			dbg.printLine(terminal.StyleFeedback, "auto-command on halt: OFF")
		} else {
			dbg.printLine(terminal.StyleFeedback, "auto-command on halt: %s", dbg.commandOnHalt)

			// store the new command so we can reuse it after an ONHALT OFF
			//
			// !!TODO: normalise case of specified command sequence
			dbg.commandOnHaltStored = newOnHalt
		}

		return doNothing, nil

	case cmdOnStep:
		if tokens.Remaining() == 0 {
			if dbg.commandOnStep == "" {
				dbg.printLine(terminal.StyleFeedback, "auto-command on step: OFF")
			} else {
				dbg.printLine(terminal.StyleFeedback, "auto-command on step: %s", dbg.commandOnStep)
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
			dbg.printLine(terminal.StyleFeedback, "auto-command on step: OFF")
		} else {
			dbg.printLine(terminal.StyleFeedback, "auto-command on step: %s", dbg.commandOnStep)

			// store the new command so we can reuse it after an ONSTEP OFF
			// !!TODO: normalise case of specified command sequence
			dbg.commandOnStepStored = newOnStep
		}

		return doNothing, nil

	case cmdLast:
		s := strings.Builder{}

		d, err := dbg.disasm.FormatResult(dbg.vcs.CPU.LastResult)
		if err != nil {
			return doNothing, err
		}

		option, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(option) {
			case "DEFN":
				if dbg.vcs.CPU.LastResult.Defn == nil {
					dbg.printLine(terminal.StyleFeedback, "no instruction decoded yet")
				} else {
					dbg.printLine(terminal.StyleFeedback, "%s", dbg.vcs.CPU.LastResult.Defn)
				}
				return doNothing, nil

			case "BYTECODE":
				s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Bytecode, d.Bytecode))
			}
		}

		s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Address, d.Address))
		s.WriteString(" ")
		s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Mnemonic, d.Mnemonic))
		s.WriteString(" ")
		s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Operand, d.Operand))
		s.WriteString(" ")
		s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Cycles, d.Cycles))
		s.WriteString(" ")
		s.WriteString(fmt.Sprintf(dbg.disasm.Columns.Fmt.Notes, d.Notes))

		if dbg.vcs.CPU.LastResult.Final {
			dbg.printLine(terminal.StyleCPUStep, s.String())
		} else {
			dbg.printLine(terminal.StyleVideoStep, s.String())
		}

	case cmdMemMap:
		dbg.printLine(terminal.StyleInstrument, "%v", memorymap.Summary())

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
						dbg.printLine(terminal.StyleError, "value must be a positive 16 number")
					}

					dbg.vcs.CPU.PC.Load(uint16(v))
				} else {
					// 6507 registers are 8 bit
					v, err := strconv.ParseUint(value, 0, 8)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value must be a positive 8 number")
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
				dbg.printLine(terminal.StyleError, "%s", err)
			} else {
				dbg.printLine(terminal.StyleInstrument, ai.String())
			}

			// loop through all addresses
			a, present = tokens.Get()
		}

	case cmdPoke:
		// get address token
		a, _ := tokens.Get()

		// convert address
		ai := dbg.dbgmem.mapAddress(a, false)
		if ai == nil {
			// using poke error because hexload is basically the same as poking
			dbg.printLine(terminal.StyleError, errors.New(errors.UnpokeableAddress, a).Error())
			return doNothing, nil
		}
		addr := ai.mappedAddress

		// get (first) value token
		v, present := tokens.Get()

		for present {
			val, err := strconv.ParseUint(v, 0, 8)
			if err != nil {
				dbg.printLine(terminal.StyleError, "hexload value must be an 8 bit number (%s)", v)
				v, present = tokens.Get()
				continue // for loop (without advancing address)
			}

			// perform individual poke
			ai, err := dbg.dbgmem.poke(addr, uint8(val))
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			} else {
				dbg.printLine(terminal.StyleInstrument, ai.String())
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
					dbg.printLine(terminal.StyleInstrument, fmt.Sprintf("%v", dbg.vcs.Mem.Cart.RAM()))
				} else {
					dbg.printLine(terminal.StyleFeedback, "cartridge does not contain any additional RAM")
				}

			}
		} else {
			dbg.printInstrument(dbg.vcs.Mem.RAM)
		}

	case cmdTimer:
		dbg.printInstrument(dbg.vcs.RIOT.Timer)

	case cmdTIA:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "DELAYS":
				// for convience asking for TIA delays also prints delays for
				// the sprites
				dbg.printInstrument(dbg.vcs.TIA.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Player0.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Player1.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Missile0.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Missile1.Delay)
				dbg.printInstrument(dbg.vcs.TIA.Video.Ball.Delay)
			}
		} else {
			dbg.printInstrument(dbg.vcs.TIA)
		}

	case cmdAudio:
		dbg.printInstrument(dbg.vcs.TIA.Audio)

	case cmdTV:
		option, present := tokens.Get()
		if present {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				dbg.printLine(terminal.StyleInstrument, dbg.tv.GetSpec().ID)
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printInstrument(dbg.tv)
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

		case "MASK":
			err = dbg.scr.SetFeature(gui.ReqSetMasking, false)
			if err != nil {
				return doNothing, err
			}

		case "UNMASK":
			err = dbg.scr.SetFeature(gui.ReqSetMasking, true)
			if err != nil {
				return doNothing, err
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

	case cmdPanel:
		mode, _ := tokens.Get()
		switch strings.ToUpper(mode) {
		case "TOGGLE":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0":
				dbg.vcs.Panel.Handle(input.PanelTogglePlayer0Pro)
			case "P1":
				dbg.vcs.Panel.Handle(input.PanelTogglePlayer1Pro)
			case "COL":
				dbg.vcs.Panel.Handle(input.PanelToggleColor)
			}
		case "SET":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0PRO":
				dbg.vcs.Panel.Handle(input.PanelSetPlayer0Pro)
			case "P1PRO":
				dbg.vcs.Panel.Handle(input.PanelSetPlayer1Pro)
			case "P0AM":
				dbg.vcs.Panel.Handle(input.PanelSetPlayer0Am)
			case "P1AM":
				dbg.vcs.Panel.Handle(input.PanelSetPlayer1Am)
			case "COL":
				dbg.vcs.Panel.Handle(input.PanelSetColor)
			case "BW":
				dbg.vcs.Panel.Handle(input.PanelSetBlackAndWhite)
			}
		}
		dbg.printInstrument(dbg.vcs.Panel)

	case cmdStick:
		var err error

		stick, _ := tokens.Get()
		action, _ := tokens.Get()

		var event input.Event
		switch strings.ToUpper(action) {
		case "UP":
			event = input.Up
		case "DOWN":
			event = input.Down
		case "LEFT":
			event = input.Left
		case "RIGHT":
			event = input.Right
		case "NOUP":
			event = input.NoUp
		case "NODOWN":
			event = input.NoDown
		case "NOLEFT":
			event = input.NoLeft
		case "NORIGHT":
			event = input.NoRight
		case "FIRE":
			event = input.Fire
		case "NOFIRE":
			event = input.NoFire
		}

		n, _ := strconv.Atoi(stick)
		switch n {
		case 0:
			err = dbg.vcs.HandController0.Handle(event)
		case 1:
			err = dbg.vcs.HandController1.Handle(event)
		}

		if err != nil {
			return doNothing, err
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
		err := dbg.watches.parseWatch(tokens)
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
			dbg.breakpoints.list()
			dbg.traps.list()
			dbg.watches.list()
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
			dbg.printLine(terminal.StyleFeedback, "breakpoint #%d dropped", num)
		case "TRAP":
			err := dbg.traps.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.printLine(terminal.StyleFeedback, "trap #%d dropped", num)
		case "WATCH":
			err := dbg.watches.drop(num)
			if err != nil {
				return doNothing, err
			}
			dbg.printLine(terminal.StyleFeedback, "watch #%d dropped", num)
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdClear:
		clear, _ := tokens.Get()
		clear = strings.ToUpper(clear)
		switch clear {
		case "BREAKS":
			dbg.breakpoints.clear()
			dbg.printLine(terminal.StyleFeedback, "breakpoints cleared")
		case "TRAPS":
			dbg.traps.clear()
			dbg.printLine(terminal.StyleFeedback, "traps cleared")
		case "WATCHES":
			dbg.watches.clear()
			dbg.printLine(terminal.StyleFeedback, "watches cleared")
		case "ALL":
			dbg.breakpoints.clear()
			dbg.traps.clear()
			dbg.watches.clear()
			dbg.printLine(terminal.StyleFeedback, "breakpoints, traps and watches cleared")
		default:
			// already caught by command line ValidateTokens()
		}

	}

	return doNothing, nil
}
