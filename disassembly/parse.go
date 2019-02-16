package disassembly

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
)

func distributionAnalysis(mc *cpu.CPU, mem memory.VCSMemory) {
	for bank := 0; bank < mem.Cart.NumBanks; bank++ {
		mem.Cart.BankSwitch(bank)
		mc.LoadPCIndirect(mem.Cart.Origin())

		// unfinished
	}
}

// ParseMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control.
func (dsm *Disassembly) ParseMemory(cart *memory.Cartridge, symtable *symbols.Table) error {
	dsm.Cart = cart
	dsm.Symtable = symtable
	dsm.Program = make([]map[uint16]*result.Instruction, dsm.Cart.NumBanks)

	// create new memory
	mem, err := newMinimalMemory(dsm.Cart)
	if err != nil {
		return err
	}

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return err
	}

	mc.NoFlowControl = true

	// allocate memory for disassembly
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		dsm.Cart.BankSwitch(bank)
		dsm.Program[bank] = make(map[uint16]*result.Instruction)
	}

	// make sure we're in the starting bank - at the beginning of the
	// disassembly and at the end
	dsm.Cart.BankSwitch(0)
	mc.LoadPCIndirect(memory.AddressReset)
	defer func() {
		dsm.Cart.BankSwitch(0)
		mc.LoadPCIndirect(memory.AddressReset)
	}()

	for {
		currentBank := dsm.Cart.Bank
		ir, err := mc.ExecuteInstruction(func(ir *result.Instruction) {})

		// filter out some errors
		if err != nil {
			switch err := err.(type) {
			case errors.GopherError:
				switch err.Errno {
				case errors.ProgramCounterCycled:
					// reached end of memory
					continue
				case errors.InvalidOpcode:
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

		// check validity of instruction result
		err = ir.IsValid()
		if err != nil {
			return err
		}

		// if we've seen this before then finish the disassembly
		if dsm.Program[currentBank][ir.Address] != nil {
			return nil
		}

		// add instruction result to hash table
		dsm.Program[currentBank][ir.Address] = ir
	}
}
