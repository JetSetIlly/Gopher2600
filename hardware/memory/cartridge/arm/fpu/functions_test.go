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
	var fpu fpu.FPU
	var v32 uint32

	v32 = uint32(fpu.FPZero(false, 32))
	test.ExpectEquality(t, v32, 0b00000000000000000000000000000000)
	v32 = uint32(fpu.FPZero(true, 32))
	test.ExpectEquality(t, v32, 0b10000000000000000000000000000000)

	v32 = uint32(fpu.FPInfinity(false, 32))
	test.ExpectEquality(t, v32, 0b01111111100000000000000000000000)
	v32 = uint32(fpu.FPInfinity(true, 32))
	test.ExpectEquality(t, v32, 0b11111111100000000000000000000000)

	v32 = uint32(fpu.FPMaxNormal(false, 32))
	test.ExpectEquality(t, v32, 0b01111111011111111111111111111111)
	v32 = uint32(fpu.FPMaxNormal(true, 32))
	test.ExpectEquality(t, v32, 0b11111111011111111111111111111111)

	v32 = uint32(fpu.FPDefaultNaN(32))
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
