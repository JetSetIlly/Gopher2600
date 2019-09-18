package memory

import (
	"fmt"
	"gopher2600/errors"
	"io"
)

// from bankswitch_sizes.txt:
//
// -3F: Tigervision was the only user of this intresting method.  This works
// in a similar fashion to the above method; however, there are only 4 2K
// segments instead of 4 1K ones, and the ROM image is broken up into 4 2K
// slices.  As before, the last 2K always points to the last 2K of the image.
// You select the desired bank by performing an STA $3F instruction.  The
// accumulator holds the desired bank number (0-3; only the lower two bits are
// used).  Any STA in the $00-$3F range will change banks.  This appears to
// interfere with the TIA addresses, which it does; however you just use $40 to
// $7F instead! :-)  $3F does not have a corresponding TIA register, so writing
// here has no effect other than switching banks.  Very clever; especially
// since you can implement this with only one chip! (a 74LS173)

func fingerprintTigervision(b []byte) bool {
	// tigervision cartridges change banks by writing to memory address 0x3f. we
	// can hypothesize that these types of cartridges will have that instruction
	// sequence "85 3f" many times in a ROM whereas other cartridge types will not

	threshold := 5
	for i := 0; i < len(b)-1; i++ {
		if b[i] == 0x85 && b[i+1] == 0x3f {
			threshold--
		}
		if threshold == 0 {
			return true
		}
	}
	return false
}

type tigervision struct {
	method string
	banks  [][]uint8

	// tigervision cartridges divide memory into two 2k segments
	//  o the last segment always points to the last bank
	//  o the first segment can point to any of the other three
	//
	// the bank pointed to by the first segment is changed through the listen()
	// function (part of the implementation of the cartMapper interface).
	segment [2]int
}

func newTigervision(cf io.ReadSeeker) (cartMapper, error) {
	cart := &tigervision{method: "tigervision (3F)"}

	cart.banks = make([][]uint8, cart.numBanks())

	cf.Seek(0, io.SeekStart)

	for k := 0; k < cart.numBanks(); k++ {
		const bankSize = 2048
		cart.banks[k] = make([]uint8, bankSize)

		// read cartridge
		n, err := cf.Read(cart.banks[k])
		if err != nil {
			return nil, err
		}
		if n != bankSize {
			return nil, errors.New(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.initialise()

	return cart, nil
}

func (cart tigervision) String() string {
	return fmt.Sprintf("%s Banks: %d, %d", cart.method, cart.segment[0], cart.segment[1])
}

func (cart *tigervision) initialise() {
	cart.segment[0] = cart.numBanks() - 2

	// the last segment always points to the last bank
	cart.segment[1] = cart.numBanks() - 1
}

func (cart *tigervision) read(addr uint16) (uint8, error) {
	var data uint8
	if addr >= 0x0000 && addr <= 0x07ff {
		data = cart.banks[cart.segment[0]][addr&0x07ff]
	} else if addr >= 0x0800 && addr <= 0x0fff {
		data = cart.banks[cart.segment[1]][addr&0x07ff]
	}
	return data, nil
}

func (cart *tigervision) write(addr uint16, data uint8) error {
	return errors.New(errors.UnwritableAddress, addr)
}

func (cart *tigervision) numBanks() int {
	return 4 // four banks of 2k
}

func (cart *tigervision) getBank(addr uint16) (bank int) {
	if addr >= 0x0000 && addr <= 0x07ff {
		return cart.segment[0]
	}
	return cart.segment[1]
}

func (cart *tigervision) setBank(addr uint16, bank int) error {
	if bank < 0 || bank > cart.numBanks() {
		return errors.New(errors.CartridgeError, fmt.Sprintf("invalid bank (%d) for cartridge type (%s)", bank, cart.method))
	}

	if addr >= 0x0000 && addr <= 0x07ff {
		cart.segment[0] = bank
	} else if addr >= 0x0800 && addr <= 0x0fff {
		// last segment always points to the last bank
	} else {
		return errors.New(errors.CartridgeError, fmt.Sprintf("invalid address (%d) for cartridge type (%s)", bank, cart.method))
	}

	return nil
}

func (cart *tigervision) saveState() interface{} {
	return cart.segment
}

func (cart *tigervision) restoreState(state interface{}) error {
	cart.segment = state.([len(cart.segment)]int)
	return nil
}

func (cart *tigervision) ram() []uint8 {
	return []uint8{}
}

func (cart *tigervision) listen(addr uint16, data uint8) error {
	// tigervision is seemingly unique in that in bank-switches when an address
	// outside of cartridge space is written to. for this to work, we need the
	// listen() function .

	// althought address 3F is used primarily for bank switching in actual
	// fact writing anywhere in TIA space is okay
	if addr < 0x40 {
		// only the lowest three bits of the data value are used
		cart.segment[0] = int(data & 0x03)
		return nil
	}
	return errors.New(errors.CartridgeListen, addr)
}
