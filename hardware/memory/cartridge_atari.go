package memory

import (
	"fmt"
	"gopher2600/errors"
	"io"
)

// these atari* types implement the cartMapper interface

type atari struct {
	method string
	memory [][]uint8
	bank   int

	extraRAM []uint8
}

func (cart atari) String() string {
	return cart.method
}

func (cart *atari) initialise() {
	cart.bank = 0
}

func (cart atari) addressBank(addr uint16) int {
	return cart.bank
}

func (cart *atari) saveState() interface{} {
	return cart.bank
}

func (cart *atari) restoreState(state interface{}) error {
	cart.bank = state.(int)
	return nil
}

func (cart *atari) read(addr uint16) (uint8, bool) {
	if cart.extraRAM != nil {
		if addr > 127 && addr < 256 {
			return cart.extraRAM[addr-128], true
		}
	}
	return 0, false
}

func (cart *atari) write(addr uint16, data uint8) bool {
	if cart.extraRAM != nil {
		if addr <= 127 {
			cart.extraRAM[addr] = data
			return true
		}
	}
	return false
}

func (cart *atari) addCartridgeRAM() bool {
	// check for cartridge memory:
	//  - this method of detection simply checks whether the first 256 of each
	// bank are empty
	//  - I've guessed that this is a good method. if there's another one I
	// don't know about it.
	nullChar := cart.memory[0][0]
	for b := 0; b < len(cart.memory); b++ {
		for a := 0; a < 256; a++ {
			if cart.memory[b][a] != nullChar {
				return false
			}
		}
	}

	// allocate RAM
	cart.extraRAM = make([]uint8, 128)

	// update method string
	cart.method = fmt.Sprintf("%s (inc. extra RAM)", cart.method)

	return true
}

// atari4k is the original and most straightforward format
//  o Pitfall
//  o River Raid
//  o Barnstormer
//  o etc.
type atari4k struct {
	atari
}

// this is a regular cartridge of 4096 bytes
//  o Pitfall
//  o Adventure
//  o Yars Revenge
//  o etc.
func newAtari4k(cf io.ReadSeeker) (*atari4k, error) {
	cart := &atari4k{}

	cart.method = "atari 4k"
	cart.memory = make([][]uint8, 1)
	cart.memory[0] = make([]uint8, 4096)

	if cf != nil {
		cf.Seek(0, io.SeekStart)

		// read cartridge
		n, err := cf.Read(cart.memory[0])
		if err != nil {
			return nil, err
		}
		if n != 4096 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.addCartridgeRAM()

	return cart, nil
}

func (cart atari4k) numBanks() int {
	return 1
}

func (cart *atari4k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}
	return cart.memory[0][addr], nil
}

func (cart *atari4k) write(addr uint16, data uint8, isPoke bool) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if isPoke {
		return errors.NewFormattedError(errors.UnpokeableAddress, addr)
	}

	return errors.NewFormattedError(errors.UnwritableAddress, addr)
}

// atari2k is the half-size cartridge of 2048 bytes
//	o Combat
//  o Dragster
//  o Outlaw
//	o Surround
//  o early cartridges
type atari2k struct {
	atari
}

func newAtari2k(cf io.ReadSeeker) (*atari2k, error) {
	cart := &atari2k{}

	cart.method = "atari 2k"
	cart.memory = make([][]uint8, 1)
	cart.memory[0] = make([]uint8, 2048)

	if cf != nil {
		cf.Seek(0, io.SeekStart)

		// read cartridge
		n, err := cf.Read(cart.memory[0])
		if err != nil {
			return nil, err
		}
		if n != 2048 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.addCartridgeRAM()

	return cart, nil
}

func (cart atari2k) numBanks() int {
	return 1
}

func (cart *atari2k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}
	return cart.memory[0][addr&0x07ff], nil
}

func (cart *atari2k) write(addr uint16, data uint8, isPoke bool) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if isPoke {
		return errors.NewFormattedError(errors.UnpokeableAddress, addr)
	}

	return errors.NewFormattedError(errors.UnwritableAddress, addr)
}

// atari8k (F8)
//	o ET
//  o Krull
//  o etc.
type atari8k struct {
	atari
}

func newAtari8k(cf io.ReadSeeker) (cartMapper, error) {
	cart := &atari8k{}

	cart.method = "atari 8k (F8)"
	cart.memory = make([][]uint8, cart.numBanks())

	cf.Seek(0, io.SeekStart)

	for b := 0; b < cart.numBanks(); b++ {
		cart.memory[b] = make([]uint8, 4096)

		// read cartridge
		n, err := cf.Read(cart.memory[b])
		if err != nil {
			return nil, err
		}
		if n != 4096 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.addCartridgeRAM()

	return cart, nil
}

func (cart atari8k) numBanks() int {
	return 2
}

func (cart *atari8k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.memory[cart.bank][addr]

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	}

	return data, nil
}

