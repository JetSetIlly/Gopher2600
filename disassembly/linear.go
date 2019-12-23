package disassembly

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) linearDisassembly(mc *cpu.CPU) error {
	for bank := 0; bank < len(dsm.linear); bank++ {
		for address := memorymap.OriginCart; address <= memorymap.MemtopCart; address++ {
			if err := dsm.cart.SetBank(address, bank); err != nil {
				return err
			}

			mc.PC.Load(address)

			// deliberately ignoring errors
			_ = mc.ExecuteInstruction(nil)

			// continue for loop on invalid results. we don't want to be as
			// discerning as in flowDisassembly(). the nature of
			// linearDisassembly() means that we're likely to try executing
			// invalid instructions. best just to ignore such errors.
			if mc.LastResult.IsValid() != nil {
				continue // for loop
			}

			ent, err := dsm.FormatResult(mc.LastResult)
			if err != nil {
				return err
			}

			dsm.linear[bank][address&disasmMask] = ent
		}
	}

	return nil
}
