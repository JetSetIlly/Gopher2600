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

// Clear is an implementation of CPUBus.Clear
func (pia *PIA) Clear() {
	for i := range pia.memory {
		pia.memory[i] = 0
	}
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
