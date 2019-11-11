package register_test

import (
	"gopher2600/hardware/cpu/register"
	"gopher2600/hardware/cpu/register/assert"
	"testing"
)

func TestRegister(t *testing.T) {
	r16(t)
	r8(t)
	r4(t)
}

func TestSubtract(t *testing.T) {
	r8 := register.NewRegister(0, 8, "TEST")
	r8.Load(0)
	r8.Subtract(0, true)
	assert.Assert(t, r8, 0)
	r8.Subtract(0, false)
	assert.Assert(t, r8, 255)
}

func r16(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r16 := register.NewRegister(0, 16, "TEST")
	assert.Assert(t, r16.IsZero(), true)
	assert.Assert(t, r16, 0)

	// loading & addition
	r16.Load(127)
	assert.Assert(t, r16, 127)
	r16.Add(2, false)
	assert.Assert(t, r16, 129)
	assert.Assert(t, r16, "0000000010000001b")

	r16.Load(0xffff)
	assert.Assert(t, r16.IsNegative(), true)
	carry, overflow = r16.Add(1, false)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
	assert.Assert(t, r16.IsZero(), true)

	// register operand
	r16b := register.NewRegister(10, 16, "TEST B")
	assert.Assert(t, r16b, 10)
	r16.Add(r16b, true)
	assert.Assert(t, r16, 11)

	// subtraction
	r16.Subtract(1, true)
	assert.Assert(t, r16, 10)
	r16.Subtract(10, true)
	assert.Assert(t, r16.IsZero(), true)
	assert.Assert(t, r16.IsNegative(), false)
	r16.Subtract(1, true)
	assert.Assert(t, r16.IsZero(), false)
	assert.Assert(t, r16.IsNegative(), true)
	r16.Subtract(1, true)
	assert.Assert(t, r16, 0xFFFE)

	r16.Load(0x01)
	r16.Subtract(2, true)
	assert.Assert(t, r16, 0xFFFF)

	// logical operators
	r16.Load(0x21)
	r16.AND(0x01)
	assert.Assert(t, r16, 0x01)
	r16.EOR(0xFFFF)
	assert.Assert(t, r16, 0xFFFE)
	r16.ORA(0x1)
	assert.Assert(t, r16, 0xFFFF)

	// shifts
	r16.Load(0xFF)
	carry = r16.ASL()
	assert.Assert(t, r16, 0x01FE)
	assert.Assert(t, carry, false)
	carry = r16.LSR()
	assert.Assert(t, r16, 0x00FF)
	assert.Assert(t, carry, false)
	carry = r16.LSR()
	assert.Assert(t, carry, true)

	// rotation
	r16.Load(0xff)
	carry = r16.ROL(false)
	assert.Assert(t, r16, 0x1fe)
	assert.Assert(t, carry, false)
	carry = r16.ROR(true)
	assert.Assert(t, r16, 0x80ff)
	assert.Assert(t, carry, false)
}

func r8(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r8 := register.NewRegister(0, 8, "TEST")
	assert.Assert(t, r8.IsZero(), true)
	assert.Assert(t, r8, 0)

	// loading & addition
	r8.Load(127)
	assert.Assert(t, r8, 127)
	r8.Add(2, false)
	assert.Assert(t, r8, 129)
	assert.Assert(t, r8, "10000001b")

	r8.Load(0xff)
	assert.Assert(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, false)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
	assert.Assert(t, r8.IsZero(), true)

	// register operand
	r8b := register.NewRegister(10, 8, "TEST B")
	assert.Assert(t, r8b, 10)
	r8.Add(r8b, true)
	assert.Assert(t, r8, 11)

	// subtraction
	r8.Subtract(1, true)
	assert.Assert(t, r8, 10)

	r8.Load(0x01)
	r8.Subtract(0x06, false)
	assert.Assert(t, r8, 0xFA)

	// logical operators
	r8.Load(0x21)
	r8.AND(0x01)
	assert.Assert(t, r8, 0x01)
	r8.EOR(0xFFFF)
	// note how we're EORing with a 16 bit value but the test
	// against an 8 value is correct (because the register is 8 bit)
	assert.Assert(t, r8, 0x00FE)
	r8.ORA(0x1)
	assert.Assert(t, r8, 0xFF)

	// shifts
	carry = r8.ASL()
	assert.Assert(t, r8, 0xFE)
	assert.Assert(t, carry, true)
	carry = r8.LSR()
	assert.Assert(t, r8, 0x007F)
	assert.Assert(t, carry, false)
	carry = r8.LSR()
	assert.Assert(t, carry, true)

	// rotation
	r8.Load(0xff)
	carry = r8.ROL(false)
	assert.Assert(t, r8, 0xfe)
	assert.Assert(t, carry, true)
	carry = r8.ROR(true)
	assert.Assert(t, r8, 0xff)
	assert.Assert(t, carry, false)
}

func r4(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r4 := register.NewRegister(0, 4, "TEST")
	assert.Assert(t, r4.IsZero(), true)
	assert.Assert(t, r4, 0)

	r4.Load(0xff)
	assert.Assert(t, r4, 15)
	carry, overflow = r4.Add(1, false)
	assert.Assert(t, r4, 0)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
}
