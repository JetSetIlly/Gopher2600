package register_test

import (
	"gopher2600/assert"
	"gopher2600/hardware/cpu/register"
	"testing"
)

func TestRegister(t *testing.T) {
	r16(t)
	r8(t)
}

func r16(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r16, err := register.New(0, 16, "TEST", "TST")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.CheckValueVCS(t, r16.IsZero(), true)
	assert.CheckValueVCS(t, r16, 0)

	// loading & addition
	r16.Load(127)
	assert.CheckValueVCS(t, r16, 127)
	r16.Add(2, false)
	assert.CheckValueVCS(t, r16, 129)
	assert.CheckValueVCS(t, r16, "0000000010000001")

	r16.Load(0xffff)
	assert.CheckValueVCS(t, r16.IsNegative(), true)
	carry, overflow = r16.Add(1, false)
	assert.CheckValueVCS(t, carry, true)
	assert.CheckValueVCS(t, overflow, false)
	assert.CheckValueVCS(t, r16.IsZero(), true)

	// register operand
	r16b, err := register.New(10, 16, "TEST B", "TSTB")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.CheckValueVCS(t, r16b, 10)
	r16.Add(r16b, true)
	assert.CheckValueVCS(t, r16, 11)

	// subtraction
	r16.Subtract(1, true)
	assert.CheckValueVCS(t, r16, 10)
	r16.Subtract(10, true)
	assert.CheckValueVCS(t, r16.IsZero(), true)
	assert.CheckValueVCS(t, r16.IsNegative(), false)
	r16.Subtract(1, true)
	assert.CheckValueVCS(t, r16.IsZero(), false)
	assert.CheckValueVCS(t, r16.IsNegative(), true)

	// logical operators
	r16.Load(0x21)
	r16.AND(0x01)
	assert.CheckValueVCS(t, r16, 0x01)
	r16.EOR(0xFFFF)
	assert.CheckValueVCS(t, r16, 0xFFFE)
	r16.ORA(0x1)
	assert.CheckValueVCS(t, r16, 0xFFFF)

	// shifts
	r16.Load(0xFF)
	carry = r16.ASL()
	assert.CheckValueVCS(t, r16, 0x01FE)
	assert.CheckValueVCS(t, carry, false)
	carry = r16.LSR()
	assert.CheckValueVCS(t, r16, 0x00FF)
	assert.CheckValueVCS(t, carry, false)
	carry = r16.LSR()
	assert.CheckValueVCS(t, carry, true)

	// rotation
	r16.Load(0xff)
	carry = r16.ROL(false)
	assert.CheckValueVCS(t, r16, 0x1fe)
	assert.CheckValueVCS(t, carry, false)
	carry = r16.ROR(true)
	assert.CheckValueVCS(t, r16, 0x80ff)
	assert.CheckValueVCS(t, carry, false)
}

func r8(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r8, err := register.New(0, 8, "TEST", "TST")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.CheckValueVCS(t, r8.IsZero(), true)
	assert.CheckValueVCS(t, r8, 0)

	// loading & addition
	r8.Load(127)
	assert.CheckValueVCS(t, r8, 127)
	r8.Add(2, false)
	assert.CheckValueVCS(t, r8, 129)
	assert.CheckValueVCS(t, r8, "10000001")

	r8.Load(0xff)
	assert.CheckValueVCS(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, false)
	assert.CheckValueVCS(t, carry, true)
	assert.CheckValueVCS(t, overflow, false)
	assert.CheckValueVCS(t, r8.IsZero(), true)

	// register operand
	r8b, err := register.New(10, 8, "TEST B", "TSTB")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.CheckValueVCS(t, r8b, 10)
	r8.Add(r8b, true)
	assert.CheckValueVCS(t, r8, 11)

	// subtraction
	r8.Subtract(1, true)
	assert.CheckValueVCS(t, r8, 10)

	// logical operators
	r8.Load(0x21)
	r8.AND(0x01)
	assert.CheckValueVCS(t, r8, 0x01)
	r8.EOR(0xFFFF)
	// note how we're EORing with a 16 bit value but the test
	// against an 8 value is correct (because the register is 8 bit)
	assert.CheckValueVCS(t, r8, 0x00FE)
	r8.ORA(0x1)
	assert.CheckValueVCS(t, r8, 0xFF)

	// shifts
	carry = r8.ASL()
	assert.CheckValueVCS(t, r8, 0xFE)
	assert.CheckValueVCS(t, carry, true)
	carry = r8.LSR()
	assert.CheckValueVCS(t, r8, 0x007F)
	assert.CheckValueVCS(t, carry, false)
	carry = r8.LSR()
	assert.CheckValueVCS(t, carry, true)

	// rotation
	r8.Load(0xff)
	carry = r8.ROL(false)
	assert.CheckValueVCS(t, r8, 0xfe)
	assert.CheckValueVCS(t, carry, true)
	carry = r8.ROR(true)
	assert.CheckValueVCS(t, r8, 0xff)
	assert.CheckValueVCS(t, carry, false)
}
