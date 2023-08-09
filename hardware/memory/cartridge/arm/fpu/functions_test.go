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

package fpu_test

import (
	"math"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/fpu"
	"github.com/jetsetilly/gopher2600/test"
)

func TestSpecificValues(t *testing.T) {
	var fp fpu.FPU
	var v32 uint32

	v32 = uint32(fp.FPZero(false, 32))
	test.ExpectEquality(t, v32, 0b00000000000000000000000000000000)
	v32 = uint32(fp.FPZero(true, 32))
	test.ExpectEquality(t, v32, 0b10000000000000000000000000000000)

	v32 = uint32(fp.FPInfinity(false, 32))
	test.ExpectEquality(t, v32, 0b01111111100000000000000000000000)
	v32 = uint32(fp.FPInfinity(true, 32))
	test.ExpectEquality(t, v32, 0b11111111100000000000000000000000)

	v32 = uint32(fp.FPMaxNormal(false, 32))
	test.ExpectEquality(t, v32, 0b01111111011111111111111111111111)
	v32 = uint32(fp.FPMaxNormal(true, 32))
	test.ExpectEquality(t, v32, 0b11111111011111111111111111111111)

	v32 = uint32(fp.FPDefaultNaN(32))
	test.ExpectEquality(t, v32, 0b01111111110000000000000000000000)
}

func TestUnpack(t *testing.T) {
	var fp fpu.FPU
	var typ fpu.FPType
	var val float64

	fpscr := fp.StandardFPSCRValue()

	typ, _, val = fp.FPUnpack(0, 32, fpscr)
	test.ExpectEquality(t, typ, fpu.FPType_Zero)
	test.ExpectEquality(t, val, 0.0)

	typ, _, _ = fp.FPUnpack(0b01111111100000000000000000000000, 32, fpscr)
	test.ExpectEquality(t, typ, fpu.FPType_Infinity)
}

func TestRound(t *testing.T) {
	var fp fpu.FPU
	var v float64
	var b uint64
	var c uint32

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	v = 1.0
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = -1.0
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = 10.0
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = -10.0
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = 1000000.0
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = math.Pi
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)

	v = math.E
	b = fp.FPRound(v, 32, fpscr)
	c = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(b), c)
}

func TestRoundToUnpack(t *testing.T) {
	var fp fpu.FPU
	var v float64
	var b uint64
	var c float64
	var typ fpu.FPType
	var sign bool

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	v = 1.0
	b = fp.FPRound(v, 32, fpscr)
	typ, sign, c = fp.FPUnpack(b, 32, fpscr)
	test.ExpectEquality(t, typ, fpu.FPType_Nonzero)
	test.ExpectEquality(t, sign, false)
	test.ExpectEquality(t, c, v)

	v = -10.0
	b = fp.FPRound(v, 32, fpscr)
	typ, sign, c = fp.FPUnpack(b, 32, fpscr)
	test.ExpectEquality(t, typ, fpu.FPType_Nonzero)
	test.ExpectEquality(t, sign, true)
	test.ExpectEquality(t, c, v)

	v = math.Pi
	b = fp.FPRound(v, 32, fpscr)
	typ, sign, c = fp.FPUnpack(b, 32, fpscr)
	test.ExpectEquality(t, sign, false)
	test.ExpectEquality(t, typ, fpu.FPType_Nonzero)
	// 32 bits is not enough to preserve accuracy for Pi

	v = math.Pi
	b = fp.FPRound(v, 64, fpscr)
	typ, sign, c = fp.FPUnpack(b, 64, fpscr)
	test.ExpectEquality(t, sign, false)
	test.ExpectEquality(t, typ, fpu.FPType_Nonzero)
	test.ExpectEquality(t, c, v)
}

func TestFixedToFP(t *testing.T) {
	var fp fpu.FPU
	var c uint64

	c = fp.FixedToFP(0, 32, 0, false, true, true)
	test.ExpectEquality(t, c, fp.FPZero(false, 32))

	var v uint64

	v = 64
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

	v = 1000
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

	v = 1000000
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

	// 64bit
	v = 1000000
	c = fp.FixedToFP(v, 64, 0, false, true, true)
	test.ExpectEquality(t, c, math.Float64bits(float64(v)))
}

