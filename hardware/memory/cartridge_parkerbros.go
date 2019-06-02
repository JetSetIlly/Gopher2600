package memory

import (
	"fmt"
	"gopher2600/errors"
	"io"
)

func fingerprintParkerBros(byts []byte) bool {
	// fingerprint patterns taken from Stella CartDetector.cxx
	for i := 0; i <= len(byts)-3; i++ {
		if (byts[i] == 0x8d && byts[i+1] == 0xe0 && byts[i+2] == 0x1f) ||
			(byts[i] == 0x8d && byts[i+1] == 0xe0 && byts[i+2] == 0x5f) ||
			(byts[i] == 0x8d && byts[i+1] == 0xe9 && byts[i+2] == 0xff) ||
			(byts[i] == 0x0c && byts[i+1] == 0xe0 && byts[i+2] == 0x1f) ||
			(byts[i] == 0xad && byts[i+1] == 0xe0 && byts[i+2] == 0x1f) ||
			(byts[i] == 0xad && byts[i+1] == 0xe9 && byts[i+2] == 0xff) ||
			(byts[i] == 0xad && byts[i+1] == 0xed && byts[i+2] == 0xff) ||
			(byts[i] == 0xad && byts[i+1] == 0xf3 && byts[i+2] == 0xbf) {
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
	//  o each segment can point to one of eight banks in the ROM
	//  o the last segment always points to the last bank
	//  o the other segments can be be pointed to different banks by accessing
	//		specific addresses (see below)
	segment [4]int
}

func newparkerBros(cf io.ReadSeeker) (cartMapper, error) {
	cart := &parkerBros{method: "parker bros. (E0)"}
	cart.initialise()

	cart.banks = make([][]uint8, cart.numBanks())

	cf.Seek(0, io.SeekStart)

	for b := 0; b < cart.numBanks(); b++ {
		// bank sizes are 1028 in the parkerBros format
		cart.banks[b] = make([]uint8, 1024)

		// read cartridge
		n, err := cf.Read(cart.banks[b])
		if err != nil {
			return nil, err
		}
		if n != 1024 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.initialise()

	return cart, nil
}

func (cart parkerBros) String() string {
	return fmt.Sprintf("%s Banks: %d, %d, %d, %d", cart.method, cart.segment[0], cart.segment[1], cart.segment[2], cart.segment[3])
}

func (cart *parkerBros) initialise() {
	cart.segment[0] = 4
	cart.segment[1] = 5
	cart.segment[2] = 6
	cart.segment[3] = 7
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
		cart.bankSwitchAddress(addr)
	}
	return data, nil
}

func (cart *parkerBros) write(addr uint16, data uint8, isPoke bool) error {
	if addr >= 0x0fe0 && addr <= 0x0ff7 {
		cart.bankSwitchAddress(addr)
		return nil
	}
	return errors.NewFormattedError(errors.UnwritableAddress, addr)
}

func (cart *parkerBros) bankSwitchAddress(addr uint16) {
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
	}

	// segment 3 always points to bank 7
}

func (cart parkerBros) numBanks() int {
	return 8
}

func (cart parkerBros) getAddressBank(addr uint16) int {
	if addr >= 0x0000 && addr <= 0x03ff {
		return cart.segment[0]
	} else if addr >= 0x0400 && addr <= 0x07ff {
		return cart.segment[1]
	} else if addr >= 0x0800 && addr <= 0x0bff {
		return cart.segment[2]
	}
	return cart.segment[3]
}

func (cart *parkerBros) setAddressBank(addr uint16, bank int) error {
	if bank < 0 || bank > cart.numBanks() {
		return errors.NewFormattedError(errors.CartridgeError, fmt.Sprintf("invalid bank (%d) for cartridge type (%s)", bank, cart.method))
	}

	if addr >= 0x0000 && addr <= 0x03ff {
		cart.segment[0] = bank
	} else if addr >= 0x0400 && addr <= 0x07ff {
		cart.segment[1] = bank
	} else if addr >= 0x0800 && addr <= 0x0bff {
		cart.segment[2] = bank
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		// segment 4 always points to bank 7
	} else {
		return errors.NewFormattedError(errors.CartridgeError, fmt.Sprintf("invalid address (%d) for cartridge type (%s)", bank, cart.method))
	}

	return nil
}

func (cart *parkerBros) saveBanks() interface{} {
	return cart.segment
}

func (cart *parkerBros) restoreBanks(state interface{}) error {
	cart.segment = state.([4]int)
	return nil
}
