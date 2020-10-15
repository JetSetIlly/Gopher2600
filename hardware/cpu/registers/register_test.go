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

func TestRegister(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r8 := registers.NewRegister(0, "test")
	test.Equate(t, r8.IsZero(), true)
	rtest.EquateRegisters(t, r8, 0)

	// loading & addition
	r8.Load(127)
	rtest.EquateRegisters(t, r8, 127)
	r8.Add(2, false)
	rtest.EquateRegisters(t, r8, 129)

	// addition boundary
	r8.Load(255)
	test.Equate(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, false)
	test.Equate(t, carry, true)
	test.Equate(t, overflow, false)
	test.Equate(t, r8.IsZero(), true)
	rtest.EquateRegisters(t, r8, 0)

	// addition boundary with carry
	r8.Load(254)
	test.Equate(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, true)
	test.Equate(t, carry, true)
	test.Equate(t, overflow, false)
	test.Equate(t, r8.IsZero(), true)
	rtest.EquateRegisters(t, r8, 0)

	// addition boundary with carry
	r8.Load(255)
	test.Equate(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, true)
	test.Equate(t, carry, true)
	test.Equate(t, overflow, false)
	test.Equate(t, r8.IsZero(), false)
	rtest.EquateRegisters(t, r8, 1)

	// subtraction
	r8.Load(11)
	r8.Subtract(1, true)
	rtest.EquateRegisters(t, r8, 10)

	r8.Load(12)
	r8.Subtract(1, false)
	rtest.EquateRegisters(t, r8, 10)

	r8.Load(0x01)
	r8.Subtract(0x06, false)
	rtest.EquateRegisters(t, r8, 0xFA)

	// subtract on boundary
	r8.Load(0)
	r8.Subtract(1, true)
	rtest.EquateRegisters(t, r8, 255)
	r8.Load(1)
	r8.Subtract(1, false)
	rtest.EquateRegisters(t, r8, 255)
	r8.Load(1)
	r8.Subtract(2, true)
	rtest.EquateRegisters(t, r8, 255)

	// logical operators
	r8.Load(0x21)
	r8.AND(0x01)
	rtest.EquateRegisters(t, r8, 0x01)
	r8.EOR(0xFF)
	rtest.EquateRegisters(t, r8, 0xFE)
	r8.ORA(0x1)
	rtest.EquateRegisters(t, r8, 0xFF)

	// shifts
	carry = r8.ASL()
	rtest.EquateRegisters(t, r8, 0xFE)
	test.Equate(t, carry, true)
	carry = r8.LSR()
	rtest.EquateRegisters(t, r8, 0x007F)
	test.Equate(t, carry, false)
	carry = r8.LSR()
	test.Equate(t, carry, true)

	// rotation
	r8.Load(0xff)
	carry = r8.ROL(false)
	rtest.EquateRegisters(t, r8, 0xfe)
	test.Equate(t, carry, true)
	carry = r8.ROR(true)
	rtest.EquateRegisters(t, r8, 0xff)
	test.Equate(t, carry, false)
}
