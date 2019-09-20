package memory

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
)

// cartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument recieve that address
// normalised to a range of 0x0000 to 0x0fff
type cartMapper interface {
	initialise()
	read(addr uint16) (data uint8, err error)
	write(addr uint16, data uint8) error
	numBanks() int
	getBank(addr uint16) (bank int)
	setBank(addr uint16, bank int) error
	saveState() interface{}
	restoreState(interface{}) error
	ram() []uint8

	// listen differs from write in that the address is the unmapped address on
	// the address bus. for convenience, memory functions deal with addresses
	// that have been mapped and normalised so they count from zero.
	// cartMapper.listen() is the exception.
	listen(addr uint16, data uint8) error
}

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	CPUBus
	DebuggerBus

	label  string
	origin uint16
	memtop uint16

	// full path to the cartridge as stored on disk
	Filename string

	// hash of binary loaded from disk. any subsequent pokes to cartridge
	// memory will not be reflected in the value
	Hash string

	// the specific cartridge data, mapped appropriately to the memory
	// interfaces
	mapper cartMapper
}

// NewCartridge is the preferred method of initialisation for the cartridges
func NewCartridge() *Cartridge {
	cart := new(Cartridge)
	cart.label = "Cartridge"
	cart.origin = 0x1000
	cart.memtop = 0x1fff
	cart.Eject()
	return cart
}

// MachineInfoTerse returns the cartridge information in terse format
func (cart Cartridge) MachineInfoTerse() string {
	return fmt.Sprintf("%s [%s]", cart.Filename, cart.mapper)
}

// MachineInfo returns the cartridge information in verbose format
func (cart Cartridge) MachineInfo() string {
	return cart.MachineInfoTerse()
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

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.Filename = ejectedName
	cart.Hash = ejectedHash
	cart.mapper = newEjected()
}

// Implementation of CPUBus.Read
func (cart Cartridge) Read(addr uint16) (uint8, error) {
	addr &= cart.Origin() - 1
	return cart.mapper.read(addr)
}

// Implementation of CPUBus.Write
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	addr &= cart.Origin() - 1
	return cart.mapper.write(addr, data)
}

// Peek is the implementation of Memory.Area.Peek
func (cart Cartridge) Peek(addr uint16) (uint8, error) {
	addr &= cart.Origin() - 1
	return cart.mapper.read(addr)
}

// Poke is the implementation of Memory.Area.Poke
func (cart Cartridge) Poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

// fingerprint8k attempts a divination of 8k cartridge data and decide on a
// suitable cartMapper implementation
func (cart Cartridge) fingerprint8k(cf io.ReadSeeker) func(io.ReadSeeker) (cartMapper, error) {
	b := make([]byte, 8192)
	cf.Seek(0, io.SeekStart)
	cf.Read(b)

	if fingerprintTigervision(b) {
		return newTigervision
	}

	if fingerprintParkerBros(b) {
		return newparkerBros
	}

	return newAtari8k
}

// fingerprint16k attempts a divination of 16k cartridge data and decide on a
// suitable cartMapper implementation
func (cart Cartridge) fingerprint16k(cf io.ReadSeeker) func(io.ReadSeeker) (cartMapper, error) {
	b := make([]byte, 16384)
	cf.Seek(0, io.SeekStart)
	cf.Read(b)

	if fingerprintMnetwork(b) {
		return newMnetwork
	}

	return newAtari16k
}

// Attach loads the bytes from a cartridge (represented by 'filename')
func (cart *Cartridge) Attach(filename string) error {
	cf, err := os.Open(filename)
	if err != nil {
		return errors.New(errors.CartridgeFileUnavailable, filename)
	}
	defer func() {
		_ = cf.Close()
	}()

	// get file info
	cfi, err := cf.Stat()
	if err != nil {
		return err
	}

	// note name of cartridge
	cart.Filename = filename
	cart.mapper = newEjected()

	// generate hash
	key := sha1.New()
	if _, err := io.Copy(key, cf); err != nil {
		return err
	}
	cart.Hash = fmt.Sprintf("%x", key.Sum(nil))

	// how cartridges are mapped into the 4k space can differs dramatically.
	// the following implementation details have been cribbed from Kevin
	// Horton's "Cart Information" document [sizes.txt]

	switch cfi.Size() {
	case 2048:
		cart.mapper, err = newAtari2k(cf)
		if err != nil {
			return err
		}

	case 4096:
		cart.mapper, err = newAtari4k(cf)
		if err != nil {
			return err
		}

	case 8192:
		cart.mapper, err = cart.fingerprint8k(cf)(cf)
		if err != nil {
			return err
		}

	case 12288:
		cart.mapper, err = newCBS(cf)
		if err != nil {
			return err
		}

	case 16384:
		cart.mapper, err = cart.fingerprint16k(cf)(cf)
		if err != nil {
			return err
		}

	case 32768:
		cart.mapper, err = newAtari32k(cf)
		if err != nil {
			return err
		}

	case 65536:
		return errors.New(errors.CartridgeFileError, "65536 bytes not yet supported")

	default:
		return errors.New(errors.CartridgeFileError, fmt.Sprintf("unrecognised cartridge size (%d bytes)", cfi.Size()))
	}

	return nil
}

// Initialise calls the current mapper's initialise function
func (cart *Cartridge) Initialise() {
	cart.mapper.initialise()
}

// NumBanks calls the current mapper's numBanks function
func (cart Cartridge) NumBanks() int {
	return cart.mapper.numBanks()
}

// GetBank calls the current mapper's addressBank function. it returns the
// current bank number for the specified address
func (cart Cartridge) GetBank(addr uint16) int {
	addr &= cart.Origin() - 1
	return cart.mapper.getBank(addr)
}

// SetBank sets the bank for the specificed address. it sets the specified
// address to reference the specified bank
func (cart *Cartridge) SetBank(addr uint16, bank int) error {
	addr &= cart.Origin() - 1
	return cart.mapper.setBank(addr, bank)
}

// SaveState calls the current mapper's saveState function
func (cart *Cartridge) SaveState() interface{} {
	return cart.mapper.saveState()
}

// RestoreState calls the current mapper's restoreState function
func (cart *Cartridge) RestoreState(state interface{}) error {
	return cart.mapper.restoreState(state)
}

// RAM returns a read only instance of any cartridge RAM
func (cart Cartridge) RAM() []uint8 {
	return cart.mapper.ram()
}

// Listen for data at the specified address. return CartridgeListen error if
// nothing was done with the information. Callers to Listen() will probably
// want to filter out that error.
func (cart Cartridge) Listen(addr uint16, data uint8) error {
	return cart.mapper.listen(addr, data)
}
