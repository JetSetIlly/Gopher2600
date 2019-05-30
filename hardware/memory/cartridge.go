package memory

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
)

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	CPUBus
	Area
	AreaInfo

	// full path to the cartridge as stored on disk
	Filename string

	// hash of binary loaded from disk. any subsequent pokes to cartridge
	// memory will not be reflected in the value
	Hash string

	// cartridge bank-switching method
	method string

	NumBanks int
	Bank     int
	memory   [][]uint8

	// readHook allows custom read routines depending on the cartridge type
	// that has been inserted.
	readHook func(uint16) (uint8, error)

	// similarly for writeHook
	writeHook func(uint16, uint8) error

	// some cartridges have extra RAM - they'll need custom read/write hooks to
	// make use of it
	extraRAM []uint8
}

const ejectedName = "ejected"
const ejectedHash = "nohash"
const ejectedMethod = "none"

// NewCart is the preferred method of initialisation for the cartridges
// -- public naming because it is sometimes useful to be able to initialise a
// this type of memory when analysing a cartidge (eg. disassembly)
func NewCart() *Cartridge {
	cart := new(Cartridge)
	cart.label = "Cartridge"
	cart.origin = 0x1000
	cart.memtop = 0x1fff
	cart.Eject()
	return cart
}

// MachineInfoTerse returns the cartridge information in terse format
func (cart Cartridge) MachineInfoTerse() string {
	return fmt.Sprintf("%s [%s] bank=%d", cart.Filename, cart.method, cart.Bank)
}

// MachineInfo returns the cartridge information in verbose format
func (cart Cartridge) MachineInfo() string {
	return fmt.Sprintf("name: %s\nmethod: %s\nbank:%d", cart.Filename, cart.method, cart.Bank)
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
		return 0, errors.NewFormattedError(errors.CartridgeMissing)
	}
	return cart.readHook(cart.origin | address ^ cart.origin)
}

// Implementation of CPUBus.Write
func (cart *Cartridge) Write(address uint16, data uint8) error {
	return cart.writeHook(cart.origin|address^cart.origin, data)
}

// readBanks is a generalised allocation of memory and file reading routine.
// common to all cartridge sizes
func (cart *Cartridge) readBanks(file io.ReadSeeker, numBanks int) error {
	cart.NumBanks = numBanks

	// allocate enough memory for new cartridge
	cart.memory = make([][]uint8, cart.NumBanks)

	if file == nil {
		return nil
	}

	file.Seek(0, io.SeekStart)

	for b := 0; b < cart.NumBanks; b++ {
		cart.memory[b] = make([]uint8, 4096)

		if file != nil {
			// read cartridge
			n, err := file.Read(cart.memory[b])
			if err != nil {
				return err
			}
			if n != 4096 {
				return errors.NewFormattedError(errors.CartridgeFileError, "not enough bytes in the cartridge file")
			}
		}
	}

	return nil
}

