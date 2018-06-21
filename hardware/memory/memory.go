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

// ChipBus defines the operations for the memory system when accessed from the
// VCS chips (TIA, RIOT). Only ChipMemory implements this interface.
type ChipBus interface {
	ChipRead() (bool, string, uint8)
	ChipWrite(registerName string, data uint8)
	LastReadRegister() string
}

// Area defines the meta-operations for all memory areas. Think of these
// functions as "debugging" functions, that is operations outside of the normal
// operation of the machine. We also use this interface as the "generic" type
// when we need to store collections of different types of memory areas (see
// VCSMemory.memmap)
type Area interface {
	Label() string
	Origin() uint16
	Memtop() uint16

	// peek returns value, label (if any), and any error (which may not be fatal)
	Peek(address uint16) (uint8, string, error)
}

// AreaInfo provides the basic info needed to define a memory area. All memory
// areas embed AreaInfo alongside the implementation of the Area interface
type AreaInfo struct {
	label  string
	origin uint16
	memtop uint16
}
