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

package supercharger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// Registers implements the mapper.CartRegisters interface.
type Registers struct {
	Value uint8
	Delay int // 0=off, 1=trigger

	// delay is decremented everytime address changes. we therefore need
	// to keep track of what the last address was in order to tell if the
	// address bus has transitioned
	transitionAddress uint16

	// the last value to be written to (not including fff8 writes)
	LastWriteValue   uint8
	LastWriteAddress uint16

	// config byte, raw value
	ConfigByte uint8

	// config byte broken into parts
	WriteDelay  int
	BankingMode int
	RAMwrite    bool
	ROMpower    bool
}

func (r *Registers) setConfigByte(v uint8) {
	r.ConfigByte = v
	r.ROMpower = v&0x01 != 0x01
	r.RAMwrite = v&0x02 == 0x02
	r.BankingMode = int((v >> 2) & 0x07)
	r.WriteDelay = int((v >> 5) & 0x07)
}

func (r Registers) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("Value: %#02x  Delay: %d\n", r.Value, r.Delay))

	if r.LastWriteAddress > 0x000 {
		s.WriteString(fmt.Sprintf("   last write %#02x to %#04x\n", r.LastWriteValue, r.LastWriteAddress))
	}

	s.WriteString(fmt.Sprintf("RAM write: %v", r.RAMwrite))
	s.WriteString(fmt.Sprintf("  ROM power: %v\n", r.ROMpower))

	s.WriteString(r.MappedBanks())

	return s.String()
}

// MappedBanks is like string but just the bank information. we use this when
// building the mapper summary, the String() function is too verbose for that.
func (r *Registers) MappedBanks() string {
	s := strings.Builder{}

	s.WriteString("Banks: ")
	switch r.BankingMode {
	case 0:
		s.WriteString("3 B")

	case 1:
		s.WriteString("1 B")

	case 2:
		s.WriteString("3 1")

	case 3:
		s.WriteString("1 3")

	case 4:
		s.WriteString("3 B")

	case 5:
		s.WriteString("2 B")

	case 6:
		s.WriteString("3 2")

	case 7:
		s.WriteString("2 3")
	}

	return s.String()
}

func (r *Registers) transitionCount(addr uint16) {
	// Kevin Horton in the "Mostly Inclusive Atari 2600 Mapper / Selected
	// Hardware Document" clarifies what is meant by "transition":
	//
	// "Note that when I say 'transition', I am talking about when one or more
	// of the 13 address lines changes."
	//
	// In other words, if the address hasn't changed then it does not count as
	// a transition.
	//
	// we don't strictly need to keep track of this because this function will
	// only be called as a result of cartridge.Listen() being called by the
	// memory sub-system - and that only happens if the address bus has
	// transitioned
	if addr != r.transitionAddress {
		if r.Delay > 0 {
			r.Delay--
		}
		r.transitionAddress = addr
	}
}

// GetRegisters implements the mapper.CartRegistersBus interface.
func (cart Supercharger) GetRegisters() mapper.CartRegisters {
	return cart.state.registers
}

// PutRegister implements the mapper.CartRegistersBus interface
//
// the register argument must be one of the following and after the = sign, the
// type to which the data argument will be converted.
//
//	value = int
//	delay = int (0 ... 6)
//	ramwrite = bool
//	rompower = bool
//
// note that PutRegister() will panic() if the register or data string is invalid.
func (cart *Supercharger) PutRegister(register string, data string) {
	d8, _ := strconv.ParseUint(data, 16, 8)

	switch register {
	case "value":
		cart.state.registers.Value = uint8(d8)

	case "delay":
		if d8 > 6 {
			panic("delay value out of range")
		}
		cart.state.registers.Delay = int(d8)

	case "ramwrite":
		switch data {
		case "true":
			cart.state.registers.RAMwrite = true
		case "false":
			cart.state.registers.RAMwrite = false
		default:
			panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
		}
	case "rompower":
		switch data {
		case "true":
			cart.state.registers.ROMpower = true
		case "false":
			cart.state.registers.ROMpower = false
		default:
			panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
		}

	case "bankingmode":
		if d8 > 7 {
			panic("bankingmode value out of range")
		}
		cart.state.registers.BankingMode = int(d8)

	default:
		panic(fmt.Sprintf("unrecognised variable [%s]", register))
	}
}
