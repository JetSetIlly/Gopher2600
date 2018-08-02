package memory

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory/vcssymbols"
)

// Implementation of CPUBus.Read
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	area.resolvePeriphQueue()

	address &= area.readMask

	// note the name of the register that we are reading
	area.lastReadRegister = vcssymbols.ReadSymbols[address]

	sym := vcssymbols.ReadSymbols[address]
	if sym == "" {
		// silently ignore illegal reads (we're definitely reading from the correct
		// memory space but some registers are not readable)
		//
		// TODO: add a GopherError that can be ignored or noted as appropriate
		// for the application
		return 0, nil
	}

	return area.memory[address-area.origin], nil
}

// Implementation of CPUBus.Write
func (area *ChipMemory) Write(address uint16, data uint8) error {
	area.resolvePeriphQueue()

	// check that the last write to this memory area has been serviced
	if area.writeSignal {
		return errors.GopherError{Errno: errors.UnservicedChipWrite, Values: errors.Values{vcssymbols.WriteSymbols[area.lastWriteAddress]}}
	}

	sym := vcssymbols.WriteSymbols[address]
	if sym == "" {
		// silently ignore illegal writes (we're definitely writing to the correct
		// memory space but some registers are not writable)
		return nil
	}

	// note address of write
	area.lastWriteAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}
