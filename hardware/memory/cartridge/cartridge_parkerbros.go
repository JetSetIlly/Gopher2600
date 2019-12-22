package cartridge

import (
	"fmt"
	"gopher2600/errors"
)

// from bankswitch_sizes.txt:
//
// -E0: Parker Brothers was the main user of this method.  This cart is
// segmented into 4 1K segments.  Each segment can point to one 1K slice of the
// ROM image.  You select the desired 1K slice by accessing 1FE0 to 1FE7 for
// the first 1K (1FE0 selects slice 0, 1FE1 selects slice 1, etc).  1FE8 to
// 1FEF selects the slice for the second 1K, and 1FF0 to 1FF8 selects the slice
// for the third 1K.  The last 1K always points to the last 1K of the ROM image
// so that the cart always starts up in the exact same place.

func fingerprintParkerBros(b []byte) bool {
	// fingerprint patterns taken from Stella CartDetector.cxx
	for i := 0; i <= len(b)-3; i++ {
		if (b[i] == 0x8d && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0x8d && b[i+1] == 0xe0 && b[i+2] == 0x5f) ||
			(b[i] == 0x8d && b[i+1] == 0xe9 && b[i+2] == 0xff) ||
			(b[i] == 0x0c && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0xad && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0xad && b[i+1] == 0xe9 && b[i+2] == 0xff) ||
			(b[i] == 0xad && b[i+1] == 0xed && b[i+2] == 0xff) ||
			(b[i] == 0xad && b[i+1] == 0xf3 && b[i+2] == 0xbf) {
			return true
		}

	}

	return false
}

// parkerBros implements the cartMapper interface.
//  o Montezuma's Revenge
//  o Lord of the Rings
//  o etc.
type parkerBros struct {
	method string
	banks  [][]uint8

	// parker bros. cartridges divide memory into 4 segments
	//  o the last segment always points to the last bank
	//  o the other segments can point to any one of the eight banks in the ROM
	//		(including the last bank)
	//
	// switching of segments is performed by the bankSwitchOnAccess() function
	segment [4]int
}

func newparkerBros(data []byte) (cartMapper, error) {
	const bankSize = 1024

	cart := &parkerBros{}
	cart.method = "parker bros. (E0)"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, "not enough bytes in the cartridge file")
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		copy(cart.banks[k], data[offset:offset+bankSize])
	}

	cart.initialise()

	return cart, nil
}

func (cart parkerBros) String() string {
	return fmt.Sprintf("%s Banks: %d, %d, %d, %d", cart.method, cart.segment[0], cart.segment[1], cart.segment[2], cart.segment[3])
}

func (cart *parkerBros) initialise() {
	cart.segment[0] = cart.numBanks() - 4
	cart.segment[1] = cart.numBanks() - 3
	cart.segment[2] = cart.numBanks() - 2
	cart.segment[3] = cart.numBanks() - 1
}

func (cart *parkerBros) read(addr uint16) (uint8, error) {
	var data uint8
	if addr >= 0x0000 && addr <= 0x03ff {
		data = cart.banks[cart.segment[0]][addr&0x03ff]
	} else if addr >= 0x0400 && addr <= 0x07ff {
		data = cart.banks[cart.segment[1]][addr&0x03ff]
	} else if addr >= 0x0800 && addr <= 0x0bff {
		data = cart.banks[cart.segment[2]][addr&0x03ff]
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		data = cart.banks[cart.segment[3]][addr&0x03ff]
		cart.bankSwitchOnAccess(addr)
	}
	return data, nil
}

func (cart *parkerBros) write(addr uint16, data uint8) error {
	if cart.bankSwitchOnAccess(addr) {
		return nil
	}
	return errors.New(errors.UnwritableAddress, addr)
}

func (cart *parkerBros) bankSwitchOnAccess(addr uint16) bool {
	switch addr {
	// segment 0
	case 0x0fe0:
		cart.segment[0] = 0
	case 0x0fe1:
		cart.segment[0] = 1
	case 0x0fe2:
		cart.segment[0] = 2
	case 0x0fe3:
		cart.segment[0] = 3
	case 0x0fe4:
		cart.segment[0] = 4
	case 0x0fe5:
		cart.segment[0] = 5
	case 0x0fe6:
		cart.segment[0] = 6
	case 0x0fe7:
		cart.segment[0] = 7

	// segment 1
	case 0x0fe8:
		cart.segment[1] = 0
	case 0x0fe9:
		cart.segment[1] = 1
	case 0x0fea:
		cart.segment[1] = 2
	case 0x0feb:
		cart.segment[1] = 3
	case 0x0fec:
		cart.segment[1] = 4
	case 0x0fed:
		cart.segment[1] = 5
	case 0x0fee:
		cart.segment[1] = 6
	case 0x0fef:
		cart.segment[1] = 7

	// segment 2
	case 0x0ff0:
		cart.segment[2] = 0
	case 0x0ff1:
		cart.segment[2] = 1
	case 0x0ff2:
		cart.segment[2] = 2
	case 0x0ff3:
		cart.segment[2] = 3
	case 0x0ff4:
		cart.segment[2] = 4
	case 0x0ff5:
		cart.segment[2] = 5
	case 0x0ff6:
		cart.segment[2] = 6
	case 0x0ff7:
		cart.segment[2] = 7

	// segment 3 always points to bank 7

	default:
		return false
	}

	return true
}

func (cart parkerBros) numBanks() int {
	return 8
}

func (cart parkerBros) getBank(addr uint16) int {
	if addr >= 0x0000 && addr <= 0x03ff {
		return cart.segment[0]
	} else if addr >= 0x0400 && addr <= 0x07ff {
		return cart.segment[1]
	} else if addr >= 0x0800 && addr <= 0x0bff {
		return cart.segment[2]
	}
	return cart.segment[3]
}

func (cart *parkerBros) setBank(addr uint16, bank int) error {
	if bank < 0 || bank > cart.numBanks() {
		return errors.New(errors.CartridgeError, fmt.Sprintf("invalid bank (%d) for cartridge type (%s)", bank, cart.method))
	}

	if addr >= 0x0000 && addr <= 0x03ff {
		cart.segment[0] = bank
	} else if addr >= 0x0400 && addr <= 0x07ff {
		cart.segment[1] = bank
	} else if addr >= 0x0800 && addr <= 0x0bff {
		cart.segment[2] = bank
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		// last segment always points to the last bank
	} else {
		return errors.New(errors.CartridgeError, fmt.Sprintf("invalid address (%d) for cartridge type (%s)", bank, cart.method))
	}

	return nil
}

func (cart *parkerBros) saveState() interface{} {
	return cart.segment
}

func (cart *parkerBros) restoreState(state interface{}) error {
	cart.segment = state.([len(cart.segment)]int)
	return nil
}

func (cart parkerBros) ram() []uint8 {
	return []uint8{}
}

func (cart *parkerBros) listen(addr uint16, data uint8) {
}

func (cart *parkerBros) poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *parkerBros) patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.method)
}
