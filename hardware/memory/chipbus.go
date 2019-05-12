package memory

import "gopher2600/hardware/memory/addresses"

// ChipRead is an implementation of ChipBus. returns:
// - whether a chip was last written to
// - the CPU name of the address that was written to
// - the written value
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	if area.writeSignal {
		area.writeSignal = false
		return true, addresses.Write[area.lastWriteAddress], area.writeData
	}

	return false, "", 0
}

// ChipWrite is an implementation of ChipBus. it writes the data to the memory
// area's address
func (area *ChipMemory) ChipWrite(address uint16, data uint8) {
	area.memory[address] = data
}

// LastReadRegister is an implementation of ChipBus. it returns the register
// name of the last memory location *read* by the CPU
func (area *ChipMemory) LastReadRegister() string {
	r := area.lastReadRegister
	area.lastReadRegister = ""
	return r
}