// Attach loads the bytes from a cartridge (represented by 'filename')
func (cart *Cartridge) Attach(filename string) error {
	cf, err := os.Open(filename)
	if err != nil {
		return errors.NewFormattedError(errors.CartridgeFileUnavailable, filename)
	}
	defer func() {
		_ = cf.Close()
	}()

	// get file info
	cfi, err := cf.Stat()
	if err != nil {
		return err
	}

	// generate hash
	key := sha1.New()
	if _, err := io.Copy(key, cf); err != nil {
		return err
	}
	cart.Hash = fmt.Sprintf("%x", key.Sum(nil))

	// we always start in bank 0
	cart.Bank = 0

	// set default read hooks
	cart.readHook = func(addr uint16) (uint8, error) {
		return 0, errors.NewFormattedError(errors.UnreadableAddress, addr)
	}

	cart.writeHook = func(addr uint16, data uint8) error {
		return errors.NewFormattedError(errors.UnwritableAddress, addr)
	}

	// how cartridges are mapped into the 4k space can differs dramatically.
	// the following implementation details have been cribbed from Kevin
	// Horton's "Cart Information" document [sizes.txt]

	switch cfi.Size() {
	case 2048:
		// note that while we're allocating a full 4096 bytes for this
		// cartrdige size, the readHook below ensures that we only ever read
		// the first 2048.
		cart.readBanks(cf, 1)

		// this is a half-size cartridge of 2048 bytes
		//
		//	o Combat
		//  o Dragster
		//  o Outlaw
		//	o Surround
		//  o mostly early cartridges
		cart.method = "standard 2k"

		cart.readHook = func(addr uint16) (uint8, error) {
			// because we've only allocated half the amount of memory that
			// should be there, we need a further mask to make sure the addr
			// is in range
			return cart.memory[0][addr&0x07ff], nil
		}

	case 4096:
		cart.readBanks(cf, 1)

		// this is a regular cartridge of 4096 bytes
		//
		//  o Pitfall
		//  o Adventure
		//  o Yars Revenge
		//  o most 2600 cartridges...
		cart.method = "standard 4k"

		cart.readHook = func(addr uint16) (uint8, error) {
			return cart.memory[0][addr], nil
		}

		cart.addCartridgeRAM()

	case 8192:
		cart.readBanks(cf, 2)

		// TODO: differentiation of bank switching methods

		// F8 method (standard)
		//
		//	o ET
		//  o Krull
		//  o and lots of others
		cart.method = "standard 8k (F8)"

		cart.readHook = func(addr uint16) (uint8, error) {
			data := cart.memory[cart.Bank][addr]
			if addr == 0x0ff8 {
				cart.Bank = 0
			} else if addr == 0x0ff9 {
				cart.Bank = 1
			}
			return data, nil
		}

		cart.writeHook = func(addr uint16, data uint8) error {
			if addr == 0x0ff8 {
				cart.Bank = 0
			} else if addr == 0x0ff9 {
				cart.Bank = 1
			} else {
				return errors.NewFormattedError(errors.UnwritableAddress, addr)
			}
			return nil
		}

		cart.addCartridgeRAM()

		// E0 Method (Parker Bros)
		//
		// o Montezuma's Revenge

	case 12288:
		return errors.NewFormattedError(errors.CartridgeFileError, "12288 bytes not yet supported")

	case 16384:
		cart.readBanks(cf, 4)

		// TODO: differentiation of bank switching methods

		// F6 method (standard)
		//
		//	o Crystal Castle
		//	o RS Boxing
		//  o Midnite Magic
		//  o and others
		cart.method = "standard 16k (F6)"

		cart.readHook = func(addr uint16) (uint8, error) {
			data := cart.memory[cart.Bank][addr]
			if addr == 0x0ff6 {
				cart.Bank = 0
			} else if addr == 0x0ff7 {
				cart.Bank = 1
			} else if addr == 0x0ff8 {
				cart.Bank = 2
			} else if addr == 0x0ff9 {
				cart.Bank = 3
			}
			return data, nil
		}

		cart.writeHook = func(addr uint16, data uint8) error {
			if addr == 0x0ff6 {
				cart.Bank = 0
			} else if addr == 0x0ff7 {
				cart.Bank = 1
			} else if addr == 0x0ff8 {
				cart.Bank = 2
			} else if addr == 0x0ff9 {
				cart.Bank = 3
			} else {
				return errors.NewFormattedError(errors.UnwritableAddress, addr)
			}
			return nil
		}

		cart.addCartridgeRAM()

	case 32768:
		cart.readBanks(cf, 8)

		// F4 method (standard)
		//
		// o Fatal Run
		// o Super Mario Bros.
		// o Donkey Kong
		// o other homebrew games
		cart.method = "standard 32k (F4)"

		cart.readHook = func(addr uint16) (uint8, error) {
			data := cart.memory[cart.Bank][addr]
			if addr == 0x0ff4 {
				cart.Bank = 0
			} else if addr == 0x0ff5 {
				cart.Bank = 1
			} else if addr == 0x0ff6 {
				cart.Bank = 2
			} else if addr == 0x0ff7 {
				cart.Bank = 3
			} else if addr == 0x0ff8 {
				cart.Bank = 4
			} else if addr == 0x0ff9 {
				cart.Bank = 5
			} else if addr == 0x0ffa {
				cart.Bank = 6
			} else if addr == 0x0ffb {
				cart.Bank = 7
			}
			return data, nil
		}

		cart.writeHook = func(addr uint16, data uint8) error {
			if addr == 0x0ff4 {
				cart.Bank = 0
			} else if addr == 0x0ff5 {
				cart.Bank = 1
			} else if addr == 0x0ff6 {
				cart.Bank = 2
			} else if addr == 0x0ff7 {
				cart.Bank = 3
			} else if addr == 0x0ff8 {
				cart.Bank = 4
			} else if addr == 0x0ff9 {
				cart.Bank = 5
			} else if addr == 0x0ffa {
				cart.Bank = 6
			} else if addr == 0x0ffb {
				cart.Bank = 7
			} else {
				return errors.NewFormattedError(errors.UnwritableAddress, addr)
			}
			return nil
		}

		cart.addCartridgeRAM()

	case 65536:
		return errors.NewFormattedError(errors.CartridgeFileError, "65536 bytes not yet supported")

	default:
		cart.Eject()
		return errors.NewFormattedError(errors.CartridgeFileError, fmt.Sprintf("unrecognised cartridge size (%d bytes)", cfi.Size()))
	}

	// note name of cartridge
	cart.Filename = filename

	return nil
}

