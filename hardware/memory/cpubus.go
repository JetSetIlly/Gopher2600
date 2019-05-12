package memory

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
)

// Read is an implementation of CPUBus. returns the value and/or error
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	// note the name of the register that we are reading
	area.lastReadRegister = addresses.Read[address]

	sym := addresses.Read[address]
	if sym == "" {
		return 0, errors.NewFormattedError(errors.UnreadableAddress, address)
	}

	return area.memory[area.origin|address^area.origin], nil
}

// Write is an implementation of CPUBus. it writes the data to the memory
// area's address
func (area *ChipMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory area has been serviced
	if area.writeSignal {
		return errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("unserviced write to chip memory (%s)", addresses.Write[area.lastWriteAddress]))
	}

	sym := addresses.Write[address]
	if sym == "" {
		return errors.NewFormattedError(errors.UnwritableAddress, address)
	}

	// note address of write
	area.lastWriteAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}
