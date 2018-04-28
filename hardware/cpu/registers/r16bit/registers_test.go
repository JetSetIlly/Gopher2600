package r16bit_test

import (
	"headlessVCS/hardware/cpu/registers/r16bit"
	"headlessVCS/mflib"
	"testing"
)

func TestRegister(t *testing.T) {
	r16, _ := r16bit.Generate(0, 16, "test r16")
	mflib.Assert(t, r16.IsZero(), true)
	mflib.Assert(t, r16, 0)
	r16.Load(127)
	mflib.Assert(t, r16, 127)
	r16.Add(2, false)
	mflib.Assert(t, r16, 129)
	mflib.Assert(t, r16, "0000000010000001")
	r16.Load(0xffff)
	mflib.Assert(t, r16.IsNegative(), true)
	r16.Add(1, false)
	mflib.Assert(t, r16.IsZero(), true)
}
