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
// Appendix A of the following URL was used as a reference:
//
// http://www.6502.org/tutorials/decimal_mode.html
//
// Also, the paper by Jorge Cwik was useful:
//
// https://forums.atariage.com/topic/163876-flags-on-decimal-mode-on-the-nmos-6502

func (r *Data) AddDecimal(val uint8, carry bool) (bool, bool, bool, bool) {
	// for BCD addition the zero flag is set as though it was a binary subtraction
	br := *r
	_, _ = br.Add(val, carry)
	rzero := br.IsZero()

	// for the other flags they are set according to the rules of Seq.1 and Seq.2 (Appendix A of 6502.org)

	// Seq.1

	//	1a. AL = (A & $0F) + (B & $0F) + C
	al := (r.value & 0x0f) + (val & 0x0f)
	if carry {
		al++
	}

	// 1b. If AL >= $0A, then AL = ((AL + $06) & $0F) + $10
	if al >= 0x0a {
		al = ((al + 0x06) & 0x0f) + 0x10
	}

	// 1c. A = (A & $F0) + (B & $F0) + AL
	a1 := (uint16(r.value) & 0xf0) + (uint16(val) & 0xf0) + uint16(al)

	// 1d. Note that A can be >= $100 at this point
	// 1e. If (A >= $A0), then A = A + $60
	if a1 >= 0xa0 {
		a1 += 0x60
	}

	// 1f. The accumulator result is the lower 8 bits of A
	// 1g. The carry result is 1 if A >= $100, and is 0 if A < $100
	rcarry := a1 >= 0x100

	// Seq. 2

	// 2a. AL = (A & $0F) + (B & $0F) + C
	// 2b. If AL >= $0A, then AL = ((AL + $06) & $0F) + $10
	// (AL has already been calculated and adjusted for in Seq.1)

	// 2c. A = (A & $F0) + (B & $F0) + AL, using signed (twos complement) arithmetic
	a2 := int16(r.value&0xf0) + int16(val&0xf0) + int16(al)

	// 2e. The N flag result is 1 if bit 7 of A is 1, and is 0 if bit 7 if A is 0
	rsign := a2&0x80 == 0x80

	// 2f. The V flag result is 1 if A < -128 or A > 127, and is 0 if -128 <= A <= 127
	//
	// however, this isn't actually how the NMOS 6502 works. instead the overflow flag behaves
	// more like how the overflow flag is set for binary addition
	roverflow := ((r.value ^ uint8(a2)) & (val ^ uint8(a2)) & 0x80) != 0
	//
	// or alternatively, the following expression is equivalent
	// roverflow := (^(r.value ^ val) & (r.value ^ uint8(a2)) & 0x80) != 0

	// store result in register (using a1 from Seq.2)
	r.value = uint8(a1)

	return rcarry, rzero, roverflow, rsign
}

func (r *Data) SubtractDecimal(val uint8, carry bool) (bool, bool, bool, bool) {
	// for BCD subtraction the flags are set as though it was a binary subtraction
	br := *r
	rcarry, roverflow := br.Subtract(val, carry)
	rzero := br.IsZero()
	rsign := br.IsNegative()

	// the final value however is set according the rules of Seq.3 (Appendix A of 6502.org)

	// Seq.3

	// 3a. AL = (A & $0F) - (B & $0F) + C-1
	al := (int16(r.value) & 0x0f) - (int16(val) & 0x0f) - 1
	if carry {
		al++
	}

	// 3b. If AL < 0, then AL = ((AL - $06) & $0F) - $10
	if al < 0x00 {
		al = ((al - 0x06) & 0x0f) - 0x10
	}

	// 3c. A = (A & $F0) - (B & $F0) + AL
	a := (int16(r.value) & 0xf0) - (int16(val) & 0xf0) + al

	// 3d. If A < 0, then A = A - $60
	if a < 0x00 {
		a -= 0x60
	}

	// 3e. The accumulator result is the lower 8 bits of A
	r.value = uint8(a)

	return rcarry, rzero, roverflow, rsign
}
