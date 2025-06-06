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

	// add a small fraction and see how FPToFixed() converts back to an integer
	var delta float64

	delta = 0.25
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	c = fp.FPAdd(c, fp.FPRound(delta, 32, fp.Status), 32, true)
	d = fp.FPToFixed(c, 32, 0, false, true, true)
	test.ExpectEquality(t, d, v)

	delta = 0.99
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	c = fp.FPAdd(c, fp.FPRound(delta, 32, fp.Status), 32, true)
	d = fp.FPToFixed(c, 32, 0, false, true, true)
	test.ExpectEquality(t, d, v)

	delta = 1.01
	c = fp.FixedToFP(v, 32, 0, false, true, true)
	c = fp.FPAdd(c, fp.FPRound(delta, 32, fp.Status), 32, true)
	d = fp.FPToFixed(c, 32, 0, false, true, true)
	test.ExpectInequality(t, d, v)
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

func TestImmediate(t *testing.T) {
	var fp fpu.FPU
	var a uint64
	var b float32

	a = fp.VFPExpandImm(0x00, 32)
	b = math.Float32frombits(uint32(a))
	test.ExpectEquality(t, b, 0.0)

	// tests taken from an real world example of a VMOV (immediate) instruction.
	// the GCC objdump indiates that a value of 0x50 should expand to 0.25
	a = fp.VFPExpandImm(0x50, 32)
	b = math.Float32frombits(uint32(a))
	test.ExpectEquality(t, b, 0.25)

	a = fp.VFPExpandImm(0x70, 32)
	b = math.Float32frombits(uint32(a))
	test.ExpectEquality(t, b, 1.00)
}

func TestSaturation(t *testing.T) {
	var fp fpu.FPU

	var r uint64

	// unsigned saturation
	r, _ = fp.UnsignedSatQ(0, 32)
	test.ExpectEquality(t, r, 0)

	r, _ = fp.UnsignedSatQ(-1000, 32)
	test.ExpectEquality(t, r, 0)

	r, _ = fp.UnsignedSatQ(1000, 32)
	test.ExpectEquality(t, r, 1000)

	r, _ = fp.UnsignedSatQ(-4294967295, 32)
	test.ExpectEquality(t, r, 0)

	r, _ = fp.UnsignedSatQ(-4294967295-1000, 32)
	test.ExpectEquality(t, r, 0)

	r, _ = fp.UnsignedSatQ(4294967295, 32)
	test.ExpectEquality(t, r, 4294967295)

	r, _ = fp.UnsignedSatQ(4294967295+1000, 32)
	test.ExpectEquality(t, r, 4294967295)

	// signed saturation
	r, _ = fp.SignedSatQ(0, 32)
	test.ExpectEquality(t, r, 0)

	r, _ = fp.SignedSatQ(-1000, 32)
	test.ExpectEquality(t, r, 0xfffffc18)

	r, _ = fp.SignedSatQ(4294967295, 32)
	test.ExpectEquality(t, r, 2147483647)

	r, _ = fp.SignedSatQ(4294967295+1000, 32)
	test.ExpectEquality(t, r, 2147483647)

	r, _ = fp.SignedSatQ(-4294967295, 32)
	test.ExpectEquality(t, r, 0x80000000)
}

func TestFPSCRStatus(t *testing.T) {
	var fp fpu.FPU
	fp.Status.SetNZCV(0)
	test.ExpectEquality(t, fp.Status.String(), "nzcv")
	fp.Status.SetN(true)
	test.ExpectEquality(t, fp.Status.String(), "Nzcv")
	fp.Status.SetN(false)
	test.ExpectEquality(t, fp.Status.String(), "nzcv")
	fp.Status.SetZ(true)
	test.ExpectEquality(t, fp.Status.String(), "nZcv")
	fp.Status.SetZ(false)
	test.ExpectEquality(t, fp.Status.String(), "nzcv")
	fp.Status.SetC(true)
	test.ExpectEquality(t, fp.Status.String(), "nzCv")
	fp.Status.SetC(false)
	test.ExpectEquality(t, fp.Status.String(), "nzcv")
	fp.Status.SetV(true)
	test.ExpectEquality(t, fp.Status.String(), "nzcV")
	fp.Status.SetV(false)
	test.ExpectEquality(t, fp.Status.String(), "nzcv")
}

func TestComparison(t *testing.T) {
	var fp fpu.FPU
	var v float64
	var w float64
	var c uint64
	var d uint64

	v = 1
	w = -1

	for _, N := range []int{64, 32} {
		c = fp.FPRound(v, N, fp.Status)
		d = fp.FPRound(w, N, fp.Status)

		// "Table A2-4 FP comparison flag values" of "ARMv7-M"

		// equality
		fp.Status.SetNZCV(0)
		fp.FPCompare(c, c, N, false, true)
		test.ExpectEquality(t, fp.Status.String(), "nZCv")

		// greater than
		fp.Status.SetNZCV(0)
		fp.FPCompare(c, d, N, false, true)
		test.ExpectEquality(t, fp.Status.String(), "nzCv")

		// less than
		fp.Status.SetNZCV(0)
		fp.FPCompare(d, c, N, false, true)
		test.ExpectEquality(t, fp.Status.String(), "Nzcv")
	}
}
