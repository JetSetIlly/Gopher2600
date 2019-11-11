package memory

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"strings"
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

// optionalSuperchip are implemented by cartMappers that have an optional
// superchip
type optionalSuperchip interface {
	addSuperchip() bool
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

	// the format requested by the CartridgeLoader
	RequestedFormat string

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

func (cart Cartridge) String() string {
	return fmt.Sprintf("%s [%s]", cart.Filename, cart.mapper)
}

// Label is an implementation of Area.Label
func (cart Cartridge) Label() string {
	return cart.label
}

// Origin is an implementation of Area.Origin
// * optimisation: called a lot. pointer to Cartridge to prevent duffcopy
func (cart *Cartridge) Origin() uint16 {
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
// * optimisation: called a lot. pointer to Cartridge to prevent duffcopy
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
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
func (cart Cartridge) fingerprint8k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintParkerBros(data) {
		return newparkerBros
	}

	return newAtari8k
}

// fingerprint16k attempts a divination of 16k cartridge data and decide on a
// suitable cartMapper implementation
func (cart Cartridge) fingerprint16k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	return newAtari16k
}

func (cart *Cartridge) fingerprint(data []byte) error {
	var err error

	switch len(data) {
	case 2048:
		cart.mapper, err = newAtari2k(data)
		if err != nil {
			return err
		}

	case 4096:
		cart.mapper, err = newAtari4k(data)
		if err != nil {
			return err
		}

	case 8192:
		cart.mapper, err = cart.fingerprint8k(data)(data)
		if err != nil {
			return err
		}

	case 12288:
		cart.mapper, err = newCBS(data)
		if err != nil {
			return err
		}

	case 16384:
		cart.mapper, err = cart.fingerprint16k(data)(data)
		if err != nil {
			return err
		}

	case 32768:
		cart.mapper, err = newAtari32k(data)
		if err != nil {
			return err
		}

	case 65536:
		return errors.New(errors.CartridgeError, "65536 bytes not yet supported")

	default:
		return errors.New(errors.CartridgeError, fmt.Sprintf("unrecognised cartridge size (%d bytes)", len(data)))
	}

	// if cartridge mapper implements the optionalSuperChip interface then try
	// to add the additional RAM
	if superchip, ok := cart.mapper.(optionalSuperchip); ok {
		superchip.addSuperchip()
	}

	return nil
}

// Attach loads the bytes from a cartridge (represented by 'filename')
func (cart *Cartridge) Attach(cartload cartridgeloader.Loader) error {
	data, err := cartload.Load()
	if err != nil {
		return err
	}

	// note name of cartridge
	cart.Filename = cartload.Filename
	cart.RequestedFormat = cartload.Format
	cart.mapper = newEjected()

	// generate hash
	cart.Hash = fmt.Sprintf("%x", sha1.Sum(data))

	// check that the hash matches the expected value
	if cartload.Hash != "" && cartload.Hash != cart.Hash {
		return errors.New(errors.CartridgeError, "unexpected hash value")
	}

	// how cartridges are mapped into the 4k space can differs dramatically.
	// the following implementation details have been cribbed from Kevin
	// Horton's "Cart Information" document [sizes.txt]

	cartload.Format = strings.ToUpper(cartload.Format)

	if cartload.Format == "" || cartload.Format == "AUTO" {
		return cart.fingerprint(data)
	}

	addSuperchip := false

	switch cartload.Format {
	case "2k":
		cart.mapper, err = newAtari2k(data)
	case "4k":
		cart.mapper, err = newAtari4k(data)
	case "F8":
		cart.mapper, err = newAtari8k(data)
	case "F6":
		cart.mapper, err = newAtari16k(data)
	case "F4":
		cart.mapper, err = newAtari32k(data)

	case "2k+SC":
		cart.mapper, err = newAtari2k(data)
		addSuperchip = true
	case "4k+SC":
		cart.mapper, err = newAtari4k(data)
		addSuperchip = true
	case "F8+SC":
		cart.mapper, err = newAtari8k(data)
		addSuperchip = true
	case "F6+SC":
		cart.mapper, err = newAtari16k(data)
		addSuperchip = true
	case "F4+SC":
		cart.mapper, err = newAtari32k(data)
		addSuperchip = true

	case "FA":
		cart.mapper, err = newCBS(data)
	case "FE":
		// TODO
	case "E0":
		cart.mapper, err = newparkerBros(data)
	case "E7":
		cart.mapper, err = newMnetwork(data)
	case "3F":
		cart.mapper, err = newTigervision(data)
	case "AR":
		// TODO
	}

	if addSuperchip {
		if superchip, ok := cart.mapper.(optionalSuperchip); ok {
			if !superchip.addSuperchip() {
				err = errors.New(errors.CartridgeError, "error adding superchip")
			}
		} else {
			err = errors.New(errors.CartridgeError, "error adding superchip")
		}
	}

	return err
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

// CartFormat returns the actual format of the loaded cartridge
func (cart Cartridge) CartFormat() string {
	return ""
}
