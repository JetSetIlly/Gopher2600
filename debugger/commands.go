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
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor"
	coproc_breakpoints "github.com/jetsetilly/gopher2600/coprocessor/developer/breakpoints"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/callstack"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/yield"
	"github.com/jetsetilly/gopher2600/coprocessor/faults"
	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/gui"
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
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/gopher2600/version"
)

var debuggerCommands *commandline.Commands
var scriptUnsafeCommands *commandline.Commands

// this init() function "compiles" the commandTemplate above into a more
// usuable form. It will cause the program to fail if the template is invalid.
func init() {
	var err error

	debuggerCommands, err = commandline.ParseCommandTemplate(commandTemplate)
	if err != nil {
		panic(err)
	}

	err = commandline.AddHelp(debuggerCommands)
	if err != nil {
		panic(err)
	}

	scriptUnsafeCommands, err = commandline.ParseCommandTemplate(scriptUnsafeTemplate)
	if err != nil {
		panic(err)
	}
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
			return nil, fmt.Errorf("'%s' is unsafe to use in scripts", tokens.String())
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
		return fmt.Errorf("%s is not yet implemented", command)

	case commandline.HelpCommand:
		if topic, ok := tokens.Get(); ok {
			topic = strings.ToUpper(topic)
			dbg.printLine(terminal.StyleHelp, helps[topic])

			// also print usage command if the command has arguments
			usage := debuggerCommands.Usage(topic)
			if strings.Count(usage, " ") > 0 {
				dbg.printLine(terminal.StyleHelp, "")
				dbg.printLine(terminal.StyleHelp, fmt.Sprintf("Usage: %s", debuggerCommands.Usage(topic)))
			}
		} else {
			dbg.printLine(terminal.StyleHelp, commandline.HelpSummary(debuggerCommands))
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
				// if next expected opcode is JSR then add a volatile breakpoint to the return address
				//
				// at the time of writing this comment, if execution is currently in RAM then
				// GetEntryByAddress() will return nil. this means that if "JSR" is encounterd during
				// that time, the STEP OVER command will not work as expected
				e := dbg.Disasm.GetEntryByAddress(dbg.vcs.CPU.PC.Address())
				if e != nil && e.Operator == "jsr" {
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
			// step backwards

			var coords coords.TelevisionCoords

			switch mode {
			case "":
				// use current quantum state
				switch dbg.Quantum() {
				case govern.QuantumInstruction:
					coords = dbg.cpuBoundaryLastInstruction
				case govern.QuantumCycle:
					coords = dbg.vcs.TV.AdjCoords(television.AdjCycle, adjAmount)
				case govern.QuantumClock:
					coords = dbg.vcs.TV.AdjCoords(television.AdjClock, adjAmount)
				}

			case "INSTRUCTION":
				dbg.setQuantum(govern.QuantumInstruction)
				coords = dbg.cpuBoundaryLastInstruction
			case "CYCLE":
				dbg.setQuantum(govern.QuantumCycle)
				coords = dbg.vcs.TV.AdjCoords(television.AdjCycle, adjAmount)
			case "CLOCK":
				dbg.setQuantum(govern.QuantumClock)
				coords = dbg.vcs.TV.AdjCoords(television.AdjClock, adjAmount)
			case "SCANLINE":
				coords = dbg.vcs.TV.AdjCoords(television.AdjScanline, adjAmount)
			case "FRAME":
				coords = dbg.vcs.TV.AdjCoords(television.AdjFrame, adjAmount)
			default:
				return fmt.Errorf("unknown STEP BACK mode (%s)", mode)
			}

			dbg.setState(govern.Rewinding, govern.RewindingBackwards)
			dbg.unwindLoop(func() error {
				dbg.catchupContext = catchupStepBack
				return dbg.Rewind.GotoCoords(coords)
			})

		} else {
			// step forwards

			switch mode {
			case "":
				// continue with current quantum state

				// if quantum is not the QuantumClock and CPU is not RDY then STEP
				// is best implemented as TRAP RDY. this means that the emulation
				// will stop on the next instruction boundary and will also skip
				// over instructions that trigger WSYNC
				//
				// this behaviour is more intuitive to the user because it means
				// they don't have to step over every cycle during the WSYNC state
				if dbg.Quantum() != govern.QuantumClock && !dbg.vcs.CPU.RdyFlg {
					// create volatile RDY trap
					_ = dbg.halting.volatileTraps.parseCommand(commandline.TokeniseInput("RDY"))
					dbg.runUntilHalt = true

					// when the RDY flag changes the input loop will think it's
					// inside a video step. we need to force the loop to return
					// to the non-video step loop
					dbg.stepOutOfVideoStepInputLoop = true
				}
			case "INSTRUCTION":
				dbg.setQuantum(govern.QuantumInstruction)
			case "CYCLE":
				dbg.setQuantum(govern.QuantumCycle)
			case "CLOCK":
				dbg.setQuantum(govern.QuantumClock)
			default:
				// token not recognised so forward rest of tokens to the volatile
				// traps parser
				tokens.Unget()
				_ = dbg.halting.volatileTraps.parseCommand(tokens)

				// trap may take many cycles to trigger
				dbg.runUntilHalt = true
			}

			// continue emulation. note that we don't set runUntilHalt except in the
			// specific cases above in the above switch. this is because we do no
			// always set a volatile trap. without a trap the emulation will just
			// run until it receives a HALT instruction.
			dbg.continueEmulation = true
		}

	case cmdQuantum:
		mode, _ := tokens.Get()
		mode = strings.ToUpper(mode)
		switch mode {
		case "INSTRUCTION":
			dbg.setQuantum(govern.QuantumInstruction)
		case "CYCLE":
			dbg.setQuantum(govern.QuantumCycle)
		case "CLOCK":
			dbg.setQuantum(govern.QuantumClock)
		default:
			dbg.printLine(terminal.StyleFeedback, "set to %s", strings.ToUpper(dbg.Quantum().String()))
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
				dbg.setState(govern.Rewinding, govern.RewindingForwards)
				dbg.unwindLoop(dbg.Rewind.GotoLast)
			} else if arg == "SUMMARY" {
				dbg.printLine(terminal.StyleInstrument, dbg.Rewind.Peephole())
			} else {
				frame, _ := strconv.Atoi(arg)
				coords := dbg.TV().GetCoords()
				if frame != coords.Frame {
					if frame < coords.Frame {
						dbg.setState(govern.Rewinding, govern.RewindingBackwards)
					} else {
						dbg.setState(govern.Rewinding, govern.RewindingForwards)
					}
					dbg.unwindLoop(func() error {
						err := dbg.Rewind.GotoFrame(frame)
						if err != nil {
							return err
						}
						return nil
					})
				}
			}
			return nil
		}

	case cmdComparison:
		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "LOCK":
				dbg.Rewind.LockComparison(true)
			case "UNLOCK":
				dbg.Rewind.LockComparison(false)
				if dbg.State() == govern.Running {
					dbg.Rewind.UpdateComparison()
				}
			default:
				frame, _ := strconv.Atoi(arg)
				dbg.Rewind.SetComparison(frame)
			}
		}

	case cmdGoto:
		fromCoords := dbg.vcs.TV.GetCoords()
		toCoords := fromCoords

		if s, ok := tokens.Get(); ok {
			toCoords.Clock, _ = strconv.Atoi(s)
			if s, ok := tokens.Get(); ok {
				toCoords.Scanline, _ = strconv.Atoi(s)
				if s, ok := tokens.Get(); ok {
					toCoords.Frame, _ = strconv.Atoi(s)
				}
			}
		}

		if coords.GreaterThan(toCoords, fromCoords) {
			dbg.setState(govern.Rewinding, govern.RewindingForwards)
		} else {
			dbg.setState(govern.Rewinding, govern.RewindingBackwards)
		}
		dbg.unwindLoop(func() error {
			err := dbg.Rewind.GotoCoords(toCoords)
			if err != nil {
				return err
			}
			return nil
		})

	case cmdInsert:
		dbg.unwindLoop(func() error {
			filename, _ := tokens.Get()
			err := dbg.insertCartridge(filename)
			if err != nil {
				return err
			}
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

			case "DUMP":
				romdump, err := dbg.vcs.Mem.Cart.ROMDump()
				if err != nil {
					dbg.printLine(terminal.StyleFeedback, err.Error())
				} else {
					dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("rom dumped to %s", romdump))
				}

			case "SETBANK":
				spec, _ := tokens.Get()
				err := dbg.vcs.Mem.Cart.SetBank(spec)
				if err != nil {
					dbg.printLine(terminal.StyleError, err.Error())
				}
			default:
				tokens.Unget()
				w := dbg.writerInStyle(terminal.StyleFeedback)
				err := dbg.vcs.Mem.Cart.ParseCommand(w, tokens.Remainder())
				if err != nil {
					dbg.printLine(terminal.StyleError, err.Error())
				}
			}
		} else {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.Mem.Cart.String())
		}

	case cmdPatch:
		f, _ := tokens.Get()
		err := patch.CartridgeMemoryFromFile(dbg.vcs.Mem.Cart, f)
		if err != nil {
			dbg.printLine(terminal.StyleError, "%v", err)
		} else {
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
		output := &strings.Builder{}

		s, _ := tokens.Get()
		switch strings.ToUpper(s) {
		case "COPROC":
			search, _ := tokens.Get()

			addr, err := strconv.ParseUint(search, 0, 32)
			if err != nil {
				dbg.printLine(terminal.StyleError, "search term for COPROC must be a 32bit address")
				return nil
			}

			var ln *dwarf.SourceLine

			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source files found")
					return
				}

				var ok bool

				ln, ok = src.LinesByAddress[addr]
				if !ok {
					dbg.printLine(terminal.StyleError, fmt.Sprintf("address %x does not correspond to a source line", addr))
				} else {
					dbg.printLine(terminal.StyleFeedback, ln.String())
				}
			})

			_ = dbg.gui.SetFeature(gui.ReqCoProcSourceLine, ln)
			return nil

		case "OPERATOR":
			scope = disassembly.GrepOperator
		case "OPERAND":
			scope = disassembly.GrepOperand
		default:
			tokens.Unget()
		}

		search, _ := tokens.Get()
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
					dbg.dbgmem.Sym.ListLabels(dbg.writerInStyle(terminal.StyleFeedback))

				case "READ":
					dbg.dbgmem.Sym.ListReadSymbols(dbg.writerInStyle(terminal.StyleFeedback))

				case "WRITE":
					dbg.dbgmem.Sym.ListWriteSymbols(dbg.writerInStyle(terminal.StyleFeedback))
				}
			} else {
				dbg.dbgmem.Sym.ListSymbols(dbg.writerInStyle(terminal.StyleFeedback))
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
		// if debugger is running in a non-instruction quantum then the live disasm
		// information will not have been updated. for the purposes of the last
		// instruction however, we definitely do want that information to be
		// current
		if dbg.running && dbg.quantum.Load() != govern.QuantumInstruction {
			dbg.liveBankInfo = dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())
			dbg.liveDisasmEntry = dbg.Disasm.ExecutedEntry(dbg.liveBankInfo, dbg.vcs.CPU.LastResult, true, dbg.vcs.CPU.PC.Value())
		}

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
			dbg.printLine(terminal.StyleInstructionStep, s.String())
		} else {
			dbg.printLine(terminal.StyleSubStep, s.String())
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
					v, err := strconv.ParseUint(value, 0, 16)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value must be a positive 16 bit number")
					}

					dbg.vcs.CPU.PC.Load(uint16(v))
				} else {
					// 6507 registers are 8 bit
					v, err := strconv.ParseUint(value, 0, 8)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value must be a positive 8 bit number")
					}

					var reg *registers.Data
					switch strings.ToUpper(target) {
					case "A":
						reg = &dbg.vcs.CPU.A
					case "X":
						reg = &dbg.vcs.CPU.X
					case "Y":
						reg = &dbg.vcs.CPU.Y
					case "SP":
						reg = &dbg.vcs.CPU.SP.Data
					}

					reg.Load(uint8(v))
				}

			default:
				// already caught by command line ValidateTokens()
			}
		} else {
			dbg.printLine(terminal.StyleInstrument, dbg.vcs.CPU.String())
		}

	case cmdBus:
		dbg.printLine(terminal.StyleInstrument, dbg.vcs.Mem.String())
		action, ok := tokens.Get()
		if ok {
			switch strings.ToUpper(action) {
			case "DETAIL":
				_, area := memorymap.MapAddress(dbg.vcs.Mem.AddressBus, !dbg.vcs.Mem.LastCPUWrite)
				access := "reading"
				if dbg.vcs.Mem.LastCPUWrite {
					access = "writing"
				}
				dbg.printLine(terminal.StyleInstrument, fmt.Sprintf("%s (%s)", area.String(), access))
			default:
				// already caught by command line ValidateTokens()
			}
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
		ai := dbg.dbgmem.GetAddressInfo(a, false)
		if ai == nil {
			dbg.printLine(terminal.StyleError, fmt.Sprintf("%s: %v", dbgmem.PokeError, a))
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
				dbg.printLine(terminal.StyleInstrument, fmt.Sprintf("%#02x poked to %s", val, ai.String()))
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
			case "FRAME":
				frameInfo := dbg.vcs.TV.GetFrameInfo()
				dbg.printLine(terminal.StyleInstrument, frameInfo.String())

			case "SPEC":
				spec, ok := tokens.Get()
				if ok {
					// unknown specifciations already handled by ValidateTokens()
					err := dbg.vcs.TV.SetSpec(spec)
					if err != nil {
						return err
					}

					if dbg.State() == govern.Paused {
						dbg.RerunLastNFrames(10, func(s *rewind.State) {
							s.TV.SetSpec(spec)
						})
					}
				}

				dbg.printLine(terminal.StyleInstrument, dbg.vcs.TV.SpecString())

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
			err := dbg.vcs.Env.Prefs.PlusROM.Nick.Set(nick)
			if err != nil {
				return err
			}
			err = dbg.vcs.Env.Prefs.PlusROM.Save()
			if err != nil {
				return err
			}
		case "ID":
			id, _ := tokens.Get()
			err := dbg.vcs.Env.Prefs.PlusROM.ID.Set(id)
			if err != nil {
				return err
			}
			err = dbg.vcs.Env.Prefs.PlusROM.Save()
			if err != nil {
				return err
			}
		case "HOST":
			ai := plusrom.GetAddrInfo()
			host, _ := tokens.Get()
			plusrom.SetAddrInfo(host, ai.Path)
		case "PATH":
			ai := plusrom.GetAddrInfo()
			path, _ := tokens.Get()
			plusrom.SetAddrInfo(ai.Host, path)
		default:
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Nick: %s", dbg.vcs.Env.Prefs.PlusROM.Nick.String()))
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("ID: %s", dbg.vcs.Env.Prefs.PlusROM.ID.String()))
			ai := plusrom.GetAddrInfo()
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Host: %s", ai.Host))
			dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("Path: %s", ai.Path))
		}

	case cmdCoProc:
		bus := dbg.vcs.Mem.Cart.GetCoProcBus()
		if bus == nil {
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
					dbg.printLine(terminal.StyleError, fmt.Sprintf("%s is not a number", arg))
					return nil
				}
				top = int(n)
			}

			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source files found")
					return
				}

				for i := 0; i < top; i++ {
					f := src.SortedFunctions.Functions[i]
					dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("%02d: %s", i, f.Name))
				}
			})

		case "LIST":
			arg, _ := tokens.Get()
			switch arg {

			case "FAULTS":
				dbg.CoProcDev.BorrowFaults(func(flt *faults.Faults) {
					w := dbg.writerInStyle(terminal.StyleFeedback)
					flt.WriteLog(w)
				})

			case "SOURCEFILES":
				dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
					if src == nil {
						dbg.printLine(terminal.StyleError, "no source files found")
						return
					}
					for _, fn := range src.Files {
						dbg.printLine(terminal.StyleFeedback, fn.Filename)
					}
				})
			case "FUNCTIONS":
				dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
					if src == nil {
						dbg.printLine(terminal.StyleError, "no source files found")
						return
					}
					for _, n := range src.FunctionNames {
						fn := src.Functions[n]
						dbg.printLine(terminal.StyleFeedback, fn.String())
					}
				})
			default:
			}

		case "MEM":
			bus := dbg.vcs.Mem.Cart.GetStaticBus()
			if bus == nil {
				dbg.printLine(terminal.StyleError, "cartridge does not have any coprocessor memory")
				return nil
			}

			static := bus.GetStatic()
			if static == nil {
				dbg.printLine(terminal.StyleError, "cartridge does not have any coprocessor memory")
				return nil
			}

			arg, _ := tokens.Get()
			switch arg {
			case "DUMP":
				dump := func(name string) {
					if data, ok := static.Reference(name); ok {
						fn := unique.Filename(fmt.Sprintf("dump_%s", name), dbg.cartload.Name)
						fn = fmt.Sprintf("%s.bin", fn)
						err := os.WriteFile(fn, data, 0644)
						if err != nil {
							dbg.printLine(terminal.StyleError, fmt.Sprintf("error writing %s", fn))
						} else {
							dbg.printLine(terminal.StyleFeedback, fn)
						}
					}
				}

				if arg, ok := tokens.Get(); ok {
					dump(arg)
				} else {
					for _, seg := range static.Segments() {
						dump(seg.Name)
					}
				}
			case "SEARCH":
				if v, ok := tokens.Get(); ok {
					val, err := strconv.ParseUint(v, 0, 32)
					if err != nil {
						dbg.printLine(terminal.StyleError, "value argument for COPROC SEARCH must be a 32bit value")
						return nil
					}

					if v, ok := tokens.Get(); ok {
						bitwidth, err := strconv.ParseUint(v, 0, 8)
						if err != nil {
							dbg.printLine(terminal.StyleError, "bitwidth argument for COPROC SEARCH must be 8, 16, or 32")
							return nil
						}
						if bitwidth != 8 && bitwidth != 16 && bitwidth != 32 {
							dbg.printLine(terminal.StyleError, "bitwidth argument for COPROC SEARCH must be 8, 16, or 32")
							return nil
						}

						dbg.CoProcDev.SearchStaticMemory(dbg.writerInStyle(terminal.StyleFeedback), uint32(val), int(bitwidth)/8)
					}
				}
			default:
				for _, seg := range static.Segments() {
					s := fmt.Sprintf("%s: %08x to %08x", seg.Name, seg.Origin, seg.Memtop)
					dbg.printLine(terminal.StyleFeedback, s)
				}
			}

		case "REGS":
			coproc := bus.GetCoProc()

			// list registers in order until we get a not-ok reply
			regs := func(group coprocessor.ExtendedRegisterGroup) {
				s := strings.Builder{}
				for r := group.Start; r <= group.End; r++ {
					v, f, ok := coproc.RegisterFormatted(r)
					if !ok {
						dbg.printLine(terminal.StyleError,
							fmt.Sprintf("coprocessor doesn't have the %d register in the %s group", r, group.Name))
						return
					}
					if group.Formatted {
						s.WriteString(fmt.Sprintf("%s: %s [%08x]\t", group.Label(r), f, v))
					} else {
						s.WriteString(fmt.Sprintf("%s: %08x\t", group.Label(r), v))
					}

					if (r-group.Start+1)%3 == 0 {
						dbg.printLine(terminal.StyleFeedback, s.String())
						s.Reset()
					}
				}
				if s.Len() > 0 {
					dbg.printLine(terminal.StyleFeedback, s.String())
				}
			}

			// use named group if supplied or default to core group
			spec := coproc.RegisterSpec()
			if arg, ok := tokens.Get(); ok {
				if group, ok := spec.Group(arg); ok {
					regs(group)
				} else {
					dbg.printLine(terminal.StyleError, fmt.Sprintf("coprocessor doesn't have a %s register group", arg))
				}
			} else {
				if group, ok := spec.Group(coprocessor.ExtendedRegisterCoreGroup); ok {
					regs(group)
				} else {
					dbg.printLine(terminal.StyleError, "coprocessor doesn't seem to have any registers")
				}
			}

		case "SET":
			var reg int
			var value uint32
			arg, ok := tokens.Get()
			if ok {
				n, err := strconv.ParseInt(arg, 0, 32)
				if err != nil {
					dbg.printLine(terminal.StyleError, fmt.Sprintf("%s is not a number", arg))
					return nil
				}
				reg = int(n)
			}
			arg, ok = tokens.Get()
			if ok {
				n, err := strconv.ParseInt(arg, 0, 32)
				if err != nil {
					dbg.printLine(terminal.StyleError, fmt.Sprintf("%s is not a number", arg))
					return nil
				}
				value = uint32(n)
			}
			if bus.GetCoProc().RegisterSet(reg, value) {
				dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("setting coproc register %d to %08x\n", reg, value))
			} else {
				dbg.printLine(terminal.StyleError, fmt.Sprintf("cannot set coproc register %d to %08x\n", reg, value))
			}

		case "STEP":
			dbg.CoProcDev.BreakNextInstruction()
			dbg.runUntilHalt = true
			dbg.continueEmulation = true

		case "ID":
			fallthrough
		default:
			dbg.printLine(terminal.StyleFeedback, bus.GetCoProc().ProcessorID())
		}

	case cmdDWARF:
		coproc := dbg.vcs.Mem.Cart.GetCoProcBus()
		if coproc == nil {
			dbg.printLine(terminal.StyleError, "cartridge does not have a coprocessor")
			return nil
		}

		option, _ := tokens.Get()

		switch option {
		case "FUNCTIONS":
			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source available")
					return
				}

				for _, n := range src.FunctionNames {
					f := src.Functions[n]
					dbg.printLine(terminal.StyleFeedback, f.String())
				}
			})
		case "GLOBALS":
			var w io.Writer
			if option, ok := tokens.Get(); ok {
				if option == "DERIVATION" {
					w = dbg.writerInStyle(terminal.StyleFeedbackSecondary, "\t")
				}
			}

			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source available")
					return
				}

				for _, g := range src.SortedGlobals.Variables {
					g.Update()
					dbg.printLine(terminal.StyleFeedback, g.String())
					e := g.WriteDerivation(w)
					if e != nil {
						for _, s := range strings.Split(e.Error(), ":") {
							dbg.printLine(terminal.StyleError, fmt.Sprintf("\t%s", s))
						}
					}
				}
			})
		case "LOCALS":
			var derivation bool
			var ranges bool

			option, ok := tokens.Get()
			for ok {
				derivation = derivation || option == "DERIVATION"
				ranges = ranges || option == "RANGES"
				option, ok = tokens.Get()
			}

			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source available")
				}
			})

			dbg.CoProcDev.BorrowYieldState(func(yld yield.State) {
				var w io.Writer
				if derivation {
					w = dbg.writerInStyle(terminal.StyleFeedbackSecondary, "\t")
				}
				for _, l := range yld.LocalVariables {
					dbg.printLine(terminal.StyleFeedback, l.String())
					e := l.WriteDerivation(w)
					if e != nil {
						for _, s := range strings.Split(e.Error(), ":") {
							dbg.printLine(terminal.StyleError, fmt.Sprintf("\t%s", s))
						}
					}
					if ranges {
						dbg.printLine(terminal.StyleFeedbackSecondary, fmt.Sprintf("\t%s", l.Range.String()))
					}
				}
			})
		case "FRAMEBASE":
			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source available")
				}

				var derivation bool
				option, ok := tokens.Get()
				for ok {
					derivation = derivation || option == "DERIVATION"
					option, ok = tokens.Get()
				}

				var w io.Writer
				if derivation {
					w = dbg.writerInStyle(terminal.StyleFeedback)
				}

				fb, err := src.FramebaseCurrent(w)
				if err != nil {
					dbg.printLine(terminal.StyleError, err.Error())
				} else {
					dbg.printLine(terminal.StyleFeedback, fmt.Sprintf("%08x", fb))
				}
			})
		case "LINE":
			arg, ok := tokens.Get()
			if !ok {
				dbg.printLine(terminal.StyleError, "command requires argument file:line")
				return nil
			}

			// option is divided by a maximum of one colon, meaning the split
			// array should be a length of two
			s := strings.Split(arg, ":")
			if len(s) != 2 {
				dbg.printLine(terminal.StyleError, "command requires argument file:line")
				return nil
			}

			// filename and line number
			fn := s[0]
			n, err := strconv.ParseInt(s[1], 0, 32)
			if err != nil {
				dbg.printLine(terminal.StyleError, fmt.Sprintf("%s is not a number", s[1]))
				return nil
			}
			ln := int(n)

			dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				if src == nil {
					dbg.printLine(terminal.StyleError, "no source available")
				}

				// check file by shortname and then by full name
				f, ok := src.FilesByShortname[fn]
				if !ok {
					f, ok = src.Files[fn]
					if !ok {
						dbg.printLine(terminal.StyleError, fmt.Sprintf("no file named %s", fn))
						return
					}
				}

				// line numbers are counted from one
				if ln < 1 {
					ln = 1
				}
				if ln > len(f.Content.Lines) {
					dbg.printLine(terminal.StyleError, fmt.Sprintf("%s only has %d lines", fn, len(f.Content.Lines)))
					return
				}
				l := f.Content.Lines[ln-1]

				// display what we know about line
				dbg.printLine(terminal.StyleFeedback, l.String())
			})

		case "CALLSTACK":
			dbg.CoProcDev.BorrowCallStack(func(callstack callstack.CallStack) {
				w := dbg.writerInStyle(terminal.StyleFeedback)
				callstack.WriteCallStack(w)
			})

		case "CALLERS":
			arg, ok := tokens.Get()
			if !ok {
				dbg.printLine(terminal.StyleError, "function name is required")
				return nil
			}
			dbg.CoProcDev.BorrowCallStack(func(callstack callstack.CallStack) {
				w := dbg.writerInStyle(terminal.StyleFeedback)
				if err := callstack.WriteCallers(arg, w); err != nil {
					dbg.printLine(terminal.StyleError, err.Error())
					return
				}
			})
		}

	case cmdPeripheral:
		player, _ := tokens.Get()
		player = strings.ToUpper(player)

		var id plugging.PortID

		switch player {
		case "LEFT":
			id = plugging.PortLeft
		case "RIGHT":
			id = plugging.PortRight
		case "SWAP":
			if dbg.controllers.Swap() {
				dbg.printLine(terminal.StyleFeedback, "player peripherals are in the swapped state")
			} else {
				dbg.printLine(terminal.StyleFeedback, "player peripherals are in the normal state")
			}
			return nil
		}

		controller, ok := tokens.Get()
		if ok {
			var err error

			controller = strings.ToUpper(controller)
			switch controller {
			case "AUTO":
				dbg.vcs.FingerprintPeripheral(id)
			case "STICK":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewStick)
			case "PADDLE":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewPaddlePair)
			case "KEYPAD":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewKeypad)
			case "GAMEPAD":
				err = dbg.vcs.RIOT.Ports.Plug(id, controllers.NewGamepad)
			case "SAVEKEY":
				err = dbg.vcs.RIOT.Ports.Plug(id, savekey.NewSaveKey)
			case "ATARIVOX":
				err = dbg.vcs.RIOT.Ports.Plug(id, atarivox.NewAtariVox)
			}

			if err != nil {
				return err
			}

			dbg.printLine(terminal.StyleFeedback, "%s inserted for %s player", controller, player)
			return nil
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
			return err
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
			inp := ports.InputEvent{Port: plugging.PortLeft, Ev: event, D: value}
			_, err = dbg.vcs.Input.HandleInputEvent(inp)
		case "RIGHT":
			inp := ports.InputEvent{Port: plugging.PortRight, Ev: event, D: value}
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
				inp := ports.InputEvent{Port: plugging.PortLeft, Ev: ports.KeypadUp, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			} else {
				inp := ports.InputEvent{Port: plugging.PortLeft, Ev: ports.KeypadDown, D: rune(key[0])}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		case "RIGHT":
			if strings.ToUpper(key) == "NONE" {
				inp := ports.InputEvent{Port: plugging.PortRight, Ev: ports.KeypadUp, D: nil}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			} else {
				inp := ports.InputEvent{Port: plugging.PortLeft, Ev: ports.KeypadDown, D: rune(key[0])}
				_, err = dbg.vcs.Input.HandleInputEvent(inp)
			}
		}

		if err != nil {
			return err
		}

	case cmdBreak:
		err := dbg.halting.breakpoints.parseCommand(tokens)
		if err != nil {
			return err
		}

	case cmdTrap:
		err := dbg.halting.traps.parseCommand(tokens)
		if err != nil {
			return err
		}

	case cmdWatch:
		err := dbg.halting.watches.parseCommand(tokens)
		if err != nil {
			return err
		}

	case cmdTrace:
		err := dbg.traces.parseCommand(tokens)
		if err != nil {
			return err
		}

	case cmdList:
		list, _ := tokens.Get()
		list = strings.ToUpper(list)
		switch list {
		case "BREAKS":
			dbg.halting.breakpoints.list()
			dbg.CoProcDev.BorrowBreakpoints(func(bp coproc_breakpoints.Breakpoints) {
				w := dbg.writerInStyle(terminal.StyleFeedback)
				bp.Write(w)
			})
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
			return fmt.Errorf("drop attribute must be a number (%s)", s)
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
			option = strings.ToUpper(option)
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
		option, ok := tokens.Get()
		if ok {
			option = strings.ToUpper(option)
			switch option {
			case "PROFILE":
				fn, err := dbg.memoryProfile()
				if err != nil {
					return err
				}
				dbg.printLine(terminal.StyleLog, fmt.Sprintf("memory profile written to %s", fn))
			}
		} else {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			s := strings.Builder{}

			s.WriteString(fmt.Sprintf("Alloc = %v MB\n", m.Alloc/1048576))
			s.WriteString(fmt.Sprintf("  TotalAlloc = %v MB\n", m.TotalAlloc/1048576))
			s.WriteString(fmt.Sprintf("  Sys = %v MB\n", m.Sys/1048576))
			s.WriteString(fmt.Sprintf("  NumGC = %v", m.NumGC))

			dbg.printLine(terminal.StyleLog, s.String())
		}

	case cmdVersion:
		ver, rev, _ := version.Version()
		dbg.printLine(terminal.StyleLog, ver)

		option, ok := tokens.Get()
		if ok {
			option = strings.ToUpper(option)
			switch option {
			case "REVISION":
				dbg.printLine(terminal.StyleLog, rev)
			}
		}
	}

	return nil
}
