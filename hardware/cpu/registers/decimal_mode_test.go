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

package registers_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	rtest "github.com/jetsetilly/gopher2600/hardware/cpu/registers/test"
	"github.com/jetsetilly/gopher2600/test"
)

func TestDecimalModeCarry(t *testing.T) {
	var rcarry bool

	// initialisation
	r8 := registers.NewRegister(0, "test")

	// addition without carry
	rcarry, _, _, _ = r8.AddDecimal(1, false)
	rtest.EquateRegisters(t, r8, 0x01)
	test.Equate(t, rcarry, false)

	// addition with carry
	rcarry, _, _, _ = r8.AddDecimal(1, true)
	rtest.EquateRegisters(t, r8, 0x03)
	test.Equate(t, rcarry, false)

	// subtraction with carry (subtract value)
	r8.Load(9)
	rtest.EquateRegisters(t, r8, 0x09)
	r8.SubtractDecimal(1, true)
	rtest.EquateRegisters(t, r8, 0x08)

	// subtraction without carry (subtract value and another 1)
	r8.SubtractDecimal(1, false)
	rtest.EquateRegisters(t, r8, 0x06)

	// addition on tens boundary
	r8.Load(9)
	rtest.EquateRegisters(t, r8, 0x09)
	r8.AddDecimal(1, false)
	rtest.EquateRegisters(t, r8, 0x10)

	// subtraction on tens boundary
	r8.SubtractDecimal(1, true)
	rtest.EquateRegisters(t, r8, 0x09)

	// addition on hundreds boundary
	r8.Load(0x99)
	rtest.EquateRegisters(t, r8, 0x99)
	rcarry, _, _, _ = r8.AddDecimal(1, false)
	rtest.EquateRegisters(t, r8, 0x00)
	test.Equate(t, rcarry, true)

	// subtraction on hundreds boundary
	r8.SubtractDecimal(1, true)
	rtest.EquateRegisters(t, r8, 0x99)
}

func TestDecimalModeZero(t *testing.T) {
	var zero bool

	// initialisation
	r8 := registers.NewRegister(0, "test")

	// subtract to zero
	r8.Load(0x02)
	_, zero, _, _ = r8.SubtractDecimal(1, true)
	test.Equate(t, zero, false)
	_, zero, _, _ = r8.SubtractDecimal(1, true)
	test.Equate(t, zero, true)
}

func TestDecimalModeInvalid(t *testing.T) {
	var rcarry, rzero bool

	r8 := registers.NewRegister(0x99, "test")
	rcarry, rzero, _, _ = r8.AddDecimal(1, false)
	rtest.EquateRegisters(t, r8, 0x00)
	test.Equate(t, rcarry, true)
	test.Equate(t, rzero, false)
}
