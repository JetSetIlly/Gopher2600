// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package registers

import (
	"strings"
)

// StatusRegister is the special purpose register that stores the flags of the CPU.
type StatusRegister struct {
	Sign             bool
	Overflow         bool
	Break            bool
	DecimalMode      bool
	InterruptDisable bool
	Zero             bool
	Carry            bool
}

// NewStatusRegister is the preferred method of initialisation for the status
// register.
func NewStatusRegister() StatusRegister {
	return StatusRegister{}
}

// Label returns the canonical name for the status register.
func (sr StatusRegister) Label() string {
	return "SR"
}

func (sr StatusRegister) String() string {
	s := strings.Builder{}

	if sr.Sign {
		s.WriteRune('S')
	} else {
		s.WriteRune('s')
	}
	if sr.Overflow {
		s.WriteRune('V')
	} else {
		s.WriteRune('v')
	}

	s.WriteRune('-')

	if sr.Break {
		s.WriteRune('B')
	} else {
		s.WriteRune('b')
	}
	if sr.DecimalMode {
		s.WriteRune('D')
	} else {
		s.WriteRune('d')
	}
	if sr.InterruptDisable {
		s.WriteRune('I')
	} else {
		s.WriteRune('i')
	}
	if sr.Zero {
		s.WriteRune('Z')
	} else {
		s.WriteRune('z')
	}
	if sr.Carry {
		s.WriteRune('C')
	} else {
		s.WriteRune('c')
	}

	return s.String()
}

// Reset status flags to initial state.
func (sr *StatusRegister) Reset() {
	sr.FromValue(0)
}

// Value converts the StatusRegister struct into a value suitable for pushing
// onto the stack.
func (sr StatusRegister) Value() uint8 {
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

// FromValue converts an 8 bit integer (taken from the stack, for example) to
// the StatusRegister struct receiver.
func (sr *StatusRegister) FromValue(v uint8) {
	sr.Sign = v&0x80 == 0x80
	sr.Overflow = v&0x40 == 0x40
	sr.Break = v&0x10 == 0x10
	sr.DecimalMode = v&0x08 == 0x08
	sr.InterruptDisable = v&0x04 == 0x04
	sr.Zero = v&0x02 == 0x02
	sr.Carry = v&0x01 == 0x01
}
