package disassembly

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
)

// linearDisassembly decodes every possible address in the cartridge. if the
// "execution" of the address succeeds it is stored in the linear table.
//
// compared to flowDisassembly, this form of disassembly takes into account
// areas of the cartridge that are unreachable when simply looking at where
// flow constructs take us. for instance, calling RTS with manually stacked
// return addresses are undetectable with flowDisassembly but linearDisassembly
// doesn't mind. self modifying code is still invisible.
//
// the downside of this method is that a lot of addresses in data segments will
// also be deemed to be valid instructions; so linear disassembly is no good
// for presenting the entire program.

func (dsm *Disassembly) linearDisassembly(mc *cpu.CPU) error {
	for bank := 0; bank < len(dsm.linear); bank++ {
		for address := dsm.Cart.Origin(); address <= dsm.Cart.Memtop(); address++ {
			if err := dsm.Cart.SetBank(address, bank); err != nil {
				return err
			}

			mc.PC.Load(address)

			// deliberately ignoring errors
			r, _ := mc.ExecuteInstruction(func(*result.Instruction) error { return nil })

			// check validity of instruction result and add if it "executed"
			// correctly
			if r != nil && r.IsValid() == nil {
				dsm.linear[bank][address&disasmMask] = Entry{
					style:                 result.StyleBrief,
					instruction:           r.GetString(dsm.Symtable, result.StyleBrief),
					instructionDefinition: r.Defn}
			}
		}
	}

	return nil
}
