package fpu_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/fpu"
	"github.com/jetsetilly/gopher2600/test"
)

const (
	iterations = 10000
	integerMax = 15000000
)

func TestArithmetic_random(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()

	f := func(t *testing.T) {
		var v, w float64
		var c, d uint64
		v = rand.Float64() + float64(rand.Uint64N(integerMax))
		c = fp.FPRound(v, 64, fpscr)
		w = rand.Float64() + float64(rand.Uint64N(integerMax))
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
		r = fp.FPRound(2.5, 32, fpscr)
		s = fp.FPRound(-3.1, 32, fpscr)
		q = fp.FPRound(100, 32, fpscr)
		q = fp.FPMulAdd(q, r, s, 32, false)
		_, _, f := fp.FPUnpack(q, 32, fpscr)
		test.ExpectEquality(t, f, (2.5*-3.1)+100)
	}

	fpscr.SetRMode(fpu.FPRoundNearest)
	for range iterations {
		t.Run("arithmetic (round nearest)", f)
	}

	fpscr.SetRMode(fpu.FPRoundZero)
	for range iterations {
		t.Run("arithmetic (round zero)", f)
	}

	fpscr.SetRMode(fpu.FPRoundNegInf)
	for range iterations {
		t.Run("arithmetic (round negative infinity)", f)
	}

	fpscr.SetRMode(fpu.FPRoundPlusInf)
	for range iterations {
		t.Run("arithmetic (round plus infinity)", f)
	}
}

func TestNegation_random(t *testing.T) {
	var fp fpu.FPU

	f32 := func(t *testing.T) {
		var v float64
		var c uint32
		var d uint32

		v = rand.Float64() + float64(rand.Uint64N(integerMax))
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

	f64 := func(t *testing.T) {
		var v float64
		var c uint64
		var d uint64

		v = rand.Float64() + float64(rand.Uint64N(integerMax))
		c = math.Float64bits(v)
		d = math.Float64bits(-v)

		// the two values should be unequal at this point
		test.ExpectInequality(t, c, d)

		// negate one of the values. the two value will now be equal
		d = fp.FPNeg(d, 64)
		test.ExpectEquality(t, c, d)

		// negate again to make the values unequal
		d = fp.FPNeg(d, 64)
		test.ExpectInequality(t, c, d)

		// and again to make them equal again
		d = fp.FPNeg(d, 64)
		test.ExpectEquality(t, c, d)
	}

	for range iterations {
		t.Run("negation", f32)
		t.Run("negation", f64)
	}
}

func TestAbsolute_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v float64
		var c uint32
		var d uint32

		v = rand.Float64() + float64(rand.Uint64N(integerMax))
		c = math.Float32bits(float32(v))
		d = math.Float32bits(float32(-v))

		// the two values should be unequal at this point
		test.ExpectInequality(t, c, d)

		var r uint32

		// force the negative value to be positive
		r = uint32(fp.FPAbs(uint64(d), 32))
		test.ExpectEquality(t, r, c)

		// forcing a positive value has no effect
		r = uint32(fp.FPAbs(uint64(c), 32))
		test.ExpectEquality(t, r, c)
	}

	for range iterations {
		t.Run("absolute", f)
	}
}

func TestRound_random(t *testing.T) {
	var fp fpu.FPU

	fpscr := fp.StandardFPSCRValue()

	f := func(t *testing.T) {
		var v float64
		var b uint64
		var c uint32
		v = rand.Float64() + float64(rand.Uint64N(integerMax))
		b = fp.FPRound(v, 32, fpscr)
		c = math.Float32bits(float32(v))
		test.ExpectEquality(t, uint32(b), c)
	}

	// using the math package as our baseline means we can only really test
	// using the round nearest method
	fpscr.SetRMode(fpu.FPRoundNearest)
	for range iterations {
		t.Run("rounding (round nearest)", f)
	}
}

