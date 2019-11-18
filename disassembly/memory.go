package disassembly

import (
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/memorymap"
)

// disasmMemory is a simplified memory model that allows the emulated CPU to
// read cartridge memory.
type disasmMemory struct {
	cart *memory.Cartridge
}

func (dismem *disasmMemory) Read(address uint16) (uint8, error) {
	// map address
	if address&memorymap.OriginCart == memorymap.OriginCart {
		address = address & memorymap.MemtopCart
		return dismem.cart.Read(address)
	}

	// address outside of cartidge range return nothing
	return 0, nil
}

func (dismem *disasmMemory) Write(address uint16, data uint8) error {
	// map address
	if address&memorymap.OriginCart == memorymap.OriginCart {
		address = address & memorymap.MemtopCart
		return dismem.cart.Write(address, data)
	}

	// address outside of cartidge range return nothing
	return nil
}
