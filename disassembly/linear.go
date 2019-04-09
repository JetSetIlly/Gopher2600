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
// also be deemed to be valid instructions; so lienar disassembly is no good
// for presenting the entire program.

func (dsm *Disassembly) linearDisassembly(mc *cpu.CPU) error {
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		dsm.Cart.BankSwitch(bank)
		for address := dsm.Cart.Origin(); address <= dsm.Cart.Memtop(); address++ {
			mc.PC.Load(address)
			result, _ := mc.ExecuteInstruction(func(*result.Instruction) {})

			// check validity of instruction result and add if it "executed"
			// correctly
			if result != nil && result.IsValid() == nil {
				dsm.linear[bank][address] = result
			}
		}
	}

	return nil
}