func TestFixedToFP_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v uint64
		var c uint64

		v = rand.Uint64N(integerMax)

		// fp.FixedToFP is called with nearest set to false and FCPCR controlled
		// so we can set the rounding method explicitely ourselves (see for
		// loops below)

		// 32 bit (signed)
		c = fp.FixedToFP(v, 32, 0, false, false, true)
		test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

		// 32 bit (unsigned)
		c = fp.FixedToFP(v, 32, 0, true, false, true)
		test.ExpectEquality(t, c, uint64(math.Float32bits(float32(v))))

		// 64 bit (signed)
		c = fp.FixedToFP(v, 64, 0, false, false, true)
		test.ExpectEquality(t, c, math.Float64bits(float64(v)))

		// 64 bit (unsigned)
		c = fp.FixedToFP(v, 64, 0, true, false, true)
		test.ExpectEquality(t, c, math.Float64bits(float64(v)))
	}

	fp.Status.SetRMode(fpu.FPRoundNearest)
	for range iterations {
		t.Run("fixed to fp (round nearest)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundZero)
	for range iterations {
		t.Run("fixed to fp (round zero)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundNegInf)
	for range iterations {
		t.Run("fixed to fp (round negative infinity)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundPlusInf)
	for range iterations {
		t.Run("fixed to fp (round plus infinity)", f)
	}
}

func TestFPToFixed_random(t *testing.T) {
	var fp fpu.FPU

	f := func(t *testing.T) {
		var v uint64
		var c uint64
		var d uint64

		v = rand.Uint64N(integerMax)

		// 32 bit (signed)
		c = fp.FixedToFP(v, 32, 0, false, false, true)
		d = fp.FPToFixed(c, 32, 0, false, false, true)
		test.ExpectEquality(t, d, v)

		// 32 bit (unsigned)
		c = fp.FixedToFP(v, 32, 0, true, false, true)
		d = fp.FPToFixed(c, 32, 0, true, false, true)
		test.ExpectEquality(t, d, v)

		// 64 bit (signed)
		c = fp.FixedToFP(v, 64, 0, false, false, true)
		d = fp.FPToFixed(c, 64, 0, false, false, true)
		test.ExpectEquality(t, d, v)

		// 64 bit (unsigned)
		c = fp.FixedToFP(v, 64, 0, true, false, true)
		d = fp.FPToFixed(c, 64, 0, true, false, true)
		test.ExpectEquality(t, d, v)
	}

	fp.Status.SetRMode(fpu.FPRoundNearest)
	for range iterations {
		t.Run("fp to fixed (round nearest)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundZero)
	for range iterations {
		t.Run("fp to fixed (round zero)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundNegInf)
	for range iterations {
		t.Run("fp to fixed (round negative infinity)", f)
	}

	fp.Status.SetRMode(fpu.FPRoundPlusInf)
	for range iterations {
		t.Run("fp to fixed (round plus infinity)", f)
	}
}

func TestComparison_random(t *testing.T) {
	var fp fpu.FPU
	var v float64
	var w float64
	var c uint64
	var d uint64

	fpscr := fp.StandardFPSCRValue()
	fpscr.SetRMode(fpu.FPRoundNearest)

	f := func(t *testing.T) {
		v = rand.Float64() + float64(rand.Uint64N(integerMax))
		w = rand.Float64() + float64(rand.Uint64N(integerMax))

		// convert to uint64 depending on order of v and w
		if v > w {
			c = fp.FPRound(v, 64, fpscr)
			d = fp.FPRound(w, 64, fpscr)
		} else {
			d = fp.FPRound(v, 64, fpscr)
			c = fp.FPRound(w, 64, fpscr)
		}

		// "Table A2-4 FP comparison flag values" of "ARMv7-M"

		// equality
		fp.Status.SetNZCV(0)
		fp.FPCompare(c, c, 64, false, true)
		test.ExpectEquality(t, fp.Status.String(), "nZCv")

		// greater than
		fp.Status.SetNZCV(0)
		fp.FPCompare(c, d, 64, false, true)
		test.ExpectEquality(t, fp.Status.String(), "nzCv")

		// less than
		fp.Status.SetNZCV(0)
		fp.FPCompare(d, c, 64, false, true)
		test.ExpectEquality(t, fp.Status.String(), "Nzcv")
	}

	for range iterations {
		t.Run("comparison", f)
	}
}
