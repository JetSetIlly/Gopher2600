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

// see thumb2.go for documentation information

package arm

func ThumbExpandImm_C(imm12 uint32, carry bool) (uint32, bool) {
	// "A5.3.2 Modified immediate constants in Thumb instructions" of "ARMv7-M"
	//
	// (bits(32), bit) ThumbExpandImm_C(bits(12) imm12, bit carry_in)
	//		if imm12<11:10> == ‘00’ then
	//				case imm12<9:8> of
	//				when ‘00’
	//						imm32 = ZeroExtend(imm12<7:0>, 32);
	//				when ‘01’
	//						if imm12<7:0> == ‘00000000’ then UNPREDICTABLE;
	//								imm32 = ‘00000000’ : imm12<7:0> : ‘00000000’ : imm12<7:0>;
	//				when ‘10’
	//						if imm12<7:0> == ‘00000000’ then UNPREDICTABLE;
	//								imm32 = imm12<7:0> : ‘00000000’ : imm12<7:0> : ‘00000000’;
	//				when ‘11’
	//						if imm12<7:0> == ‘00000000’ then UNPREDICTABLE;
	//								imm32 = imm12<7:0> : imm12<7:0> : imm12<7:0> : imm12<7:0>;
	//				carry_out = carry_in;
	//		else
	//				unrotated_value = ZeroExtend(‘1’:imm12<6:0>, 32);
	//				(imm32, carry_out) = ROR_C(unrotated_value, UInt(imm12<11:7>));
	//
	//		return (imm32, carry_out);

	if imm12&0xc00 == 0x00 {
		switch (imm12 & 0x300) >> 8 {
		case 0b00:
			return imm12 & 0xff, carry
		case 0b01:
			if imm12&0xff == 0x00 {
				panic("unpredicatable zero expansion")
			}
			return ((imm12 & 0xff) << 16) | (imm12 & 0xff), carry
		case 0b10:
			if imm12&0xff == 0x00 {
				panic("unpredicatable zero expansion")
			}
			return ((imm12 & 0xff) << 24) | ((imm12 & 0xff) << 8), carry
		case 0b11:
			if imm12&0xff == 0x00 {
				panic("unpredicatable zero expansion")
			}
			return ((imm12 & 0xff) << 24) | ((imm12 & 0xff) << 16) | ((imm12 & 0xff) << 8) | (imm12 & 0xff), carry
		}
	}

	unrotatedValue := (0x01 << 7) | (imm12 & 0x7f)
	return ROR_C(unrotatedValue, (imm12&0xf80)>>7)
}

func ROR_C(imm32 uint32, shift uint32) (uint32, bool) {
	// Page A2-27 or "ARMv7-M"
	//
	// (bits(N), bit) ROR_C(bits(N) x, integer shift)
	//		assert shift != 0;
	//		m = shift MOD N;
	//		result = LSR(x,m) OR LSL(x,N-m);
	//		carry_out = result<N-1>;
	//		return (result, carry_out);

	// this is specifically a 32 bit function so N is 32

	m := shift % 32
	result := (imm32 >> m) | (imm32 << (32 - m))
	return result, result&0x80000000 == 0x80000000
}

// returns result, carry, overflow
func AddWithCarry(a uint32, b uint32, c uint32) (uint32, bool, bool) {
	usum := uint64(a) + uint64(b) + uint64(c)
	ssum := int64(a) + int64(b) + int64(c)
	result := uint32(usum)
	carry := uint64(result) != usum
	overflow := int64(result) != ssum
	return result, carry, overflow
}
