package memory

// PIA defines the information for and operation allowed for PIA PIA
type PIA struct {
	CPUBus
	Area
	AreaInfo

	memory []uint8
}

// newPIA is the preferred method of initialisation for the PIA pia memory area
func newPIA() *PIA {
	pia := new(PIA)
	if pia == nil {
		return nil
	}
	pia.label = "PIA RAM"
	pia.origin = 0x0080
	pia.memtop = 0x00ff
	pia.memory = make([]uint8, pia.memtop-pia.origin+1)
	return pia
}

// Label is an implementation of Area.Label
func (pia PIA) Label() string {
	return pia.label
}

// Origin is an implementation of Area.Origin
func (pia PIA) Origin() uint16 {
	return pia.origin
}

// Memtop is an implementation of Area.Memtop
func (pia PIA) Memtop() uint16 {
	return pia.memtop
}

// Implementation of CPUBus.Read
func (pia PIA) Read(address uint16) (uint8, error) {
	oa := address - pia.origin
	return pia.memory[oa], nil
}

// Implementation of CPUBus.Write
func (pia *PIA) Write(address uint16, data uint8) error {
	oa := address - pia.origin
	pia.memory[oa] = data
	return nil
}

// Peek is the implementation of Area.Peek
func (pia PIA) Peek(address uint16) (uint8, string, error) {
	oa := address - pia.origin
	return pia.memory[oa], "", nil
}
