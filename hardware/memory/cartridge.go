package memory

import (
	"fmt"
	"gopher2600/errors"
	"os"
)

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	CPUBus
	Area
	AreaInfo
	memory []uint8
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
	oa := address - cart.origin
	return cart.memory[oa], nil
}

// Implementation of CPUBus.Write
func (cart *Cartridge) Write(address uint16, data uint8) error {
	return errors.NewGopherError(errors.UnwritableAddress, address)
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

	switch cfi.Size() {
	case 4096:
		// this is a regular cartridge of 4096 bytes
		//  o Pitfall
		//  o Advenure
		//  o Yars Revenge
		//  o most 2600 cartridges...
		// for this cartridge type we simply read the entire cartridge into a
		// memory allocation of 4096 bytes. there is no need for further memory
		// mapping.

		// allocate enough memory for new cartridge
		cart.memory = make([]uint8, 4096)

		// read cartridge
		n, err := cf.Read(cart.memory)
		if err != nil {
			return err
		}
		if n != 4096 {
			return errors.NewGopherError(errors.CartridgeFileError, errors.FileTruncated)
		}

	case 2048:
		// this is a half-size cartridge of 2048 bytes
		//	o Combat
		//  o Dragster
		//  o Outlaw
		//	o Surround
		//  o mostly early cartrdiges
		// for this cartridge type we simply read the cartridge twice into a
		// memory space of 4096. there is no need for further memory mappping
		// using this method; however, POKEing into cartridge space will also
		// need to be performed twice. as this isn't normal 2600 behaviour
		// though, I'm not too concerned.

		// allocate enough memory for new cartridge -- for now, allocate the
		// full 4096 and read cartridge twice
		cart.memory = make([]uint8, 4096)

		// read cartridge
		n, err := cf.Read(cart.memory[:2048])
		if err != nil {
			return err
		}
		if n != 2048 {
			return errors.NewGopherError(errors.CartridgeFileError, errors.FileTruncated)
		}

		// read cartridge again (into second half of memory)
		cf.Seek(0, 0)
		n, err = cf.Read(cart.memory[2048:])
		if err != nil {
			return err
		}
		if n != 2048 {
			return errors.NewGopherError(errors.CartridgeFileError, errors.FileTruncated)
		}

	case 8192:
		return errors.NewGopherError(errors.CartridgeUnsupported, "8192 bytes not yet supported")

	case 16384:
		return errors.NewGopherError(errors.CartridgeUnsupported, "16384 bytes not yet supported")

	case 32768:
		return errors.NewGopherError(errors.CartridgeUnsupported, "32768 bytes not yet supported")

	default:
		return errors.NewGopherError(errors.CartridgeUnsupported, fmt.Sprintf("file size unrecognised %d bytes", cfi.Size()))
	}

	return nil
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.memory = make([]uint8, 4096)
}

// Peek is the implementation of Memory.Area.Peek
func (cart Cartridge) Peek(address uint16) (uint8, uint16, string, string, error) {
	if len(cart.memory) == 0 {
		return 0, 0, "", "", errors.NewGopherError(errors.CartridgeMissing)
	}
	oa := address - cart.origin
	return cart.memory[oa], address, cart.Label(), "", nil
}
