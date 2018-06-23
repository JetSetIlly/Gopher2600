package register

import "fmt"

func addDecimal(a, b uint8, carry bool) (r uint8, rcarry bool) {
	r = a + b
	if carry {
		r++
	}
	if r >= 10 {
		r -= 10
		rcarry = true
	}
	return r, rcarry
}

// AddDecimal adds value to register as though both registers are decimal
// representations. Returns new carry state
func (r *Register) AddDecimal(v interface{}, carry bool) (rcarry bool) {
	if r.size != 8 {
		panic(fmt.Sprintf("decimal mode addition only supported for uint8 values with 8 bit registers"))
	}

	val, ok := v.(uint8)
	if !ok {
		panic(fmt.Sprintf("decimal mode addition only supported for uint8 values with 8 bit registers"))
	}

	// no need to do anything if operand is zero
	if val == 0 {
		return carry
	}

	runits := uint8(r.value) & 0x0f
	rtens := (uint8(r.value) & 0xf0) >> 4

	vunits := uint8(val) & 0x0f
	vtens := (uint8(val) & 0xf0) >> 4

	runits, rcarry = addDecimal(runits, vunits, carry)
	rtens, rcarry = addDecimal(rtens, vtens, rcarry)

	r.value = uint32((rtens << 4) | runits)

	return rcarry
}

func subtractDecimal(a, b int, carry bool) (r int, rcarry bool) {
	r = a - b
	if carry {
		r--
	}
	if r < 0 {
		r += 10
		rcarry = true
	}
	return r, rcarry
}

// SubtractDecimal subtracts value to from as though both registers are decimal
// representations. Returns new carry state
func (r *Register) SubtractDecimal(v interface{}, carry bool) (rcarry bool) {
	if r.size != 8 {
		panic(fmt.Sprintf("decimal mode subtraction only supported for uint8 values with 8 bit registers"))
	}

	val, ok := v.(uint8)
	if !ok {
		panic(fmt.Sprintf("decimal mode subtraction only supported for uint8 values with 8 bit registers"))
	}

	// no need to do anything if operand is zero
	if val == 0 {
		return carry
	}

	runits := int(r.value) & 0x0f
	rtens := (int(r.value) & 0xf0) >> 4

	vunits := int(val) & 0x0f
	vtens := (int(val) & 0xf0) >> 4

	// invert carry flag - the 6502 uses the carry flag opposite to what you
	// might expect when subtracting
	runits, rcarry = subtractDecimal(runits, vunits, !carry)

	// rcarry from previous call to subtractDecimal() is correct
	rtens, rcarry = subtractDecimal(rtens, vtens, rcarry)

	r.value = uint32((rtens << 4) | runits)

	return !rcarry
}
