package registers_test

import (
	"gopher2600/hardware/cpu/registers"
	rtest "gopher2600/hardware/cpu/registers/test"
	"gopher2600/test"
	"testing"
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

	// addtion boundary
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
