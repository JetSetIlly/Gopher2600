package memory

// PeriphWrite implements PeriphBus. it writes the data to the memory area's
// address
func (area *ChipMemory) PeriphWrite(address uint16, data uint8) {
	area.memory[address] = data
}
