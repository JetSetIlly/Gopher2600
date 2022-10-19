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

package debugger

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/controllers"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/patch"
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
		panic(err)
	}

	err = debuggerCommands.AddHelp(cmdHelp, helps)
	if err != nil {
		panic(err)
	}
	sort.Stable(debuggerCommands)

	scriptUnsafeCommands, err = commandline.ParseCommandTemplate(scriptUnsafeTemplate)
	if err != nil {
		panic(err)
	}
	sort.Stable(scriptUnsafeCommands)
}

// parseCommand tokenises the input and processes the tokens.
func (dbg *Debugger) parseCommand(cmd string, scribe bool, echo bool) error {
	tokens, err := dbg.tokeniseCommand(cmd, scribe, echo)
	if err != nil {
		return err
	}
	return dbg.processTokens(tokens)
}

// return tokenised command.
func (dbg *Debugger) tokeniseCommand(cmd string, scribe bool, echo bool) (*commandline.Tokens, error) {
	// tokenise input
	tokens := commandline.TokeniseInput(cmd)

	// if there are no tokens in the input then continue with a default action
	if tokens.Remaining() == 0 {
		return dbg.tokeniseCommand("STEP", true, false)
	}

	// check validity of tokenised input
	err := debuggerCommands.ValidateTokens(tokens)
	if err != nil {
		return nil, err
	}

	// print normalised input if this is command from an interactive source
	// and not an auto-command
	if echo {
		dbg.printLine(terminal.StyleEcho, tokens.String())
	}

	// test to see if command is allowed when recording/playing a script
	if dbg.scriptScribe.IsActive() && scribe {
		tokens.Reset()

		err := scriptUnsafeCommands.ValidateTokens(tokens)

		// fail when the tokens DO match the scriptUnsafe template (ie. when
		// there is no err from the validate function)
		if err == nil {
			return nil, curated.Errorf("'%s' is unsafe to use in scripts", tokens.String())
		}

		// record command if it auto is false (is not a result of an "auto" command
		// eg. ONHALT). if there's an error then the script will be rolled back and
		// the write removed.
		dbg.scriptScribe.WriteInput(tokens.String())
	}

	return tokens, nil
}

// processTokensList call processTokens for each entry in the array of tokens.
// this is useful when we have already parsed and tokenised command input and
// simply want to rerun those commands.
//
// used by the ONSTEP, ONHALT and ONTRACE features.
func (dbg *Debugger) processTokensList(tokenGrp []*commandline.Tokens) error {
	var err error

	for _, t := range tokenGrp {
		err = dbg.processTokens(t)
		if err != nil {
			return err
		}
	}
	return nil
}

