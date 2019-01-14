package memory

import (
	"fmt"
	"gopher2600/errors"
	"os"
)

// AddressReset is the address where the reset address is stored
// - used by VCS.Reset() and Disassembly module
const AddressReset = uint16(0xfffc)

// AddressIRQ is the address where the interrupt address is stored
const AddressIRQ = 0xfffe

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	CPUBus
	Area
	AreaInfo

	bank   int
	memory [][]uint8

	// readHook allows custom read routines depending on the cartridge type
	// that has been inserted. note that the address has been normalised so that
	// the origin is at 0 bytes
	readHook func(uint16) uint8
}

// newCart is the preferred method of initialisation for the cartridges
func newCart() *Cartridge {
	cart := new(Cartridge)
	cart.label = "Cartridge"
	cart.origin = 0x1000
	cart.memtop = 0x1fff
	return cart
}

// Label is an implementation of Area.Label
func (cart Cartridge) Label() string {
	return cart.label
}

// Origin is an implementation of Area.Origin
func (cart Cartridge) Origin() uint16 {
	return cart.origin
}

// Memtop is an implementation of Area.Memtop
func (cart Cartridge) Memtop() uint16 {
	return cart.memtop
}

// Clear is an implementation of CPUBus.Clear
func (cart *Cartridge) Clear() {
	// clearing cartridge memory is semantically the same as ejecting the cartridge
	cart.Eject()
}

// Implementation of CPUBus.Read
func (cart Cartridge) Read(address uint16) (uint8, error) {
	if len(cart.memory) == 0 {
		return 0, errors.NewGopherError(errors.CartridgeMissing)
	}
	return cart.readHook(cart.origin | address ^ cart.origin), nil
}

// Implementation of CPUBus.Write
func (cart *Cartridge) Write(address uint16, data uint8) error {
	return errors.NewGopherError(errors.UnwritableAddress, address)
}

// allocateCartridgeSpace is a generalised allocation of memory and file
// reading routine. common to all cartridge sizes
func (cart *Cartridge) allocateCartridgeSpace(file *os.File, numberOfBanks int) error {
	// allocate enough memory for new cartridge
	cart.memory = make([][]uint8, numberOfBanks)

	for b := 0; b < numberOfBanks; b++ {
		cart.memory[b] = make([]uint8, 4096)

		if file != nil {
			// read cartridge
			n, err := file.Read(cart.memory[b])
			if err != nil {
				return err
			}
			if n != 4096 {
				return errors.NewGopherError(errors.CartridgeFileError, errors.FileTruncated)
			}
		}
	}

	return nil
}

// Attach loads the bytes from a cartridge (represented by 'filename')
func (cart *Cartridge) Attach(filename string) error {
	cf, err := os.Open(filename)
	if err != nil {
		return errors.NewGopherError(errors.CartridgeFileError, err)
	}
	defer func() {
		_ = cf.Close()
	}()

	// get file info
	cfi, err := cf.Stat()
	if err != nil {
		return err
	}

	// set null read hook
	cart.readHook = func(uint16) uint8 { return 0 }
	cart.bank = 0

	// how cartridges are mapped into the 4k space can differs dramatically.
	// the following implementation details have been cribbed from Kevin
	// Horton's "Cart Information" document [sizes.txt]

	switch cfi.Size() {
	case 2048:
		// this is a half-size cartridge of 2048 bytes
		//
		//	o Combat
		//  o Dragster
		//  o Outlaw
		//	o Surround
		//  o mostly early cartridges

		// note that while we're allocating a full 4096 bytes for this
		// cartrdige size, the readHook below ensures that we only ever read
		// the first 2048.
		cart.allocateCartridgeSpace(cf, 1)

		cart.readHook = func(address uint16) uint8 {
			// because we've only allocated half the amount of memory that
			// should be there, we need a further mask to make sure the address
			// is in range
			return cart.memory[0][address&0x07ff]
		}

	case 4096:
		// this is a regular cartridge of 4096 bytes
		//
		//  o Pitfall
		//  o Adventure
		//  o Yars Revenge
		//  o most 2600 cartridges...

		cart.allocateCartridgeSpace(cf, 1)
		cart.readHook = func(address uint16) uint8 { return cart.memory[0][address] }

	case 8192:
		cart.allocateCartridgeSpace(cf, 2)

		// TODO: differentiation of bank switching methods

		// F8 method (standard)
		//
		//	o ET
		//  o Krull
		//  o and lots of others

		cart.readHook = func(address uint16) uint8 {
			data := cart.memory[cart.bank][address]
			if address == 0x0ff8 {
				cart.bank = 0
			} else if address == 0x0ff9 {
				cart.bank = 1
			}
			return data
		}

		// E0 Method (Parker Bros)
		//
		// o Montezuma's Revenge

	case 12288:
		return errors.NewGopherError(errors.CartridgeUnsupported, "12288 bytes not yet supported")

	case 16384:
		cart.allocateCartridgeSpace(cf, 4)

		// TODO: differentiation of bank switching methods

		// F6 method (standard)
		//
		//	o Crystal Castle
		//	o RS Boxing
		//  o Midnite Magic
		//  o and others
		cart.readHook = func(address uint16) uint8 {
			data := cart.memory[cart.bank][address]
			if address == 0x0ff6 {
				cart.bank = 0
			} else if address == 0x0ff7 {
				cart.bank = 1
			} else if address == 0x0ff8 {
				cart.bank = 2
			} else if address == 0x0ff9 {
				cart.bank = 3
			}
			return data
		}

	case 32768:
		cart.allocateCartridgeSpace(cf, 8)

		// F4 method (standard)
		//
		// o Fatal Run
		// o Super Mario Bros.
		// o other homebrew games

		cart.readHook = func(address uint16) uint8 {
			data := cart.memory[cart.bank][address]
			if address == 0x0ff4 {
				cart.bank = 0
			} else if address == 0x0ff5 {
				cart.bank = 1
			} else if address == 0x0ff6 {
				cart.bank = 2
			} else if address == 0x0ff7 {
				cart.bank = 3
			} else if address == 0x0ff8 {
				cart.bank = 4
			} else if address == 0x0ff9 {
				cart.bank = 5
			} else if address == 0x0ffa {
				cart.bank = 6
			} else if address == 0x0ffb {
				cart.bank = 7
			}
			return data
		}

	case 65536:
		return errors.NewGopherError(errors.CartridgeUnsupported, "65536 bytes not yet supported")

	default:
		return errors.NewGopherError(errors.CartridgeUnsupported, fmt.Sprintf("unrecognised cartridge size (%d bytes)", cfi.Size()))
	}

	return nil
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.allocateCartridgeSpace(nil, 1)
	cart.bank = 0
	cart.readHook = func(oa uint16) uint8 { return cart.memory[0][oa] }
}

// Peek is the implementation of Memory.Area.Peek
func (cart Cartridge) Peek(address uint16) (uint8, uint16, string, string, error) {
	if len(cart.memory) == 0 {
		return 0, 0, "", "", errors.NewGopherError(errors.CartridgeMissing)
	}
	return cart.memory[cart.bank][cart.origin|address^cart.origin], address, cart.Label(), "", nil
}

// Poke is the implementation of Memory.Area.Poke
func (cart Cartridge) Poke(address uint16, value uint8) error {
	// if we want to poke cartridge memory we need to account for different
	// cartridge sizes.
	return errors.NewGopherError(errors.UnPokeableAddress, address)
}
