package memorymap

// Type representing the different areas of memory
type Area int

func (a Area) String() string {
	switch a {
	case TIA:
		return "TIA"
	case RAM:
		return "RAM"
	case RIOT:
		return "RIOT"
	case Cartridge:
		return "Cartridge"
	}

	return "undefined"
}

// The different memory areas in the VCS
const (
	Undefined Area = iota
	TIA
	RAM
	RIOT
	Cartridge
)

// The origin and memory top for each ares of memory
const (
	OriginTIA  = uint16(0x0000)
	MemtopTIA  = uint16(0x003f)
	OriginRAM  = uint16(0x0080)
	MemtopRAM  = uint16(0x00ff)
	OriginRIOT = uint16(0x0280)
	MemtopRIOT = uint16(0x0297)
	OriginCart = uint16(0x1000)
	MemtopCart = uint16(0x1fff)
)

// Memtop is the top most address of memory in the VCS. It is the same as the
// cartridge memtop.
const Memtop = uint16(0x1fff)

// Adressess in the RIOT and TIA areas that are being used to read from from
// memory require an additional transformation
const (
	AddressMaskRIOT = uint16(0x02f7)
	AddressMaskTIA  = uint16(0x000f)
)

// MapAddress translates the address argument from mirror space to primary
// space.  Generally, an address should be passed through this function before
// accessing memory.
func MapAddress(address uint16, read bool) (uint16, Area) {
	// note that the order of these filters is important

	// cartridge addresses
	if address&OriginCart == OriginCart {
		return address & MemtopCart, Cartridge
	}

	// RIOT addresses
	if address&OriginRIOT == OriginRIOT {
		if read {
			return address & MemtopRIOT & AddressMaskRIOT, RIOT
		}
		return address & MemtopRIOT, RIOT
	}

	// RAM addresses
	if address&OriginRAM == OriginRAM {
		return address & MemtopRAM, RAM
	}

	// everything else is in TIA space
	if read {
		return address & MemtopTIA & AddressMaskTIA, TIA
	}

	return address & MemtopTIA, TIA
}
