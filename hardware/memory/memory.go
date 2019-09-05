package memory

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
	Name  string
	Value uint8
}

// ChipBus defines the operations for the memory system when accessed from the
// VCS chips (TIA, RIOT). Only ChipMemory implements this interface.
type ChipBus interface {
	ChipRead() (bool, ChipData)
	ChipWrite(address uint16, data uint8)
	LastReadRegister() string
}

// PeriphBus defines the operations for the memory system when accessed from
// parts of the emulation are peripheral to the operation of the machine. In
// practice, this includes the front panel in addition to joysticks, etc.
type PeriphBus interface {
	PeriphWrite(address uint16, data uint8, mask uint8)
}

// DebuggerBus defines the meta-operations for all memory areas. Think of these
// functions as "debugging" functions, that is operations outside of the normal
// operation of the machine.
type DebuggerBus interface {
	Label() string
	Origin() uint16
	Memtop() uint16
	Peek(address uint16) (uint8, error)
	Poke(address uint16, value uint8) error
}