func TestFPToFixed(t *testing.T) {
	var fp fpu.FPU
	var v uint64
	var c uint64

	v = fp.FPZero(false, 32)
	c = fp.FPToFixed(v, 32, 0, false, true, true)
	test.ExpectEquality(t, c, 0)

	var d uint64

	v = 64
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	d = fp.FPToFixed(c, 32, 0, false, true, true)
	test.ExpectEquality(t, d, v)
}

func TestNegative(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	var v float64
	var c uint64
	var d uint32

	v = -100
	c = fp.FPRound(v, 32, fpscr)
	d = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(c), d)

	v = -100.1011
	c = fp.FPRound(v, 32, fpscr)
	d = math.Float32bits(float32(v))
	test.ExpectEquality(t, uint32(c), d)
}

func TestArithmetic(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	var v, w float64
	var c, d uint64
	v = 123.12
	c = fp.FPRound(v, 64, fpscr)
	w = 456.842
	d = fp.FPRound(w, 64, fpscr)

	var r, s uint64

	// addition
	r = fp.FPAdd(c, d, 64, false)
	s = math.Float64bits(v + w)
	test.ExpectEquality(t, r, s)

	// subtraction
	r = fp.FPSub(c, d, 64, false)
	s = math.Float64bits(v - w)
	test.ExpectEquality(t, r, s)

	// multiplication
	r = fp.FPMul(c, d, 64, false)
	s = math.Float64bits(v * w)
	test.ExpectEquality(t, r, s)

	// division
	r = fp.FPDiv(c, d, 64, false)
	s = math.Float64bits(v / w)
	test.ExpectEquality(t, r, s)

	var q uint64

	// mutliplication and add
	r = fp.FPRound(2, 32, fpscr)
	s = fp.FPRound(3, 32, fpscr)
	q = fp.FPRound(1, 32, fpscr)
	q = fp.FPMulAdd(q, r, s, 32, false)
	_, _, f := fp.FPUnpack(q, 32, fpscr)
	test.ExpectEquality(t, f, (2*3)+1)
}

func TestNegation(t *testing.T) {
	var fp fpu.FPU

	var v float64
	var c uint32
	var d uint32

	v = 100.223
	c = math.Float32bits(float32(v))
	d = math.Float32bits(float32(-v))

	// the two values should be unequal at this point
	test.ExpectInequality(t, c, d)

	// negate one of the values. the two value will now be equal
	d = uint32(fp.FPNeg(uint64(d), 32))
	test.ExpectEquality(t, c, d)

	// negate again to make the values unequal
	d = uint32(fp.FPNeg(uint64(d), 32))
	test.ExpectInequality(t, c, d)

	// and again to make them equal again
	d = uint32(fp.FPNeg(uint64(d), 32))
	test.ExpectEquality(t, c, d)
}

func TestAbsolute(t *testing.T) {
	var fp fpu.FPU

	var v float64
	var c uint32
	var d uint32

	v = 100.223
	c = math.Float32bits(float32(v))
	d = math.Float32bits(float32(-v))

	// the two values should be unequal at this point
	test.ExpectInequality(t, c, d)

	// force the negative value to be positive
	d = uint32(fp.FPAbs(uint64(d), 32))
	test.ExpectEquality(t, c, d)

	// forcing a positive value has no effect
	d = uint32(fp.FPAbs(uint64(d), 32))
	test.ExpectEquality(t, c, d)
}

func TestImmediate(t *testing.T) {
	var fp fpu.FPU

	// test taken from an real world example of a VMOV (immediate) instruction.
	// the GCC objdump indiates that a value of 0x50 should expand to 0.25
	a := fp.VFPExpandImm(0x50, 32)
	b := math.Float32frombits(uint32(a))
	test.ExpectEquality(t, b, 0.25)
}