// process a single command (with arguments).
func (dbg *Debugger) processTokens(tokens *commandline.Tokens) error {
	// check first token. if this token makes sense then we will consume the
	// rest of the tokens appropriately
	tokens.Reset()
	command, _ := tokens.Get()

	switch command {
	default:
		return curated.Errorf("%s is not yet implemented", command)

	case cmdHelp:
		keyword, ok := tokens.Get()
		if ok {
			dbg.printLine(terminal.StyleHelp, debuggerCommands.Help(keyword))
		} else {
			dbg.printLine(terminal.StyleHelp, debuggerCommands.HelpOverview())
		}

		// we don't want the HELP command to appear in the script
		dbg.scriptScribe.Rollback()

		return nil

	case cmdQuit:
		if dbg.scriptScribe.IsActive() {
			dbg.printLine(terminal.StyleFeedback, "ending script recording")

			// we don't want the QUIT command to appear in the script
			dbg.scriptScribe.Rollback()

			return dbg.scriptScribe.EndSession()
		} else {
			dbg.running = false
		}

	case cmdReset:
		// resetting in the middle of a CPU instruction requires the input loop
		// to be unwound before continuing
		dbg.unwindLoop(func() error {
			// don't reset breakpoints, etc.
			err := dbg.reset(false)
			if err != nil {
				return err
			}
			dbg.printLine(terminal.StyleFeedback, "machine reset")
			return nil
		})

	case cmdRun:
		dbg.runUntilHalt = true
		dbg.continueEmulation = true
		return nil

	case cmdHalt:
		dbg.haltImmediately = true

	case cmdStep:
		adjAmount := 1
		back := false

		if tk, ok := tokens.Get(); ok {
			switch tk {
			case "BACK":
				back = true
				adjAmount *= -1

			case "OVER":
				// if next expected opcode is JSR then add a volatile breakpoint to the
				// return address
				e := dbg.Disasm.GetEntryByAddress(dbg.vcs.CPU.PC.Address())
				if e.Operator == "jsr" {
					brk := commandline.TokeniseInput(fmt.Sprintf("%#4x", dbg.vcs.CPU.PC.Address()+3))
					dbg.halting.volatileBreakpoints.parseCommand(brk)

					// breakpoing will take many cycles to trigger so we need to run the emulation
					dbg.runUntilHalt = true
				}

			default:
				tokens.Unget()
			}
		}

		// get mode
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)

		if back {
			var instruction bool
			var adj television.Adj

			switch mode {
			case "":
				// continue with current quantum state
				if dbg.stepQuantum == QuantumInstruction {
					instruction = true
				} else {
					adj = television.AdjClock
				}
			case "INSTRUCTION":
				dbg.stepQuantum = QuantumInstruction
				instruction = true
			case "CLOCK":
				dbg.stepQuantum = QuantumClock
				adj = television.AdjClock
			case "SCANLINE":
				adj = television.AdjScanline
			case "FRAME":
				adj = television.AdjFrame
			default:
				return curated.Errorf("unknown STEP BACK mode (%s)", mode)
			}

			var coords coords.TelevisionCoords

			if instruction {
				coords = dbg.cpuBoundaryLastInstruction
			} else {
				coords = dbg.vcs.TV.AdjCoords(adj, adjAmount)
			}

			dbg.setState(govern.Rewinding)
			dbg.unwindLoop(func() error {
				// update catchupQuantum before starting rewind process
				dbg.catchupQuantum = dbg.stepQuantum

				return dbg.Rewind.GotoCoords(coords)
			})

			return nil
		}

		// step forward
		switch mode {
		case "":
			// continue with current quantum state

			// if quantum is instruction and CPU is not RDY then STEP is best
			// implemented as TRAP RDY
			if dbg.stepQuantum == QuantumInstruction && !dbg.vcs.CPU.RdyFlg {
				_ = dbg.halting.volatileTraps.parseCommand(commandline.TokeniseInput("RDY"))
				dbg.runUntilHalt = true

				// when the RDY flag changes the input loop will think it's
				// inside a video step. we need to force the loop to return
				// to the non-video step loop
				dbg.stepOutOfVideoStepInputLoop = true
			}
		case "INSTRUCTION":
			dbg.stepQuantum = QuantumInstruction
		case "CLOCK":
			dbg.stepQuantum = QuantumClock
		default:
			// do not change quantum
			tokens.Unget()

			// ignoring error
			_ = dbg.halting.volatileTraps.parseCommand(tokens)

			// trap may take many cycles to trigger
			dbg.runUntilHalt = true
		}

		// always continue
		dbg.continueEmulation = true

	case cmdQuantum:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "INSTRUCTION":
			dbg.stepQuantum = QuantumInstruction
		case "CLOCK":
			dbg.stepQuantum = QuantumClock
		default:
			dbg.printLine(terminal.StyleFeedback, "set to %s", dbg.stepQuantum)
		}

	case cmdScript:
		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "RECORD":
			var err error
			saveFile, _ := tokens.Get()
			err = dbg.scriptScribe.StartSession(saveFile)
			if err != nil {
				return err
			}

			// we don't want SCRIPT RECORD command to appear in the script
			dbg.scriptScribe.Rollback()

			return nil

		case "END":
			// we don't want SCRIPT END command to appear in the script
			dbg.scriptScribe.Rollback()

			return dbg.scriptScribe.EndSession()

		default:
			// run a script
			scr, err := script.RescribeScript(option)
			if err != nil {
				return err
			}

			if dbg.scriptScribe.IsActive() {
				// if we're currently recording a script we want to write this
				// command to the new script file but indicate that we'll be
				// entering a new script and so don't want to repeat the
				// commands from that script
				err := dbg.scriptScribe.StartPlayback()
				if err != nil {
					return err
				}

				defer dbg.scriptScribe.EndPlayback()
			}

			err = dbg.inputLoop(scr, false)
			if err != nil {
				return err
			}
		}

	case cmdRewind:
		// note that we calling the rewind.GotoFrame() functions directly and not
		// using the debugger.PushRewind() function.
		arg, ok := tokens.Get()
		if ok {
			// stop emulation on rewind
			dbg.runUntilHalt = false

			if arg == "LAST" {
				dbg.setState(govern.Rewinding)
				dbg.unwindLoop(dbg.Rewind.GotoLast)
			} else if arg == "SUMMARY" {
				dbg.printLine(terminal.StyleInstrument, dbg.Rewind.String())
			} else {
				frame, _ := strconv.Atoi(arg)
				dbg.setState(govern.Rewinding)
				dbg.unwindLoop(func() error {
					err := dbg.Rewind.GotoFrame(frame)
					if err != nil {
						return err
					}
					return nil
				})
			}
			return nil
		}

	case cmdGoto:
		coords := dbg.vcs.TV.GetCoords()

		if s, ok := tokens.Get(); ok {
			coords.Clock, _ = strconv.Atoi(s)
			if s, ok := tokens.Get(); ok {
				coords.Scanline, _ = strconv.Atoi(s)
				if s, ok := tokens.Get(); ok {
					coords.Frame, _ = strconv.Atoi(s)
				}
			}
		}

		dbg.setState(govern.Rewinding)
		dbg.unwindLoop(func() error {
			err := dbg.Rewind.GotoCoords(coords)
			if err != nil {
				return err
			}
			return nil
		})

	case cmdInsert:
		dbg.unwindLoop(func() error {
			cart, _ := tokens.Get()
			cl, err := cartridgeloader.NewLoader(cart, "AUTO")
			if err != nil {
				return err
			}
			err = dbg.attachCartridge(cl)
			if err != nil {
				return err
			}

			// use cartridge's idea of the filename. the attach process may have
			// caused a different cartridge to load than the one requested (most
			// typically this will mean that the cartridge has been ejected)
			dbg.printLine(terminal.StyleFeedback, "machine reset with new cartridge (%s)", dbg.vcs.Mem.Cart.Filename)

			return nil
		})

	case cmdCartridge:
		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "PATH":
				dbg.printLine(
					terminal.StyleInstrument,
					dbg.vcs.Mem.Cart.Filename,
				)

			case "NAME":
				dbg.printLine(
					terminal.StyleInstrument,
					dbg.vcs.Mem.Cart.ShortName,
				)

			case "MAPPER":
				dbg.printLine(
					terminal.StyleInstrument,
					dbg.vcs.Mem.Cart.ID(),
				)

			case "CONTAINER":
				dbg.printLine(
					terminal.StyleInstrument,
					dbg.vcs.Mem.Cart.ContainerID(),
				)

			case "MAPPEDBANKS":
				dbg.printLine(
					terminal.StyleInstrument,
					dbg.vcs.Mem.Cart.MappedBanks(),
				)

			case "HASH":
				dbg.printLine(
					terminal.StyleFeedback,
					dbg.vcs.Mem.Cart.Hash,
				)

			case "STATIC":
				// !!TODO: poke/peek static cartridge static data areas
				if bus := dbg.vcs.Mem.Cart.GetStaticBus(); bus != nil {
					static := bus.GetStatic()
					if static != nil {
						dbg.printLine(terminal.StyleFeedback, "cartridge has a static data area")
					} else {
						dbg.printLine(terminal.StyleFeedback, "cartridge has no static data area")
					}
				} else {
					dbg.printLine(terminal.StyleFeedback, "cartridge has no static data area")
				}
			case "REGISTERS":
				// !!TODO: poke/peek cartridge registers
				if bus := dbg.vcs.Mem.Cart.GetRegistersBus(); bus != nil {
					dbg.printLine(terminal.StyleInstrument, bus.GetRegisters().String())
				} else {
					dbg.printLine(terminal.StyleFeedback, "cartridge has no registers")
				}

			case "RAM":
				// cartridge RAM is accessible through the normal VCS buses so
				// the normal peek/poke commands will work
				if bus := dbg.vcs.Mem.Cart.GetRAMbus(); bus != nil {
					ram := bus.GetRAM()
					if ram != nil {
						s := &strings.Builder{}
						for b := 0; b < len(ram); b++ {
							s.WriteString(ram[b].Label + "\n")
							s.WriteString(strings.Repeat("-", len(ram[b].Label)))
							s.WriteString("\n")
							s.WriteString(hex.Dump(ram[b].Data))
							s.WriteString("\n\n")
						}

						dbg.printLine(terminal.StyleInstrument, s.String())
					} else {
						dbg.printLine(terminal.StyleFeedback, "cartridge has no RAM")
					}
				} else {
					dbg.printLine(terminal.StyleFeedback, "cartridge has no RAM")
				}

			case "HOTLOAD":
				err := dbg.hotload()
				if err != nil {
					dbg.printLine(terminal.StyleFeedback, err.Error())
				} else {
					dbg.printLine(terminal.StyleFeedback, "hotload successful")
				}

			case "DUMP":
				romdump, err := dbg.vcs.Mem.Cart.ROMDump()
				if err != nil {
					dbg.printLine(terminal.StyleFeedback, err.Error())
				} else {
					dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("rom dumped to %s", romdump))
				}
			}
		} else {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.Mem.Cart.String())
		}

	case cmdPatch:
		f, _ := tokens.Get()
		patched, err := patch.CartridgeMemory(dbg.vcs.Mem.Cart, f)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%v", err)
			if patched {
				dbg.printLine(terminal.StyleError, "error during patching. cartridge might be unusable.")
			}
			return nil
		}
		if patched {
			dbg.printLine(terminal.StyleFeedback, "cartridge patched")
		}

	case cmdDisasm:
		bytecode := false

		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "REDUX":
				err := dbg.Disasm.FromMemory()
				if err != nil {
					dbg.printLine(terminal.StyleFeedback, err.Error())
				}
				return nil
			case "BYTECODE":
				bytecode = true
			}
		}

		attr := disassembly.ColumnAttr{
			ByteCode: bytecode,
			Label:    true,
			Cycles:   true,
		}

		s := strings.Builder{}
		err := dbg.Disasm.Write(&s, attr)
		if err != nil {
			dbg.printLine(terminal.StyleFeedback, err.Error())
		}

		dbg.printLine(terminal.StyleFeedback, s.String())

	case cmdGrep:
		scope := disassembly.GrepAll

		s, _ := tokens.Get()
		switch strings.ToUpper(s) {
		case "OPERATOR":
			scope = disassembly.GrepOperator
		case "OPERAND":
			scope = disassembly.GrepOperand
		default:
			tokens.Unget()
		}

		search, _ := tokens.Get()
		output := &strings.Builder{}
		err := dbg.Disasm.Grep(output, scope, search, false)
		if err != nil {
			return err
		}
		if output.Len() == 0 {
			dbg.printLine(terminal.StyleError, "%s not found in disassembly", search)
		} else {
			dbg.printLine(terminal.StyleFeedback, output.String())
		}

	case cmdSymbol:
		tok, _ := tokens.Get()
		switch strings.ToUpper(tok) {
		case "LIST":
			option, ok := tokens.Get()
			if ok {
				switch strings.ToUpper(option) {
				default:
					// already caught by command line ValidateTokens()

				case "LABELS":
					dbg.dbgmem.Sym.ListLabels(dbg.printStyle(terminal.StyleFeedback))

				case "READ":
					dbg.dbgmem.Sym.ListReadSymbols(dbg.printStyle(terminal.StyleFeedback))

				case "WRITE":
					dbg.dbgmem.Sym.ListWriteSymbols(dbg.printStyle(terminal.StyleFeedback))
				}
			} else {
				dbg.dbgmem.Sym.ListSymbols(dbg.printStyle(terminal.StyleFeedback))
			}

		default:
			symbol := tok

			symSearch := dbg.Disasm.Sym.SearchBySymbol(symbol, symbols.SearchLabel)
			if symSearch != nil {
				ai := dbg.dbgmem.GetAddressInfo(symSearch.Address, true)
				if ai != nil {
					dbg.printLine(terminal.StyleFeedback, "%s [LABEL]", ai.String())
				} else {
					symSearch = nil
				}
			}

			aiRead := dbg.dbgmem.GetAddressInfo(symbol, true)
			if aiRead != nil {
				dbg.printLine(terminal.StyleFeedback, "%s [READ]", aiRead.String())
			}

			aiWrite := dbg.dbgmem.GetAddressInfo(symbol, false)
			if aiWrite != nil {
				dbg.printLine(terminal.StyleFeedback, "%s [WRITE]", aiWrite.String())
			}

			if symSearch == nil && aiRead == nil && aiWrite == nil {
				dbg.printLine(terminal.StyleFeedback, "%s not found in any symbol table", symbol)
			}
		}

	case cmdOnHalt:
		if tokens.Remaining() == 0 {
			if len(dbg.commandOnHalt) == 0 {
				dbg.printLine(terminal.StyleFeedback, "auto-command on halt: OFF")
			} else {
				s := strings.Builder{}
				for _, c := range dbg.commandOnHalt {
					s.WriteString(c.String())
					s.WriteString("; ")
				}
				dbg.printLine(terminal.StyleFeedback, "command on halt: %s", strings.TrimSuffix(s.String(), "; "))
			}
			return nil
		}

		var input string

		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			dbg.commandOnHalt = dbg.commandOnHalt[:0]
			dbg.printLine(terminal.StyleFeedback, "no command on halt")
			return nil

		case "ON":
			dbg.commandOnHalt = dbg.commandOnHaltStored
			for _, c := range dbg.commandOnHalt {
				dbg.printLine(terminal.StyleFeedback, "auto-command on halt: %s", c)
			}
			return nil

		default:
			// token isn't one we recognise so push it back onto the token queue
			tokens.Unget()

			// use remaininder of command line to form the ONHALT command sequence
			input = strings.TrimSpace(tokens.Remainder())
			tokens.End()
		}

		// empty list of tokens. taking note of existing command - not the same
		// as commandOnHaltStored because ONHALT might be OFF
		existingOnHalt := dbg.commandOnHalt
		dbg.commandOnHalt = dbg.commandOnHalt[:0]

		// tokenise commands to check for integrity
		for _, s := range strings.Split(input, ",") {
			toks, err := dbg.tokeniseCommand(s, false, false)
			if err != nil {
				dbg.commandOnHalt = existingOnHalt
				return err
			}
			dbg.commandOnHalt = append(dbg.commandOnHalt, toks)
		}

		// make a copy of
		dbg.commandOnHaltStored = dbg.commandOnHalt

		// display the new ONHALT command(s)
		s := strings.Builder{}
		for _, c := range dbg.commandOnHalt {
			s.WriteString(c.String())
			s.WriteString("; ")
		}
		dbg.printLine(terminal.StyleFeedback, "command on halt: %s", strings.TrimSuffix(s.String(), "; "))

		return nil

	case cmdOnStep:
		if tokens.Remaining() == 0 {
			if len(dbg.commandOnStep) == 0 {
				dbg.printLine(terminal.StyleFeedback, "no command on step")
			} else {
				s := strings.Builder{}
				for _, c := range dbg.commandOnStep {
					s.WriteString(c.String())
					s.WriteString("; ")
				}
				dbg.printLine(terminal.StyleFeedback, "command on step: %s", strings.TrimSuffix(s.String(), "; "))
			}
			return nil
		}

		var input string

		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			dbg.commandOnStep = dbg.commandOnStep[:0]
			dbg.printLine(terminal.StyleFeedback, "auto-command on step: OFF")
			return nil

		case "ON":
			dbg.commandOnStep = dbg.commandOnStepStored
			for _, c := range dbg.commandOnStep {
				dbg.printLine(terminal.StyleFeedback, "auto-command on step: %s", c)
			}
			return nil

		default:
			// token isn't one we recognise so push it back onto the token queue
			tokens.Unget()

			// use remaininder of command line to form the ONSTEP command sequence
			input = strings.TrimSpace(tokens.Remainder())
			tokens.End()
		}

		// empty list of tokens. taking note of existing command - not the same
		// as commandOnStepStored because ONSTEP might be OFF
		existingOnStep := dbg.commandOnStep
		dbg.commandOnStep = dbg.commandOnStep[:0]

		// tokenise commands to check for integrity
		for _, s := range strings.Split(input, ",") {
			toks, err := dbg.tokeniseCommand(s, false, false)
			if err != nil {
				dbg.commandOnStep = existingOnStep
				return err
			}
			dbg.commandOnStep = append(dbg.commandOnStep, toks)
		}

		// store new commandOnStep
		dbg.commandOnStepStored = dbg.commandOnStep

		// display the new ONSTEP command(s)
		s := strings.Builder{}
		for _, c := range dbg.commandOnStep {
			s.WriteString(c.String())
			s.WriteString("; ")
		}
		dbg.printLine(terminal.StyleFeedback, "command on step: %s", strings.TrimSuffix(s.String(), "; "))

		return nil

	case cmdOnTrace:
		if tokens.Remaining() == 0 {
			if len(dbg.commandOnTrace) == 0 {
				dbg.printLine(terminal.StyleFeedback, "no command on trace")
			} else {
				s := strings.Builder{}
				for _, c := range dbg.commandOnTrace {
					s.WriteString(c.String())
					s.WriteString("; ")
				}
				dbg.printLine(terminal.StyleFeedback, "command on trace: %s", strings.TrimSuffix(s.String(), "; "))
			}
			return nil
		}

		var input string

		option, _ := tokens.Get()
		switch strings.ToUpper(option) {
		case "OFF":
			dbg.commandOnTrace = dbg.commandOnTrace[:0]
			dbg.printLine(terminal.StyleFeedback, "auto-command on trace: OFF")
			return nil

		case "ON":
			dbg.commandOnTrace = dbg.commandOnTraceStored
			for _, c := range dbg.commandOnTrace {
				dbg.printLine(terminal.StyleFeedback, "auto-command on trace: %s", c)
			}
			return nil

		default:
			// token isn't one we recognise so push it back onto the token queue
			tokens.Unget()

			// use remaininder of command line to form the ONTRACE command sequence
			input = strings.TrimSpace(tokens.Remainder())
			tokens.End()
		}

		// empty list of tokens. taking note of existing command
		existingOnTrace := dbg.commandOnTrace
		dbg.commandOnTrace = dbg.commandOnTrace[:0]

		// tokenise commands to check for integrity
		for _, s := range strings.Split(input, ",") {
			toks, err := dbg.tokeniseCommand(s, false, false)
			if err != nil {
				dbg.commandOnTrace = existingOnTrace
				return err
			}
			dbg.commandOnTrace = append(dbg.commandOnTrace, toks)
		}

		// store new commandOnTrace
		dbg.commandOnTraceStored = dbg.commandOnTrace

		// display the new ONTRACE command(s)
		s := strings.Builder{}
		for _, c := range dbg.commandOnTrace {
			s.WriteString(c.String())
			s.WriteString("; ")
		}
		dbg.printLine(terminal.StyleFeedback, "command on trace: %s", strings.TrimSuffix(s.String(), "; "))

		return nil

	case cmdLast:
		if dbg.liveDisasmEntry == nil || dbg.liveDisasmEntry.Result.Defn == nil {
			dbg.printLine(terminal.StyleFeedback, "no instruction decoded yet")
			return nil
		}

		// whether to show bytecode
		bytecode := false

		option, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(option) {
			case "DEFN":
				if dbg.vcs.CPU.LastResult.Defn == nil {
					dbg.printLine(terminal.StyleFeedback, "no instruction decoded yet")
				} else {
					dbg.printLine(terminal.StyleFeedback, "%s", dbg.vcs.CPU.LastResult.Defn)
				}
				return nil

			case "BYTECODE":
				bytecode = true
			}
		}

		s := strings.Builder{}

		if dbg.vcs.Mem.Cart.NumBanks() > 1 {
			s.WriteString(fmt.Sprintf("[%s] ", dbg.liveBankInfo))
		}
		s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldAddress))
		s.WriteString(" ")
		if bytecode {
			s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldBytecode))
			s.WriteString(" ")
		}
		s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldOperator))
		s.WriteString(" ")
		s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldOperand))
		s.WriteString(" ")
		s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldCycles))
		s.WriteString(" ")
		s.WriteString(dbg.liveDisasmEntry.GetField(disassembly.FldNotes))

		// change terminal output style depending on condition of last CPU result
		if dbg.liveDisasmEntry.Result.Final {
			dbg.printLine(terminal.StyleCPUStep, s.String())
		} else {
			dbg.printLine(terminal.StyleVideoStep, s.String())
		}

	case cmdMemMap:
		address, ok := tokens.Get()
		if ok {
			// if an address argument has been specified then map the address
			// in a read and write context and display the information

			// if hasMapped is false after the read/write mappings then the
			// address could no be resolved and we print an appropriate notice
			// to the user
			hasMapped := false

			s := strings.Builder{}

			ai := dbg.dbgmem.GetAddressInfo(address, true)
			if ai != nil {
				hasMapped = true
				s.WriteString("Read:\n")
				if ai.Address != ai.MappedAddress {
					s.WriteString(fmt.Sprintf("  %#04x maps to %#04x ", ai.Address, ai.MappedAddress))
				} else {
					s.WriteString(fmt.Sprintf("  %#04x ", ai.Address))
				}
				s.WriteString(fmt.Sprintf("in area %s\n", ai.Area.String()))
				if ai.Symbol != "" {
					s.WriteString(fmt.Sprintf("  labelled as %s\n", ai.Symbol))
				}
			}
			ai = dbg.dbgmem.GetAddressInfo(address, false)
			if ai != nil {
				hasMapped = true
				s.WriteString("Write:\n")
				if ai.Address != ai.MappedAddress {
					s.WriteString(fmt.Sprintf("  %#04x maps to %#04x ", ai.Address, ai.MappedAddress))
				} else {
					s.WriteString(fmt.Sprintf("  %#04x ", ai.Address))
				}
				s.WriteString(fmt.Sprintf("in area %s\n", ai.Area.String()))
				if ai.Symbol != "" {
					s.WriteString(fmt.Sprintf("  labelled as %s\n", ai.Symbol))
				}
			}

			// print results
			if hasMapped {
				dbg.printLine(terminal.StyleInstrument, "%s", s.String())
			} else {
				dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("%v is not a mappable address", address))
			}
		} else {
			// without an address argument print the memorymap summary table
			dbg.printLine(terminal.StyleInstrument, "%v", memorymap.Summary())
		}

	case cmdCPU:
		action, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(action) {
			case "STATUS":
				action, ok = tokens.Get()
				if ok {
					target, _ := tokens.Get()
					var targetVal *bool
					switch target {
					case "S":
						targetVal = &dbg.vcs.CPU.Status.Sign
					case "O":
						targetVal = &dbg.vcs.CPU.Status.Overflow
					case "B":
						targetVal = &dbg.vcs.CPU.Status.Break
					case "D":
						targetVal = &dbg.vcs.CPU.Status.DecimalMode
					case "I":
						targetVal = &dbg.vcs.CPU.Status.InterruptDisable
					case "Z":
						targetVal = &dbg.vcs.CPU.Status.Zero
					case "C":
						targetVal = &dbg.vcs.CPU.Status.Carry
					}

					switch action {
					case "SET":
						*targetVal = true
					case "UNSET":
						*targetVal = false
					case "TOGGLE":
						*targetVal = !*targetVal
					}
				} else {
					dbg.printLine(terminal.StyleInstrument, dbg.vcs.CPU.Status.String())
				}

			case "SET":
				target, _ := tokens.Get()
				value, _ := tokens.Get()

				target = strings.ToUpper(target)
				if target == "PC" {
					// program counter is a 16 bit number
					v, err := strconv.ParseUint(value, 16, 16)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value must be a positive 16 bit number")
					}

					dbg.vcs.CPU.PC.Load(uint16(v))
				} else {
					// 6507 registers are 8 bit
					v, err := strconv.ParseUint(value, 16, 8)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value must be a positive 8 bit number")
					}

					var reg *registers.Register
					switch strings.ToUpper(target) {
					case "A":
						reg = &dbg.vcs.CPU.A
					case "X":
						reg = &dbg.vcs.CPU.X
					case "Y":
						reg = &dbg.vcs.CPU.Y
					case "SP":
						reg = &dbg.vcs.CPU.SP
					}

					reg.Load(uint8(v))
				}

			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.CPU.String())
		}

	case cmdPeek:
		// get first address token
		a, ok := tokens.Get()

		for ok {
			// perform peek
			ai, err := dbg.dbgmem.Peek(a)
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			} else {
				dbg.printLine(terminal.StyleInstrument, ai.String())
			}

			// loop through all addresses
			a, ok = tokens.Get()
		}

	case cmdPoke:
		// get address token
		a, _ := tokens.Get()

		// convert address. note that the calls to dbgmem.poke() also call
		// MapAddress(). the reason we map the address here is because we want
		// a numeric address that we can iterate with in the for loop below.
		// simply converting to a number is no good because we want the user to
		// be able to specify an address by name, so we may as well just call
		// MapAddress(), even if it does seem redundant
		//
		// see comment in DbgMem.Poke() for why we treat the address as a
		// "read" address
		ai := dbg.dbgmem.GetAddressInfo(a, true)
		if ai == nil {
			dbg.printLine(terminal.StyleError, fmt.Sprintf(dbgmem.PokeError, a))
			return nil
		}
		addr := ai.MappedAddress

		// get (first) value token
		v, ok := tokens.Get()

		for ok {
			val, err := strconv.ParseUint(v, 0, 8)
			if err != nil {
				dbg.printLine(terminal.StyleError, "value must be an 8 bit number (%s)", v)
				v, ok = tokens.Get()
				continue // for loop (without advancing address)
			}

			ai, err := dbg.dbgmem.Poke(addr, uint8(val))
			if err != nil {
				dbg.printLine(terminal.StyleError, "%s", err)
			} else {
				dbg.printLine(terminal.StyleInstrument, ai.String())
			}

			// loop through all values
			v, ok = tokens.Get()
			addr++
		}

	case cmdSwap:
		// get address token
		a, _ := tokens.Get()
		b, _ := tokens.Get()

		ai, err := dbg.dbgmem.Peek(a)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		bi, err := dbg.dbgmem.Peek(b)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		if _, err := dbg.dbgmem.Poke(ai.MappedAddress, bi.Data); err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		if _, err := dbg.dbgmem.Poke(bi.MappedAddress, ai.Data); err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		aj, err := dbg.dbgmem.Peek(ai.Address)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		bj, err := dbg.dbgmem.Peek(bi.Address)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%s", err)
			return nil
		}

		dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("%04x: %02x->%02x and %04x: %02x->%02x", ai.Address, ai.Data, aj.Data, bi.Address, bi.Data, bj.Data))

	case cmdRAM:
		dbg.printLine(terminal.StyleInstrument, dbg.vcs.Mem.RAM.String())

	case cmdTIA:
		arg, _ := tokens.Get()
		switch arg {
		case "HMOVE":
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Hmove.String())
		default:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.String())
		}

	case cmdRIOT:
		arg, _ := tokens.Get()
		switch arg {
		case "TIMER":
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.RIOT.Timer.String())
		case "PORTS":
			fallthrough
		default:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.RIOT.Ports.String())
		}

	case cmdAudio:
		dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Audio.String())

	case cmdTV:
		option, ok := tokens.Get()
		if ok {
			option = strings.ToUpper(option)
			switch option {
			case "SPEC":
				newspec, ok := tokens.Get()
				if ok {
					// unknown specifciations already handled by ValidateTokens()
					err := dbg.vcs.TV.SetSpec(newspec)
					if err != nil {
						return err
					}
				}

				spec := dbg.vcs.TV.GetFrameInfo().Spec
				s := strings.Builder{}
				s.WriteString(spec.ID)
				dbg.printLine(terminal.StyleInstrument, s.String())
			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TV.String())
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
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Player0.String())

		case 1:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Player1.String())

		default:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Player0.String())
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Player1.String())
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
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Missile0.String())

		case 1:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Missile1.String())

		default:
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Missile0.String())
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Missile1.String())
		}

	case cmdBall:
		dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Ball.String())

	case cmdPlayfield:
		dbg.printLine(terminal.StyleInstrument, dbg.vcs.TIA.Video.Playfield.String())

	case cmdPlusROM:
		plusrom, ok := dbg.vcs.Mem.Cart.GetContainer().(*plusrom.PlusROM)
		if !ok {
			dbg.printLine(terminal.StyleError, "not a plusrom cartridge")
			return nil
		}

		option, _ := tokens.Get()

		switch option {
		case "NICK":
			nick, _ := tokens.Get()
			err := dbg.vcs.Instance.Prefs.PlusROM.Nick.Set(nick)
			if err != nil {
				return err
			}
			err = dbg.vcs.Instance.Prefs.PlusROM.Save()
			if err != nil {
				return err
			}
		case "ID":
			id, _ := tokens.Get()
			err := dbg.vcs.Instance.Prefs.PlusROM.ID.Set(id)
			if err != nil {
				return err
			}
			err = dbg.vcs.Instance.Prefs.PlusROM.Save()
			if err != nil {
				return err
			}
		case "HOST":
			ai := plusrom.CopyAddrInfo()
			host, _ := tokens.Get()
			plusrom.SetAddrInfo(host, ai.Path)
		case "PATH":
			ai := plusrom.CopyAddrInfo()
			path, _ := tokens.Get()
			plusrom.SetAddrInfo(ai.Host, path)
		default:
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Nick: %s", dbg.vcs.Instance.Prefs.PlusROM.Nick.String()))
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("ID: %s", dbg.vcs.Instance.Prefs.PlusROM.ID.String()))
			ai := plusrom.CopyAddrInfo()
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Host: %s", ai.Host))
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Path: %s", ai.Path))
		}

	case cmdCoProc:
		coproc := dbg.vcs.Mem.Cart.GetCoProc()
		if coproc == nil {
			dbg.printLine(terminal.StyleError, "cartridge does not have a coprocessor")
			return nil
		}

		option, _ := tokens.Get()

		switch option {
		case "TOP":
			top := 10 // default of top 10

			arg, ok := tokens.Get()
			if ok {
				n, err := strconv.ParseInt(arg, 0, 32)
				if err != nil {
					dbg.printLine(terminal.StyleError, err.Error())
					return nil
				}
				top = int(n)
			}

			dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source files found")
					return
				}

				for i := 0; i < top; i++ {
					l := src.SortedLines.Lines[i]
					dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("%02d: %s", i, l.String()))
				}
			})

		case "LIST":
			arg, _ := tokens.Get()
			switch arg {
			case "ILLEGAL":
				dbg.CoProcDev.BorrowIllegalAccess(func(log *developer.IllegalAccess) {
					for _, e := range log.Log {
						if e.SrcLine != nil {
							dbg.printLine(terminal.StyleFeedback, e.SrcLine.String())
							dbg.printLine(terminal.StyleFeedback, e.SrcLine.PlainContent)
						} else {
							dbg.printLine(terminal.StyleFeedback,
								fmt.Sprintf("%s at address %08x (PC: %08x)", e.Event, e.AccessAddr, e.PC))
						}
					}
				})

			case "SOURCEFILES":
				dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
					if src == nil {
						dbg.printLine(terminal.StyleError, "no source files found")
						return
					}
					for _, fn := range src.Files {
						dbg.printLine(terminal.StyleFeedback, fn.Filename)
					}
				})
			default:
			}
		case "ID":
			fallthrough
		default:
			dbg.printLine(terminal.StyleFeedback, coproc.CoProcID())
		}

	case cmdPeripheral:
		player, _ := tokens.Get()

		var id plugging.PortID
		switch strings.ToUpper(player) {
		case "LEFT":
			id = plugging.PortLeftPlayer
		case "RIGHT":
			id = plugging.PortRightPlayer
		}

		var err error

		controller, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(controller) {
			case "AUTO":
				dbg.vcs.FingerprintPeripheral(id, *dbg.loader)
			case "STICK":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewStick)
			case "PADDLE":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewPaddle)
			case "KEYPAD":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewKeypad)
			case "GAMEPAD":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewGamepad)
			case "SAVEKEY":
				err = dbg.vcs.RIOT.Ports.Plug(id, savekey.NewSaveKey)
			case "ATARIVOX":
				err = dbg.vcs.RIOT.Ports.Plug(id, atarivox.NewAtariVox)
			}
		}

		if err != nil {
			return curated.Errorf("%v", err)
		}

		var p ports.Peripheral
		switch strings.ToUpper(player) {
		case "LEFT":
			p = dbg.vcs.RIOT.Ports.LeftPlayer
		case "RIGHT":
			p = dbg.vcs.RIOT.Ports.RightPlayer
		}

		dbg.printLine(terminal.StyleInstrument, p.String())

	case cmdPanel:
		mode, ok := tokens.Get()
		if !ok {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.RIOT.Ports.Panel.String())
			return nil
		}

		var err error

		switch strings.ToUpper(mode) {
		case "TOGGLE":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelTogglePlayer0Pro, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "P1":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelTogglePlayer1Pro, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "COL":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelToggleColor, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		case "SET":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "P0PRO":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer0Pro, D: true}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "P1PRO":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer1Pro, D: true}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "P0AM":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer0Pro, D: false}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "P1AM":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer1Pro, D: false}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "COL":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetColor, D: true}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "BW":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetColor, D: false}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		case "HOLD":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "SELECT":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSelect, D: true}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "RESET":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelReset, D: true}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		case "RELEASE":
			arg, _ := tokens.Get()
			switch strings.ToUpper(arg) {
			case "SELECT":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSelect, D: false}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			case "RESET":
				inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelReset, D: false}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		}

		if err != nil {
			return curated.Errorf("%v", err)
		}

		dbg.printLine(terminal.StyleInstrument, dbg.vcs.RIOT.Ports.Panel.String())

	case cmdStick:
		var err error

		port, _ := tokens.Get()
		action, _ := tokens.Get()

		var event ports.Event
		var value ports.EventData

		switch strings.ToUpper(action) {
		case "FIRE":
			event = ports.Fire
			value = ports.DataStickTrue
		case "UP":
			event = ports.Up
			value = ports.DataStickTrue
		case "DOWN":
			event = ports.Down
			value = ports.DataStickTrue
		case "LEFT":
			event = ports.Left
			value = ports.DataStickTrue
		case "RIGHT":
			event = ports.Right
			value = ports.DataStickTrue

		case "NOFIRE":
			event = ports.Fire
			value = ports.DataStickFalse
		case "NOUP":
			event = ports.Up
			value = ports.DataStickFalse
		case "NODOWN":
			event = ports.Down
			value = ports.DataStickFalse
		case "NOLEFT":
			event = ports.Left
			value = ports.DataStickFalse
		case "NORIGHT":
			event = ports.Right
			value = ports.DataStickFalse
		}

		switch port {
		case "LEFT":
			inp := ports.InputEvent{Port: plugging.PortLeftPlayer, Ev: event, D: value}
			_, err = dbg.vcs.Input.HandleInputEvent(inp)
		case "RIGHT":
			inp := ports.InputEvent{Port: plugging.PortRightPlayer, Ev: event, D: value}
			_, err = dbg.vcs.Input.HandleInputEvent(inp)
		}

		if err != nil {
			return err
		}

	case cmdKeypad:
		var err error

		port, _ := tokens.Get()
		key, _ := tokens.Get()

		switch port {
		case "LEFT":
			if strings.ToUpper(key) == "NONE" {
				inp := ports.InputEvent{Port: plugging.PortLeftPlayer, Ev: ports.KeypadUp, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			} else {
				inp := ports.InputEvent{Port: plugging.PortLeftPlayer, Ev: ports.KeypadDown, D: rune(key[0])}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		case "RIGHT":
			if strings.ToUpper(key) == "NONE" {
				inp := ports.InputEvent{Port: plugging.PortRightPlayer, Ev: ports.KeypadUp, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			} else {
				inp := ports.InputEvent{Port: plugging.PortLeftPlayer, Ev: ports.KeypadDown, D: rune(key[0])}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		}

		if err != nil {
			return err
		}

	case cmdBreak:
		err := dbg.halting.breakpoints.parseCommand(tokens)
		if err != nil {
			return curated.Errorf("%v", err)
		}

	case cmdTrap:
		err := dbg.halting.traps.parseCommand(tokens)
		if err != nil {
			return curated.Errorf("%v", err)
		}

	case cmdWatch:
		err := dbg.halting.watches.parseCommand(tokens)
		if err != nil {
			return curated.Errorf("%v", err)
		}

	case cmdTrace:
		err := dbg.traces.parseCommand(tokens)
		if err != nil {
			return curated.Errorf("%v", err)
		}

	case cmdList:
		list, _ := tokens.Get()
		list = strings.ToUpper(list)
		switch list {
		case "BREAKS":
			dbg.halting.breakpoints.list()
		case "TRAPS":
			dbg.halting.traps.list()
		case "WATCHES":
			dbg.halting.watches.list()
		case "TRACES":
			dbg.traces.list()
		case "ALL":
			dbg.halting.breakpoints.list()
			dbg.halting.traps.list()
			dbg.halting.watches.list()
			dbg.traces.list()
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdDrop:
		drop, _ := tokens.Get()

		s, _ := tokens.Get()
		num, err := strconv.Atoi(s)
		if err != nil {
			return curated.Errorf("drop attribute must be a number (%s)", s)
		}

		drop = strings.ToUpper(drop)
		switch drop {
		case "BREAK":
			err := dbg.halting.breakpoints.drop(num)
			if err != nil {
				return err
			}
			dbg.printLine(terminal.StyleFeedback, "breakpoint #%d dropped", num)
		case "TRAP":
			err := dbg.halting.traps.drop(num)
			if err != nil {
				return err
			}
			dbg.printLine(terminal.StyleFeedback, "trap #%d dropped", num)
		case "WATCH":
			err := dbg.halting.watches.drop(num)
			if err != nil {
				return err
			}
			dbg.printLine(terminal.StyleFeedback, "watch #%d dropped", num)
		case "TRACE":
			err := dbg.traces.drop(num)
			if err != nil {
				return err
			}
			dbg.printLine(terminal.StyleFeedback, "trace #%d dropped", num)
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdClear:
		clear, _ := tokens.Get()
		clear = strings.ToUpper(clear)
		switch clear {
		case "BREAKS":
			dbg.halting.breakpoints.clear()
			dbg.printLine(terminal.StyleFeedback, "breakpoints cleared")
		case "TRAPS":
			dbg.halting.traps.clear()
			dbg.printLine(terminal.StyleFeedback, "traps cleared")
		case "WATCHES":
			dbg.halting.watches.clear()
			dbg.printLine(terminal.StyleFeedback, "watches cleared")
		case "TRACES":
			dbg.traces.clear()
			dbg.printLine(terminal.StyleFeedback, "traces cleared")
		case "ALL":
			dbg.halting.breakpoints.clear()
			dbg.halting.traps.clear()
			dbg.halting.watches.clear()
			dbg.traces.clear()
			dbg.printLine(terminal.StyleFeedback, "breakpoints, traps, watches and traces cleared")
		default:
			// already caught by command line ValidateTokens()
		}

	case cmdLog:
		option, ok := tokens.Get()
		if ok {
			switch option {
			case "LAST":
				s := &strings.Builder{}
				logger.Tail(s, 1)
				dbg.printLine(terminal.StyleLog, s.String())
			case "RECENT":
				s := &strings.Builder{}
				logger.WriteRecent(s)
				dbg.printLine(terminal.StyleLog, s.String())
			case "CLEAR":
				logger.Clear()
			}
		} else {
			s := &strings.Builder{}
			logger.Write(s)
			if s.Len() == 0 {
				dbg.printLine(terminal.StyleFeedback, "log is empty")
			} else {
				dbg.printLine(terminal.StyleLog, s.String())
			}
		}

	case cmdMemUsage:
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		s := strings.Builder{}

		s.WriteString(fmt.Sprintf("Alloc = %v MB\n", m.Alloc/1048576))
		s.WriteString(fmt.Sprintf("  TotalAlloc = %v MB\n", m.TotalAlloc/1048576))
		s.WriteString(fmt.Sprintf("  Sys = %v MB\n", m.Sys/1048576))
		s.WriteString(fmt.Sprintf("  NumGC = %v", m.NumGC))

		dbg.printLine(terminal.StyleLog, s.String())
	}

	return nil
}
