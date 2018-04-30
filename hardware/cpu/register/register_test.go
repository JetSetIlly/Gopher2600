package register_test

import (
	"gopher2600/hardware/cpu/register"
	"gopher2600/mflib"
	"testing"
)

func TestRegister(t *testing.T) {
	r16(t)
	r8(t)
}

func r16(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r16, err := register.NewRegister(0, 16, "TEST")
	if err != nil {
		t.Fatalf(err.Error())
	}
	mflib.Assert(t, r16.IsZero(), true)
	mflib.Assert(t, r16, 0)

	// loading & addition
	r16.Load(127)
	mflib.Assert(t, r16, 127)
	r16.Add(2, false)
	mflib.Assert(t, r16, 129)
	mflib.Assert(t, r16, "0000000010000001")

	r16.Load(0xffff)
	mflib.Assert(t, r16.IsNegative(), true)
	carry, overflow = r16.Add(1, false)
	mflib.Assert(t, carry, true)
	mflib.Assert(t, overflow, false)
	mflib.Assert(t, r16.IsZero(), true)

	// register operand
	r16b, err := register.NewRegister(10, 16, "TEST B")
	if err != nil {
		t.Fatalf(err.Error())
	}
	mflib.Assert(t, r16b, 10)
	r16.Add(r16b, true)
	mflib.Assert(t, r16, 11)

	// subtraction
	r16.Subtract(1, true)
	mflib.Assert(t, r16, 10)
	r16.Subtract(10, true)
	mflib.Assert(t, r16.IsZero(), true)
	mflib.Assert(t, r16.IsNegative(), false)
	r16.Subtract(1, true)
	mflib.Assert(t, r16.IsZero(), false)
	mflib.Assert(t, r16.IsNegative(), true)

	// logical operators
	r16.Load(0x21)
	r16.AND(0x01)
	mflib.Assert(t, r16, 0x01)
	r16.EOR(0xFFFF)
	mflib.Assert(t, r16, 0xFFFE)
	r16.ORA(0x1)
	mflib.Assert(t, r16, 0xFFFF)

	// shifts
	r16.Load(0xFF)
	carry = r16.ASL()
	mflib.Assert(t, r16, 0x01FE)
	mflib.Assert(t, carry, false)
	carry = r16.LSR()
	mflib.Assert(t, r16, 0x00FF)
	mflib.Assert(t, carry, false)
	carry = r16.LSR()
	mflib.Assert(t, carry, true)

	// rotation
	r16.Load(0xff)
	carry = r16.ROL(false)
	mflib.Assert(t, r16, 0x1fe)
	mflib.Assert(t, carry, false)
	carry = r16.ROR(true)
	mflib.Assert(t, r16, 0x80ff)
	mflib.Assert(t, carry, false)
}

func r8(t *testing.T) {
	var carry, overflow bool

	// initialisation
	r8, err := register.NewRegister(0, 8, "TEST")
	if err != nil {
		t.Fatalf(err.Error())
	}
	mflib.Assert(t, r8.IsZero(), true)
	mflib.Assert(t, r8, 0)

	// loading & addition
	r8.Load(127)
	mflib.Assert(t, r8, 127)
	r8.Add(2, false)
	mflib.Assert(t, r8, 129)
	mflib.Assert(t, r8, "10000001")

	r8.Load(0xff)
	mflib.Assert(t, r8.IsNegative(), true)
	carry, overflow = r8.Add(1, false)
	mflib.Assert(t, carry, true)
	mflib.Assert(t, overflow, false)
	mflib.Assert(t, r8.IsZero(), true)

	// register operand
	r8b, err := register.NewRegister(10, 8, "TEST B")
	if err != nil {
		t.Fatalf(err.Error())
	}
	mflib.Assert(t, r8b, 10)
	r8.Add(r8b, true)
	mflib.Assert(t, r8, 11)

	// subtraction
	r8.Subtract(1, true)
	mflib.Assert(t, r8, 10)

	// logical operators
	r8.Load(0x21)
	r8.AND(0x01)
	mflib.Assert(t, r8, 0x01)
	r8.EOR(0xFFFF)
	// note how we're EORing with a 16 bit value but the test
	// against an 8 value is correct (because the register is 8 bit)
	mflib.Assert(t, r8, 0x00FE)
	r8.ORA(0x1)
	mflib.Assert(t, r8, 0xFF)

	// shifts
	carry = r8.ASL()
	mflib.Assert(t, r8, 0xFE)
	mflib.Assert(t, carry, true)
	carry = r8.LSR()
	mflib.Assert(t, r8, 0x007F)
	mflib.Assert(t, carry, false)
	carry = r8.LSR()
	mflib.Assert(t, carry, true)

	// rotation
	r8.Load(0xff)
	carry = r8.ROL(false)
	mflib.Assert(t, r8, 0xfe)
	mflib.Assert(t, carry, true)
	carry = r8.ROR(true)
	mflib.Assert(t, r8, 0xff)
	mflib.Assert(t, carry, false)
}
