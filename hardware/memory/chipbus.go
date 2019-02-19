package memory

import "gopher2600/hardware/memory/vcssymbols"

// ChipRead is an implementation of ChipBus.ChipRead. returns:
// - whether a chip was last written to
// - the CPU name of the address that was written to
// - the written value
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	area.resolvePeriphQueue()

	if area.writeSignal {
		area.writeSignal = false
		return true, vcssymbols.WriteSymbols[area.lastWriteAddress], area.writeData
	}

	return false, "", 0
}

// ChipWrite is an implementation of ChipBus.ChipWrite.  it writes the data to
// the memory area's address specified by registerName
func (area *ChipMemory) ChipWrite(address uint16, data uint8) {
	area.memory[address] = data
}

// LastReadRegister is an implementation of ChipBus.LastReadRegister. it
// returns the register name of the last memory location *read* by the CPU
func (area *ChipMemory) LastReadRegister() string {
	r := area.lastReadRegister
	area.lastReadRegister = ""
	return r
}
