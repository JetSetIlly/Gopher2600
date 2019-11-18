package memorymap

// Area type
type Area int

func (a Area) String() string {
	switch a {
	case TIA:
		return "TIA"
	case PIA:
		return "PIA"
	case RIOT:
		return "RIOT"
	case Cartridge:
		return "Cartridge"
	}

	return "undefined"
}

// list of memory areas
const (
	Undefined Area = iota
	TIA
	PIA
	RIOT
	Cartridge
)

// the origin and memory top for each are of the VCS memory
const (
	OriginTIA  = uint16(0x0000)
	MemtopTIA  = uint16(0x003f)
	OriginPIA  = uint16(0x0080)
	MemtopPIA  = uint16(0x00ff)
	OriginRIOT = uint16(0x0280)
	MemtopRIOT = uint16(0x0297)
	OriginCart = uint16(0x1000)
	MemtopCart = uint16(0x1fff)
)

// when reading addresses from memory, TIA and RIOT addresses are filtered down
// even further
const (
	AddressMaskRIOT = uint16(0x02f7)
	AddressMaskTIA  = uint16(0x000f)
)

// MapAddress translates the quoted address from mirror space to primary space.
// Generally, an address should be passed through this function before
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

	// PIA RAM addresses
	if address&OriginPIA == OriginPIA {
		return address & MemtopPIA, PIA
	}

	// everything else is in TIA space
	if read {
		return address & MemtopTIA & AddressMaskTIA, TIA
	}

	return address & MemtopTIA, TIA
}
