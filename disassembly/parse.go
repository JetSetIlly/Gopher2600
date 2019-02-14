package disassembly

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
)

// ParseMemory disassembles an existing memory instance. uses a new cpu
// instance which has no side effects, so it's safe to use with "live" memory
func (dsm *Disassembly) ParseMemory(mem *memory.VCSMemory, symtable *symbols.Table) error {
	dsm.Cart = mem.Cart
	dsm.Symtable = symtable
	dsm.Program = make([]map[uint16]*result.Instruction, dsm.Cart.NumBanks)
	dsm.sequencePoints = make([][]uint16, dsm.Cart.NumBanks)

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return err
	}
	mc.NoSideEffects = true

	// start disassembly at reset point
	mc.LoadPCIndirect(memory.AddressReset)

	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		dsm.Cart.BankSwitch(bank)
		dsm.Program[bank] = make(map[uint16]*result.Instruction)
		dsm.sequencePoints[bank] = make([]uint16, 0, dsm.Cart.Memtop()-dsm.Cart.Origin())

		nextBank := false
		for !nextBank {
			ir, err := mc.ExecuteInstruction(func(ir *result.Instruction) {})

			// filter out some errors
			if err != nil {
				switch err := err.(type) {
				case errors.GopherError:
					switch err.Errno {
					case errors.ProgramCounterCycled:
						// reached end of memory
						nextBank = true
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

			// if nextBank flag has been set then break inner for loop
			if nextBank {
				break
			}

			// check validity of instruction result
			err = ir.IsValid()
			if err != nil {
				return err
			}

			// add instruction result to disassembly result. an instruction result
			// of nil means that the part of the program just read by the CPU does
			// not contain valid instructions (maybe the assembler reasoned that
			// the code is unreachable)
			dsm.sequencePoints[bank] = append(dsm.sequencePoints[bank], ir.Address)
			dsm.Program[bank][ir.Address] = ir
		}

		// start disassembly of subsequent bank at origin point of cartridge
		// space - this may not be correct in all instances
		mc.LoadPC(dsm.Cart.Origin())
	}

	// make sure we're in the starting bank
	dsm.Cart.BankSwitch(0)

	return nil
}
