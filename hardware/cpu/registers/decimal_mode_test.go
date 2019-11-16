package registers_test

import (
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/cpu/registers/assert"
	"testing"
)

func TestDecimalMode(t *testing.T) {
	var rcarry bool

	// initialisation
	r8 := registers.NewRegister(0, "test")

	// addition without carry
	rcarry = r8.AddDecimal(1, false)
	assert.Assert(t, r8, 0x01)
	assert.Assert(t, rcarry, false)

	// addition with carry
	rcarry = r8.AddDecimal(1, true)
	assert.Assert(t, r8, 0x03)
	assert.Assert(t, rcarry, false)

	// subtraction with carry (subtract value)
	r8.Load(9)
	assert.Assert(t, r8, 0x09)
	rcarry = r8.SubtractDecimal(1, true)
	assert.Assert(t, r8, 0x08)

	// subtraction without carry (subtract value and another 1)
	rcarry = r8.SubtractDecimal(1, false)
	assert.Assert(t, r8, 0x06)

	// addition on tens boundary
	r8.Load(9)
	assert.Assert(t, r8, 0x09)
	rcarry = r8.AddDecimal(1, false)
	assert.Assert(t, r8, 0x10)

	// subtraction on tens boundary
	rcarry = r8.SubtractDecimal(1, true)
	assert.Assert(t, r8, 0x09)

	// addition on hundreds boundary
	r8.Load(0x99)
	assert.Assert(t, r8, 0x99)
	rcarry = r8.AddDecimal(1, false)
	assert.Assert(t, r8, 0x00)
	assert.Assert(t, rcarry, true)

	// subtraction on hundreds boundary
	rcarry = r8.SubtractDecimal(1, true)
	assert.Assert(t, r8, 0x99)
}
