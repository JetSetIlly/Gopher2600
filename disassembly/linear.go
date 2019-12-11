package disassembly

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
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

			// check validity of instruction result and add if it "executed"
			// correctly
			if mc.LastResult.IsValid() == nil {
				dsm.linear[bank][address&disasmMask] = Entry{
					style:                 result.StyleBrief,
					instruction:           mc.LastResult.GetString(dsm.Symtable, result.StyleBrief),
					instructionDefinition: mc.LastResult.Defn}
			}
		}
	}

	return nil
}
