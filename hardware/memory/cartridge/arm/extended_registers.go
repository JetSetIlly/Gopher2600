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
	{
		Name:  coprocessor.ExtendedRegisterCoreGroup,
		Start: 0,
		End:   15,
		Label: func(r int) string {
			return fmt.Sprintf("R%02d", r)
		},
	},
}

var armv7mRegisterSpec = coprocessor.ExtendedRegisterSpec{
	{
		Name:  coprocessor.ExtendedRegisterCoreGroup,
		Start: 0,
		End:   15,
		Label: func(r int) string {
			return fmt.Sprintf("R%02d", r)
		},
	},
	{
		Name:  "FPU",
		Start: 64,
		End:   95,
		Label: func(r int) string {
			return fmt.Sprintf("S%02d", r-64)
		},
		Formatted: true,
	},
	{
		Name:  "TIM2",
		Start: 10000,
		End:   10001,
		Label: func(r int) string {
			switch r {
			case 10000:
				return "CR1"
			case 10001:
				return "CNT"
			}
			return "unknown TIM2 register"
		},
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
	for _, spec := range arm.RegisterSpec() {
		if register >= spec.Start && register <= spec.End {
			switch spec.Name {
			case coprocessor.ExtendedRegisterCoreGroup:
				var s string
				if formatted {
					s = fmt.Sprintf("%08x", arm.state.registers[register])
				}
				return arm.state.registers[register], s, true
			case "FPU":
				var s string
				if formatted {
					s = fmt.Sprintf("%f", math.Float32frombits(arm.state.fpu.Registers[register-spec.Start]))
				}
				return arm.state.fpu.Registers[register-spec.Start], s, true
			case "TIM2":
				var v uint32

				switch register {
				case 10000:
					v, _, _ = arm.state.timer2.Read(arm.mmap.TIM2CR1)
				case 10001:
					v, _, _ = arm.state.timer2.Read(arm.mmap.TIM2CNT)
				default:
					return 0, "", false
				}

				var s string
				if formatted {
					s = fmt.Sprintf("%08x", v)
				}

				return v, s, true
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
	for _, spec := range arm.RegisterSpec() {
		if register >= spec.Start && register <= spec.End {
			switch spec.Name {
			case coprocessor.ExtendedRegisterCoreGroup:
				arm.state.registers[register] = value
				return true
			case "FPU":
				arm.state.fpu.Registers[register-spec.Start] = value
				return true
			case "TIM2":
				switch register {
				case 10000:
					arm.state.timer2.Write(arm.mmap.TIM2CR1, value)
				case 10001:
					arm.state.timer2.Write(arm.mmap.TIM2CNT, value)
				default:
					return false
				}
				return true
			}
		}
	}
	fmt.Println(1)
	return false
}
