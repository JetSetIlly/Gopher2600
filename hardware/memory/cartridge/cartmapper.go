package cartridge

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

	// tigervision cartridges have a very wierd bank-switching method that
	// require a way of notifying the cartridge of writes to addresses outside
	// of cartridge space
	listen(addr uint16, data uint8)

	// poke new value anywhere into currently selected bank of cartridge memory
	// (including ROM).
	poke(addr uint16, data uint8) error

	// patch differs from poke in that it alters the data as though it was
	// being read from disk
	patch(offset uint16, data uint8) error
}

// optionalSuperchip are implemented by cartMappers that have an optional
// superchip
type optionalSuperchip interface {
	addSuperchip() bool
}
