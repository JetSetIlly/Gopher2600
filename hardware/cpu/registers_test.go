package cpu_test

import (
	"headlessVCS/hardware/cpu"
	"testing"
)

func TestRegister(t *testing.T) {
	r8 := make(cpu.Register, 8)
	r16 := make(cpu.Register, 16)

	var carry, overflow bool

	// register loading
	r8.Load(127)
	assert(t, r8, "01111111")
	assert(t, r8, 127)

	r8.Load(r8)
	assert(t, r8, "01111111")

	r16.Load(1)
	assert(t, r16, "0000000000000001")
	assert(t, r16, 1)

	r16.Load(256)
	assert(t, r16, "0000000100000000")
	assert(t, r16, 256)

	r16.Load(1024)
	assert(t, r16, "0000010000000000")
	assert(t, r16, 1024)

	// arithemtic
	r16.Add(2, false)
	assert(t, r16, "0000010000000010")
	assert(t, r16, 1026)

	r16.Subtract(2, true)
	assert(t, r16, "0000010000000000")
	assert(t, r16, 1024)

	r8.Add(1, false)
	r8.Subtract(1, true)
	assert(t, r8, "01111111")

	// arithmetic - carry/overflow
	r8.Load(255)
	assert(t, r8, 255)
	carry, overflow = r8.Add(1, false)
	assert(t, r8, 0)
	assert(t, carry, true)
	assert(t, overflow, false)
	carry, overflow = r8.Subtract(1, true)
	assert(t, r8, 255)
	assert(t, carry, false)
	assert(t, overflow, false)
	carry, overflow = r8.Subtract(1, true)
	assert(t, r8, 254)
	assert(t, carry, true)
	assert(t, overflow, false)

	// bitwise
	r16.EOR(65535)
	assert(t, r16, "1111101111111111")

	r16.ORA(65535)
	assert(t, r16, "1111111111111111")

	r16.AND(253)
	assert(t, r16, "0000000011111101")

	r8.Load(1)
	assert(t, r8, "00000001")

	// rotation
	r8.ROL(false)
	assert(t, r8, "00000010")

	r8.ROL(true)
	assert(t, r8, "00000101")

	r8.ROR(true)
	assert(t, r8, "10000010")

	r8.ROR(false)
	assert(t, r8, "01000001")

	// cpu flags
	assert(t, r8.IsZero(), false)
	assert(t, r8.IsNegative(), false)
	r8.ROL(false)
	assert(t, r8.IsNegative(), true)
	r8.Load(0)
	assert(t, r8.IsZero(), true)

	r8.Load(255)
	carry, overflow = r8.Add(2, false)
	assert(t, carry, true)
	assert(t, overflow, false)

	// addition of different sized registers
	r8.Load(1)
	r16.Load(255)
	r16.Add(r8, false)
	assert(t, r16, 256)
}
