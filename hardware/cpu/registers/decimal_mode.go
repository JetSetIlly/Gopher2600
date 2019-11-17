package registers

// !!TODO: handle all the nuances of 6507 decimal mode
//  . invalid BCD values (ie. nibble values A to F) correctly
//  . probably others

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
func (r *Register) AddDecimal(val uint8, carry bool) (rcarry bool) {
	runits := uint8(r.value) & 0x0f
	rtens := (uint8(r.value) & 0xf0) >> 4

	vunits := uint8(val) & 0x0f
	vtens := (uint8(val) & 0xf0) >> 4

	runits, rcarry = addDecimal(runits, vunits, carry)
	rtens, rcarry = addDecimal(rtens, vtens, rcarry)

	r.value = (rtens << 4) | runits

	return rcarry
}

func subtractDecimal(a, b uint8, carry bool) (r uint8, rcarry bool) {
	rcarry = b > a || carry && b == a

	r = a - b
	if carry {
		r--
	}

	if rcarry {
		r += 10
	}

	return r, rcarry
}

// SubtractDecimal subtracts value to from as though both registers are decimal
// representations. Returns new carry state
func (r *Register) SubtractDecimal(val uint8, carry bool) (rcarry bool) {
	runits := r.value & 0x0f
	rtens := (r.value & 0xf0) >> 4

	vunits := val & 0x0f
	vtens := (val & 0xf0) >> 4

	// invert carry flag - the 6507 uses the carry flag opposite to what you
	// might expect when subtracting
	runits, rcarry = subtractDecimal(runits, vunits, !carry)

	// rcarry from previous call to subtractDecimal() is correct
	rtens, rcarry = subtractDecimal(rtens, vtens, rcarry)

	r.value = (rtens << 4) | runits

	return !rcarry
}
