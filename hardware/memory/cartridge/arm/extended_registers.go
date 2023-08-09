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

package arm

import (
	"fmt"
	"math"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// this file implements the coprocessor.CartCoProc interface where it relates to
// coprocessor registers
//
// extended registers are how the DWARF specification describes the system for
// identifying registers in the coprocessor. details of the extended registers
// can be found at:
//
// https://github.com/ARM-software/abi-aa/releases/download/2023Q1/aadwarf32.pdf

var arm7tdmiRegisterSpec = coprocessor.ExtendedRegisterSpec{
	coprocessor.ExtendedRegisterCoreGroup: {
		Name:   coprocessor.ExtendedRegisterCoreGroup,
		Prefix: "R",
		Start:  0,
		End:    15,
	},
}

var armv7mRegisterSpec = coprocessor.ExtendedRegisterSpec{
	coprocessor.ExtendedRegisterCoreGroup: {
		Name:   coprocessor.ExtendedRegisterCoreGroup,
		Prefix: "R",
		Start:  0,
		End:    15,
	},
	"fpu": {
		Name:      "fpu",
		Prefix:    "S",
		Start:     64,
		End:       95,
		Formatted: true,
	},
}

// RegisterSpec implements the coprocessor.CartCoProc interface
func (arm *ARM) RegisterSpec() coprocessor.ExtendedRegisterSpec {
	switch arm.mmap.ARMArchitecture {
	case architecture.ARM7TDMI:
		return arm7tdmiRegisterSpec
	case architecture.ARMv7_M:
		return armv7mRegisterSpec
	}
	panic("register spec: unrecognised arm architecture")
}

func (arm *ARM) register(register int, formatted bool) (uint32, string, bool) {
	for k, spec := range arm.RegisterSpec() {
		if register >= spec.Start && register <= spec.End {
			switch k {
			case coprocessor.ExtendedRegisterCoreGroup:
				return arm.state.registers[register], "", true
			case "fpu":
				var s string
				if formatted {
					s = fmt.Sprintf("%f", math.Float32frombits(arm.state.fpu.Registers[register-64]))
				}
				return arm.state.fpu.Registers[register-spec.Start], s, true
			}
		}
	}

	return 0, "", false
}

// Register implements the coprocess.CartCoProc interface. Returns the value in
// the register. Returns false if the requested register is not recognised
func (arm *ARM) Register(register int) (uint32, bool) {
	v, _, ok := arm.register(register, false)
	return v, ok
}

// RegisterFormatted implements the coprocess.CartCoProc interface. Returns the value in
// the register. Returns false if the requested register is not recognised
func (arm *ARM) RegisterFormatted(register int) (uint32, string, bool) {
	return arm.register(register, true)
}

// RegisterSet implements the coprocess.CartCoProc interface. Sets the register
// to the specified value. Returns false if the requested register is not
// recognised
func (arm *ARM) RegisterSet(register int, value uint32) bool {
	for k, spec := range arm.RegisterSpec() {
		if register >= spec.Start && register <= spec.End {
			switch k {
			case coprocessor.ExtendedRegisterCoreGroup:
				arm.state.registers[register] = value
				return true
			case "fpu":
				arm.state.fpu.Registers[register-spec.Start] = value
				return true
			}
		}
	}
	return false
}
