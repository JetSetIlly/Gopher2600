// Package bus defines the memory bus concept. For an explanation see the
// memory package documentation.
package bus

// CPUBus defines the operations for the memory system when accessed from the CPU
// All memory areas implement this interface because they are all accessible
// from the CPU (compare to ChipBus). The VCSMemory type also implements this
// interface and maps the read/write address to the correct memory area --
// meaning that CPU access need not care which part of memory it is writing to
type CPUBus interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

// ChipData is returned by ChipBus.ChipRead()
type ChipData struct {
	// the canonical name of the chip register writter to
	Name string

	// the data value written to the chip register
	Value uint8
}

// ChipBus defines the operations for the memory system when accessed from the
// VCS chips (TIA, RIOT). Only ChipMemory implements this interface.
type ChipBus interface {
	// ChipRead checks to see if the chip's memory area has been written to. if
	// it has the function returns true and an instance of ChipData
	ChipRead() (bool, ChipData)

	// ChipWrite writes the data to the chip memory
	ChipWrite(address uint16, data uint8)

	// LastReadRegister returns the register name of the last memory location
	// *read* by the CPU
	LastReadRegister() string
}

// InputDeviceBus defines the operations for the memory system when accessed from
// parts of the emulation are peripheral to the operation of the machine. In
// practice, this includes the front panel in addition to joysticks, etc.
type InputDeviceBus interface {
	InputDeviceWrite(address uint16, data uint8, mask uint8)
}

// DebuggerBus defines the meta-operations for all memory areas. Think of these
// functions as "debugging" functions, that is operations outside of the normal
// operation of the machine.
type DebuggerBus interface {
	Peek(address uint16) (uint8, error)
	Poke(address uint16, value uint8) error
}
