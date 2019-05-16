package cpu

import "fmt"

// TODO: Status register N,V,Z flag bug

// StatusRegister is the special purpose register that stores the flags of the CPU
type StatusRegister struct {
	label            string
	shortLabel       string
	Sign             bool
	Overflow         bool
	Break            bool
	DecimalMode      bool
	InterruptDisable bool
	Zero             bool
	Carry            bool
}

// NewStatusRegister is the preferred method of initialisation for the status
// register
func NewStatusRegister(label string, shortLabel string) StatusRegister {
	sr := new(StatusRegister)
	sr.label = label
	sr.shortLabel = shortLabel
	return *sr
}

// MachineInfoTerse returns the status register information in terse format
func (sr StatusRegister) MachineInfoTerse() string {
	return fmt.Sprintf("%s=%s", sr.shortLabel, sr.ToBits())
}

// MachineInfo returns the status register information in verbose format
func (sr StatusRegister) MachineInfo() string {
	return fmt.Sprintf("%s: %v", sr.label, sr.ToBits())
}

// map String to MachineInfo
func (sr StatusRegister) String() string {
	return sr.MachineInfo()
}

// ToBits returns the register as a labelled bit pattern
func (sr StatusRegister) ToBits() string {
	var v string

	if sr.Sign {
		v += "S"
	} else {
		v += "s"
	}
	if sr.Overflow {
		v += "V"
	} else {
		v += "v"
	}
	v += "-"
	if sr.Break {
		v += "B"
	} else {
		v += "b"
	}
	if sr.DecimalMode {
		v += "D"
	} else {
		v += "d"
	}
	if sr.InterruptDisable {
		v += "I"
	} else {
		v += "i"
	}
	if sr.Zero {
		v += "Z"
	} else {
		v += "z"
	}
	if sr.Carry {
		v += "C"
	} else {
		v += "c"
	}

	return v
}

func (sr *StatusRegister) reset() {
	sr.FromUint8(0)
}

// ToUint8 converts the StatusRegister struct into a value suitable for pushing
// onto the stack
func (sr StatusRegister) ToUint8() uint8 {
	var v uint8

	if sr.Sign {
		v |= 0x80
	}
	if sr.Overflow {
		v |= 0x40
	}
	if sr.Break {
		v |= 0x10
	}
	if sr.DecimalMode {
		v |= 0x08
	}
	if sr.InterruptDisable {
		v |= 0x04
	}
	if sr.Zero {
		v |= 0x02
	}
	if sr.Carry {
		v |= 0x01
	}

	// unused bit in the status register is always 1. this doesn't matter when
	// we're in normal form but it does matter in uint8 context
	v |= 0x20

	return v
}

// FromUint8 converts an 8 bit integer (taken from the stack, for example) to
// the StatusRegister struct receiver
func (sr *StatusRegister) FromUint8(v uint8) {
	sr.Sign = v&0x80 == 0x80
	sr.Overflow = v&0x40 == 0x40
	sr.Break = v&0x10 == 0x10
	sr.DecimalMode = v&0x08 == 0x08
	sr.InterruptDisable = v&0x04 == 0x04
	sr.Zero = v&0x02 == 0x02
	sr.Carry = v&0x01 == 0x01
}
