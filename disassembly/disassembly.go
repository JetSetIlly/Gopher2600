package disassembly

import (
	"fmt"
	"gopher2600/disassembly/symbols"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
)

// Disassembly represents the annotated disassembly of a 6502 binary
type Disassembly struct {
	// symbols used to build disassembly output
	Symbols *symbols.Table

	// SequencePoints contains the list of program counter values. listed in
	// order so can be used to index program map to produce complete
	// disassembly
	SequencePoints []uint16

	// table of instruction results. index with contents of sequencePoints
	Program map[uint16]*cpu.InstructionResult
}

// ParseMemory disassembles an existing memory instance. uses a new cpu
// instance which has no side effects, so it's safe to use with "live" memory
func (dsm *Disassembly) ParseMemory(memory *memory.VCSMemory, symbols *symbols.Table) error {
	dsm.Symbols = symbols
	dsm.Program = make(map[uint16]*cpu.InstructionResult)
	dsm.SequencePoints = make([]uint16, 0, memory.Cart.Memtop()-memory.Cart.Origin())

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.New(memory)
	if err != nil {
		return err
	}
	mc.NoSideEffects = true
	mc.LoadPC(hardware.AddressReset)

	for {
		ir, err := mc.ExecuteInstruction(func(ir *cpu.InstructionResult) {})

		// filter out some errors
		if err != nil {
			switch err := err.(type) {
			case errors.GopherError:
				switch err.Errno {
				case errors.ProgramCounterCycled:
					// reached end of memory, exit loop with no errors
					// TODO: handle multi-bank ROMS
					return nil
				case errors.NullInstruction:
					// we've encountered a null instruction. ignore
					continue
				case errors.UnimplementedInstruction:
					// ignore unimplemented instructions
					continue
				default:
					return err
				}
			default:
				return err
			}
		}

		// add instruction result to disassembly result. an instruction result
		// of nil means that the part of the program just read by the CPU does
		// not contain valid instructions (maybe the assembler reasoned that
		// the code is unreachable)
		dsm.SequencePoints = append(dsm.SequencePoints, ir.Address)
		dsm.Program[ir.Address] = ir
	}
}

// NewDisassembly initialises a new partial emulation and returns a
// disassembly from the supplied cartridge filename. - useful for one-shot
// disassemblies, like the gopher2600 "disasm" mode
func NewDisassembly(cartridgeFilename string) (*Disassembly, error) {
	// ignore errors caused by loading of symbols table
	symbols, _ := symbols.NewTable(cartridgeFilename)

	mem, err := memory.New()
	if err != nil {
		return nil, err
	}

	err = mem.Cart.Attach(cartridgeFilename)
	if err != nil {
		return nil, err
	}

	dsm := new(Disassembly)
	err = dsm.ParseMemory(mem, symbols)
	if err != nil {
		return dsm, err
	}

	return dsm, nil
}

// Dump returns the entire disassembly as a string
func (dsm *Disassembly) Dump() (s string) {
	for _, pc := range dsm.SequencePoints {
		s = fmt.Sprintf("%s\n%s", s, dsm.Program[pc].GetString(dsm.Symbols, symbols.StyleFull))
	}
	return s
}
