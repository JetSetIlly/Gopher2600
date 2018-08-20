package memory

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory/vcssymbols"
)

// Implementation of CPUBus.Read
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	area.resolvePeriphQueue()

	// note the name of the register that we are reading
	area.lastReadRegister = vcssymbols.ReadSymbols[address]

	sym := vcssymbols.ReadSymbols[address]
	if sym == "" {
		return 0, errors.NewGopherError(errors.UnreadableAddress, address)
	}

	return area.memory[address-area.origin], nil
}

// Implementation of CPUBus.Write
func (area *ChipMemory) Write(address uint16, data uint8) error {
	area.resolvePeriphQueue()

	// check that the last write to this memory area has been serviced
	if area.writeSignal {
		return errors.NewGopherError(errors.UnservicedChipWrite, vcssymbols.WriteSymbols[area.lastWriteAddress])
	}

	sym := vcssymbols.WriteSymbols[address]
	if sym == "" {
		return errors.NewGopherError(errors.UnwritableAddress, address)
	}

	// note address of write
	area.lastWriteAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}
