package disassembly

import "gopher2600/hardware/memory"

type minimalMemory struct {
	cart *memory.Cartridge

	// origin and memtop of cartridge space (for convenience)
	cartOrigin uint16
	cartMemtop uint16
}

// newMinimalMemory is the preferred method of initialisation for minimalMemory
func newMinimalMemory(cart *memory.Cartridge) (*minimalMemory, error) {
	mem := new(minimalMemory)
	mem.cart = cart
	mem.cartOrigin = mem.cart.Origin()
	mem.cartMemtop = mem.cart.Memtop()
	return mem, nil
}

func (mem *minimalMemory) Read(address uint16) (uint8, error) {
	// map address
	if address&mem.cartOrigin == mem.cartOrigin {
		address = address & mem.cartMemtop
		return mem.cart.Read(address)
	}

	return 0, nil
}

func (mem *minimalMemory) Write(address uint16, data uint8) error {
	// map address
	if address&mem.cartOrigin == mem.cartOrigin {
		address = address & mem.cartMemtop
		return mem.cart.Write(address, data)
	}

	return nil
}
