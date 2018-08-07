package disassembly

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
)

// Disassembly represents the annotated disassembly of a 6502 binary
type Disassembly struct {
	// symbols used to build disassembly output
	Symtable *symbols.Table

	// SequencePoints contains the list of program counter values. listed in
	// order so can be used to index program map to produce complete
	// disassembly
	SequencePoints []uint16

	// table of instruction results. index with contents of sequencePoints
	Program map[uint16]*result.Instruction
}

// ParseMemory disassembles an existing memory instance. uses a new cpu
// instance which has no side effects, so it's safe to use with "live" memory
func (dsm *Disassembly) ParseMemory(memory *memory.VCSMemory, symtable *symbols.Table) error {
	dsm.Symtable = symtable
	dsm.Program = make(map[uint16]*result.Instruction)
	dsm.SequencePoints = make([]uint16, 0, memory.Cart.Memtop()-memory.Cart.Origin())

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.NewCPU(memory)
	if err != nil {
		return err
	}
	mc.NoSideEffects = true
	mc.LoadPC(hardware.AddressReset)

	for {
		ir, err := mc.ExecuteInstruction(func(ir *result.Instruction) {})

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
				case errors.UnreadableAddress:
					// ignore unreadable addresses
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
	symtable, err := symbols.ReadSymbolsFile(cartridgeFilename)
	if err != nil {
		fmt.Println(err)
		symtable, err = symbols.StandardSymbolTable()
		if err != nil {
			return nil, err
		}
	}

	mem, err := memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	err = mem.Cart.Attach(cartridgeFilename)
	if err != nil {
		return nil, err
	}

	dsm := new(Disassembly)
	err = dsm.ParseMemory(mem, symtable)
	if err != nil {
		return dsm, err
	}

	return dsm, nil
}

// Dump returns the entire disassembly as a string
func (dsm *Disassembly) Dump() (s string) {
	// TODO: buffered output - it can take too long to build the string if the
	// disassembly is too long
	for _, pc := range dsm.SequencePoints {
		s = fmt.Sprintf("%s\n%s", s, dsm.Program[pc].GetString(dsm.Symtable, result.StyleFull))
	}
	return s
}
