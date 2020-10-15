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

package registers

// these decimal functions return information about zero and sign bits in
// addition to the carry and overflow. the cpu can use these value to set the
// status flags. this is different to binary addition/subtraction which only
// returns information for the carry and overflow flags.
//
// details of this has been taken from "Flags on Decimal mode in the NMOS 6502"
// v1.0 by Jorge Cwik:
//
// https://atariage.com/forums/applications/core/interface/file/attachment.php?id=163231

func addDecimal(a, b uint8, carry bool) (r uint8, rcarry bool) {
	r = a + b
	if carry {
		r++
	}
	return r, r > 9
}

// AddDecimal adds value to register as though both registers are decimal
// representations. Returns new carry state, zero, overflow, sign bit
// information.
func (r *Register) AddDecimal(val uint8, carry bool) (bool, bool, bool, bool) {
	var zero, overflow, sign bool
	var ucarry, tcarry bool

	// binary addition of units and tens
	runits := r.value & 0x0f
	vunits := val & 0x0f
	runits, ucarry = addDecimal(runits, vunits, carry)

	rtens := (r.value & 0xf0) >> 4
	vtens := (val & 0xf0) >> 4
	rtens, tcarry = addDecimal(rtens, vtens, ucarry)

	// from the Cwik document:
	//
	// "The Z flag is computed before performing any decimal adjust."
	zero = runits == 0x00 && rtens == 0x00

	// decimal correction for units
	if ucarry {
		runits -= 10
	}

	// from the Cwik document:
	//
	// "The N and V flags are computed after a decimal adjust of the low
	// nibble, but before adjusting the high nibble."
	//
	// not forgetting that the tens value has not been shifted into the upper
	// nibble yet
	overflow = rtens&0x04 == 0x04
	sign = rtens&0x08 == 0x08

	// decimal correction for tens
	if tcarry {
		rtens -= 10
	}

	// pack units/tens nibbles into register
	r.value = (rtens << 4) | runits

	return tcarry, zero, overflow, sign
}

func subtractDecimal(a, b uint8, carry bool) (r uint8, rcarry bool) {
	r = a - b
	if carry {
		r--
	}
	return r, b > a || carry && b == a
}

// SubtractDecimal subtracts value to from as though both registers are decimal
// representations. Returns new carry state, zero, overflow, sign bit
// information.
func (r *Register) SubtractDecimal(val uint8, carry bool) (bool, bool, bool, bool) {
	var zero, overflow, sign bool
	var ucarry, tcarry bool

	// invert carry flag - the 6507 uses the carry flag opposite to what you
	// might expect when subtracting
	carry = !carry

	runits := r.value & 0x0f
	vunits := val & 0x0f
	runits, ucarry = subtractDecimal(runits, vunits, carry)

	rtens := (r.value & 0xf0) >> 4
	vtens := (val & 0xf0) >> 4
	rtens, tcarry = subtractDecimal(rtens, vtens, ucarry)

	// from the Cwik document:
	//
	// "The Z flag is computed before performing any decimal adjust."
	zero = runits == 0x00 && rtens == 0x00

	// decimal correction for units
	if ucarry {
		runits += 10
	}

	// from the Cwik document:
	//
	// "The N and V flags are computed after a decimal adjust of the low
	// nibble, but before adjusting the high nibble."
	//
	// not forgetting that the tens value has not been shifted into the upper
	// nibble yet
	overflow = rtens&0x04 == 0x04
	sign = rtens&0x08 == 0x08

	// decimal correction for tens
	if tcarry {
		rtens += 10
	}

	// pack units/tens nibbles into register
	r.value = (rtens << 4) | runits

	return !tcarry, zero, overflow, sign
}
