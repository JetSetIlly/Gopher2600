package disassembly

// disassembly.memory is a simplified memory model that allows the emulated CPU
// to read cartridge memory.

import (
	"gopher2600/hardware/memory"
)

type disasmMemory struct {
	cart *memory.Cartridge

	// origin and memtop of cartridge space (for convenience)
	cartOrigin uint16
	cartMemtop uint16
}

// newDisasmMemory is the preferred method of initialisation for disasmMemory
func newDisasmMemory(cart *memory.Cartridge) (*disasmMemory, error) {
	mem := new(disasmMemory)
	mem.cart = cart
	mem.cartOrigin = mem.cart.Origin()
	mem.cartMemtop = mem.cart.Memtop()
	return mem, nil
}

func (mem *disasmMemory) Read(address uint16) (uint8, error) {
	// map address
	if address&mem.cartOrigin == mem.cartOrigin {
		address = address & mem.cartMemtop
		return mem.cart.Read(address)
	}

	// address outside of cartidge range return nothing
	return 0, nil
}

func (mem *disasmMemory) Write(address uint16, data uint8) error {
	// map address
	if address&mem.cartOrigin == mem.cartOrigin {
		address = address & mem.cartMemtop
		return mem.cart.Write(address, data)
	}

	// address outside of cartidge range return nothing
	return nil
}
