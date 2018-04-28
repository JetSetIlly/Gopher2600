package rbits_test

import (
	"headlessVCS/hardware/cpu/registers/rbits"
	"headlessVCS/mflib"
	"testing"
)

func TestRegister(t *testing.T) {
	r8, err := rbits.Generate(0, 8, "test 8bit")
	if err != nil {
		t.Fatalf(err.Error())
	}
	r16, err := rbits.Generate(0, 16, "test 16bit")
	if err != nil {
		t.Fatalf(err.Error())
	}

	var carry, overflow bool

	// register loading
	r8.Load(127)
	mflib.Assert(t, r8, "01111111")
	mflib.Assert(t, r8, 127)

	r8.Load(r8)
	mflib.Assert(t, r8, "01111111")

	r16.Load(1)
	mflib.Assert(t, r16, "0000000000000001")
	mflib.Assert(t, r16, 1)

	r16.Load(256)
	mflib.Assert(t, r16, "0000000100000000")
	mflib.Assert(t, r16, 256)

	r16.Load(1024)
	mflib.Assert(t, r16, "0000010000000000")
	mflib.Assert(t, r16, 1024)

	// arithemtic
	r16.Add(2, false)
	mflib.Assert(t, r16, "0000010000000010")
	mflib.Assert(t, r16, 1026)

	r16.Subtract(2, true)
	mflib.Assert(t, r16, "0000010000000000")
	mflib.Assert(t, r16, 1024)

	r8.Add(1, false)
	r8.Subtract(1, true)
	mflib.Assert(t, r8, "01111111")

	// arithmetic - carry/overflow
	r8.Load(255)
	mflib.Assert(t, r8, 255)
	carry, overflow = r8.Add(1, false)
	mflib.Assert(t, r8, 0)
	mflib.Assert(t, carry, true)
	mflib.Assert(t, overflow, false)
	carry, overflow = r8.Subtract(1, true)
	mflib.Assert(t, r8, 255)
	mflib.Assert(t, carry, false)
	mflib.Assert(t, overflow, false)
	carry, overflow = r8.Subtract(1, true)
	mflib.Assert(t, r8, 254)
	mflib.Assert(t, carry, true)
	mflib.Assert(t, overflow, false)

	// bitwise
	r16.EOR(65535)
	mflib.Assert(t, r16, "1111101111111111")

	r16.ORA(65535)
	mflib.Assert(t, r16, "1111111111111111")

	r16.AND(253)
	mflib.Assert(t, r16, "0000000011111101")

	r8.Load(1)
	mflib.Assert(t, r8, "00000001")

	// rotation
	r8.ROL(false)
	mflib.Assert(t, r8, "00000010")

	r8.ROL(true)
	mflib.Assert(t, r8, "00000101")

	r8.ROR(true)
	mflib.Assert(t, r8, "10000010")

	r8.ROR(false)
	mflib.Assert(t, r8, "01000001")

	// flags
	mflib.Assert(t, r8.IsZero(), false)
	mflib.Assert(t, r8.IsNegative(), false)
	r8.ROL(false)
	mflib.Assert(t, r8.IsNegative(), true)
	r8.Load(0)
	mflib.Assert(t, r8.IsZero(), true)

	r8.Load(255)
	carry, overflow = r8.Add(2, false)
	mflib.Assert(t, carry, true)
	mflib.Assert(t, overflow, false)

	// addition of different sized registers
	r8.Load(1)
	r16.Load(255)
	r16.Add(r8, false)
	mflib.Assert(t, r16, 256)
}
