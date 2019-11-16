package registers_test

import (
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/cpu/registers/assert"
	"testing"
)

func TestRegister(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r8 := registers.NewRegister(0, "test")
	assert.Assert(t, r8.IsZero(), true)
	assert.Assert(t, r8, 0)

	// loading & addition
	r8.Load(127)
	assert.Assert(t, r8, 127)
	r8.Add(2, false)
	assert.Assert(t, r8, 129)

	// addtion boundary
	r8.Load(255)
	assert.Assert(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, false)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
	assert.Assert(t, r8.IsZero(), true)
	assert.Assert(t, r8, 0)

	// addition boundary with carry
	r8.Load(254)
	assert.Assert(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, true)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
	assert.Assert(t, r8.IsZero(), true)
	assert.Assert(t, r8, 0)

	// addition boundary with carry
	r8.Load(255)
	assert.Assert(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, true)
	assert.Assert(t, carry, true)
	assert.Assert(t, overflow, false)
	assert.Assert(t, r8.IsZero(), false)
	assert.Assert(t, r8, 1)

	// subtraction
	r8.Load(11)
	r8.Subtract(1, true)
	assert.Assert(t, r8, 10)

	r8.Load(12)
	r8.Subtract(1, false)
	assert.Assert(t, r8, 10)

	r8.Load(0x01)
	r8.Subtract(0x06, false)
	assert.Assert(t, r8, 0xFA)

	// subtract on boundary
	r8.Load(0)
	r8.Subtract(1, true)
	assert.Assert(t, r8, 255)
	r8.Load(1)
	r8.Subtract(1, false)
	assert.Assert(t, r8, 255)
	r8.Load(1)
	r8.Subtract(2, true)
	assert.Assert(t, r8, 255)

	// logical operators
	r8.Load(0x21)
	r8.AND(0x01)
	assert.Assert(t, r8, 0x01)
	r8.EOR(0xFF)
	assert.Assert(t, r8, 0xFE)
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
