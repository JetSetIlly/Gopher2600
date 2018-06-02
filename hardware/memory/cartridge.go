package memory

import (
	"fmt"
	"os"
)

// MissingCartridgeError returned by those functions that really require a
// cartridge to be inserted.
type MissingCartridgeError struct{}

func (MissingCartridgeError) Error() string {
	return "no cartridge attached"
}

const bankSize = 4096

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	CPUBus
	Area
	AreaInfo

	memory []uint8
	bank   uint16
}

// newCart is the preferred method of initialisation for the cartridges
func newCart() *Cartridge {
	cart := new(Cartridge)
	if cart == nil {
		return nil
	}
	cart.label = "Cartridge"
	cart.origin = 0x1000
	cart.memtop = 0x1fff
	return cart
}

// Label is an implementation of Area.Label
func (cart Cartridge) Label() string {
	return cart.label
}

// Clear is an implementation of CPUBus.Clear
func (cart *Cartridge) Clear() {
	// clearing cartridge memory is semantically the same as ejecting the cartridge
	cart.Eject()
}

// Implementation of CPUBus.Read
func (cart Cartridge) Read(address uint16) (uint8, error) {
	if len(cart.memory) == 0 {
		return 0, new(MissingCartridgeError)
	}
	oa := address - cart.origin
	oa += cart.bank * bankSize
	return cart.memory[oa], nil
}

// Implementation of CPUBus.Write
func (cart *Cartridge) Write(address uint16, data uint8) error {
	return fmt.Errorf("refusing to write to cartridge")
}

// Attach loads the bytes from a cartridge (represented by 'filename')
func (cart *Cartridge) Attach(filename string) error {
	cf, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening cartridge (%s)", err)
	}
	defer func() {
		_ = cf.Close()
	}()

	// get file info
	cfi, err := cf.Stat()
	if err != nil {
		return err
	}

	// check that cartridge is of a supported size
	// TODO: ensure that this is a complete and accurate check
	if cfi.Size()%bankSize != 0 {
		return fmt.Errorf("cartridge (%s) is not of a supported size (%d)", filename, cfi.Size())
	}

	// allocate enough memory for new cartridge
	cart.memory = make([]uint8, cfi.Size())

	// read cartridge
	n, err := cf.Read(cart.memory)
	if err != nil {
		return err
	}
	if n != len(cart.memory) {
		return fmt.Errorf("error reading cartridge file (%s)", filename)
	}

	// make sure we're pointing to the first bank
	cart.bank = 0

	return nil
}

// Eject removes memory from cartridge space
func (cart *Cartridge) Eject() {
	cart.memory = make([]uint8, 0)
}

// Peek is the implementation of Area.Peek
func (cart Cartridge) Peek(address uint16) (uint8, string, error) {
	if len(cart.memory) == 0 {
		return 0, "", new(MissingCartridgeError)
	}
	oa := address - cart.origin
	oa += cart.bank * bankSize
	return cart.memory[oa], "", nil
}