func (cart *Cartridge) addCartridgeRAM() bool {
	// information about cartridge RAM from Horton's "Cart Information"
	// document [sizes.txt] and observation of incorrect ROM behaviour when
	// compared to the Stella emulator.

	// check for cartridge memory
	// -- this method of detection simply checks whether the first 256
	// of each bank are "empty"
	// -- I've guessed that this is a good method. if there's another one I
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

	// update write hook
	writeHook := cart.writeHook
	cart.writeHook = func(addr uint16, data uint8) error {
		if addr > 127 {
			return writeHook(addr, data)
		}
		cart.extraRAM[addr] = data
		return nil
	}

	// update read hook
	readHook := cart.readHook
	cart.readHook = func(addr uint16) (uint8, error) {
		if addr > 127 && addr < 256 {
			return cart.extraRAM[addr-128], nil
		}
		return readHook(addr)
	}

	// update cartridge method information
	cart.method = fmt.Sprintf("%s inc. extra RAM", cart.method)

	return true
}

// BankSwitch changes the current bank number
func (cart *Cartridge) BankSwitch(bank int) error {
	if bank > cart.NumBanks {
		msg := fmt.Sprintf("cartridge error: bank out of range (%d, max=%d)", bank, cart.NumBanks)
		return errors.NewFormattedError(errors.CartridgeError, msg)
	}
	cart.Bank = bank
	return nil
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.Filename = ejectedName
	cart.method = ejectedMethod
	cart.Hash = ejectedHash
	cart.NumBanks = 1
	cart.Bank = 0
	cart.readBanks(nil, cart.NumBanks)
	cart.readHook = func(uint16) (uint8, error) { return 0, nil }
	cart.writeHook = func(addr uint16, data uint8) error { return nil }
	cart.extraRAM = nil
}

// Peek is the implementation of Memory.Area.Peek
func (cart Cartridge) Peek(address uint16) (uint8, error) {
	if len(cart.memory) == 0 {
		return 0, errors.NewFormattedError(errors.CartridgeMissing)
	}
	return cart.memory[cart.Bank][cart.origin|address^cart.origin], nil
}

// Poke is the implementation of Memory.Area.Poke
func (cart Cartridge) Poke(address uint16, value uint8) error {
	// if we want to poke cartridge memory we need to account for different
	// cartridge sizes.
	return errors.NewFormattedError(errors.UnpokeableAddress, address)
}