func (cart *atari8k) write(addr uint16, data uint8, isPoke bool) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if isPoke {
		return errors.NewFormattedError(errors.UnpokeableAddress, addr)
	}

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else {
		return errors.NewFormattedError(errors.UnwritableAddress, addr)
	}

	return nil
}

// atari16k (F6)
//	o Crystal Castle
//	o RS Boxing
//  o Midnite Magic
//  o etc.
type atari16k struct {
	atari
}

func newAtari16k(cf io.ReadSeeker) (*atari16k, error) {
	cart := &atari16k{}

	cart.method = "atari 16k (F6)"
	cart.memory = make([][]uint8, cart.numBanks())

	cf.Seek(0, io.SeekStart)

	for b := 0; b < cart.numBanks(); b++ {
		cart.memory[b] = make([]uint8, 4096)

		// read cartridge
		n, err := cf.Read(cart.memory[b])
		if err != nil {
			return nil, err
		}
		if n != 4096 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.addCartridgeRAM()

	return cart, nil
}

func (cart atari16k) numBanks() int {
	return 4
}

func (cart *atari16k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.memory[cart.bank][addr]

	if addr == 0x0ff6 {
		cart.bank = 0
	} else if addr == 0x0ff7 {
		cart.bank = 1
	} else if addr == 0x0ff8 {
		cart.bank = 2
	} else if addr == 0x0ff9 {
		cart.bank = 3
	}

	return data, nil
}

func (cart *atari16k) write(addr uint16, data uint8, isPoke bool) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if isPoke {
		return errors.NewFormattedError(errors.UnpokeableAddress, addr)
	}

	if addr == 0x0ff6 {
		cart.bank = 0
	} else if addr == 0x0ff7 {
		cart.bank = 1
	} else if addr == 0x0ff8 {
		cart.bank = 2
	} else if addr == 0x0ff9 {
		cart.bank = 3
	} else {
		return errors.NewFormattedError(errors.UnwritableAddress, addr)
	}

	return nil
}

// atari32k (F8)
// o Fatal Run
// o Super Mario Bros.
// o Donkey Kong (homebrew)
// o etc.
type atari32k struct {
	atari
}

func newAtari32k(cf io.ReadSeeker) (*atari32k, error) {
	cart := &atari32k{}

	cart.method = "atari 32k (F4)"
	cart.memory = make([][]uint8, cart.numBanks())

	cf.Seek(0, io.SeekStart)

	for b := 0; b < cart.numBanks(); b++ {
		cart.memory[b] = make([]uint8, 4096)

		// read cartridge
		n, err := cf.Read(cart.memory[b])
		if err != nil {
			return nil, err
		}
		if n != 4096 {
			return nil, errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
		}
	}

	cart.addCartridgeRAM()

	return cart, nil
}

func (cart atari32k) numBanks() int {
	return 8
}

func (cart *atari32k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.memory[cart.bank][addr]

	if addr == 0x0ff4 {
		cart.bank = 0
	} else if addr == 0x0ff5 {
		cart.bank = 1
	} else if addr == 0x0ff6 {
		cart.bank = 2
	} else if addr == 0x0ff7 {
		cart.bank = 3
	} else if addr == 0x0ff8 {
		cart.bank = 4
	} else if addr == 0x0ff9 {
		cart.bank = 5
	} else if addr == 0x0ffa {
		cart.bank = 6
	} else if addr == 0x0ffb {
		cart.bank = 7
	}

	return data, nil
}

func (cart *atari32k) write(addr uint16, data uint8, isPoke bool) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if isPoke {
		return errors.NewFormattedError(errors.UnpokeableAddress, addr)
	}

	if addr == 0x0ff4 {
		cart.bank = 0
	} else if addr == 0x0ff5 {
		cart.bank = 1
	} else if addr == 0x0ff6 {
		cart.bank = 2
	} else if addr == 0x0ff7 {
		cart.bank = 3
	} else if addr == 0x0ff8 {
		cart.bank = 4
	} else if addr == 0x0ff9 {
		cart.bank = 5
	} else if addr == 0x0ffa {
		cart.bank = 6
	} else if addr == 0x0ffb {
		cart.bank = 7
	} else {
		return errors.NewFormattedError(errors.UnwritableAddress, addr)
	}

	return nil
}
