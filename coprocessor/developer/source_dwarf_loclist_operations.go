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

package developer

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/leb128"
)

// decode loclist DWARF operation but adjust decoding addresses with an origin value.
// there's only one operator (DW_OP_addr) that needs this special handling and
// only then when the expression appears outside of a location list
func (sec *loclistSection) decodeLoclistOperationWithOrigin(expr []uint8, origin uint64) (loclistOperator, int) {
	switch expr[0] {
	case 0x03:
		// DW_OP_addr
		// (literal encoding)
		// "The DW_OP_addr operation has a single operand that encodes a machine address and whose
		// size is the size of an address on the target machine."
		address := uint64(expr[1])
		address |= uint64(expr[2]) << 8
		address |= uint64(expr[3]) << 16
		address |= uint64(expr[4]) << 24
		address += origin
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(address),
				valueOk:  false,
				operator: "DW_OP_addr",
			}, nil
		}, 5
	}

	// other operators do not need the special handling
	return sec.decodeLoclistOperation(expr)
}

// decode loclist DWARF expression operation. the expr argument is the operation
// stream. the first entry in the slice is the operator, remaining entries in
// the slice contain the operands for the operator. entries in the slice may be
// unused.
//
// the simpleLocDesc argument indicates that the operator is expected to be
// used in a context of being a single location description. the function will
// resolve the stack as appropriate if this argument is true.
//
// the function returns a resolver function and the number of bytes consumed in
// the expr slice
//
// returns nil, zero, if expression cannot be handled.
func (sec *loclistSection) decodeLoclistOperation(expr []uint8) (loclistOperator, int) {
	// expression location operators reference
	//
	// "DWARF Debugging Information Format Version 4", page 17 to 24
	//
	// also the table of values on page 153, "section 7.7.1 DWARF Expressions"

	switch expr[0] {
	case 0x03:
		// DW_OP_addr
		// (literal encoding)
		// "The DW_OP_addr operation has a single operand that encodes a machine address and whose
		// size is the size of an address on the target machine."
		address := uint64(expr[1])
		address |= uint64(expr[2]) << 8
		address |= uint64(expr[3]) << 16
		address |= uint64(expr[4]) << 24
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(address),
				valueOk:  false,
				operator: "DW_OP_addr",
			}, nil
		}, 5

	case 0x06:
		// DW_OP_deref
		// (stack operations)
		// "The DW_OP_deref operation pops the top stack entry and treats it as an address. The
		// value retrieved from that address is pushed. The size of the data retrieved from the
		// dereferenced address is the size of an address on the target machine"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			address := uint64(a.value)
			value, ok := sec.coproc.CoProcRead32bit(uint32(address))
			return location{
				address:   address,
				addressOk: true,
				value:     value,
				valueOk:   ok,
				operator:  "DW_OP_deref",
			}, nil
		}, 1

	case 0x08:
		// DW_OP_const1u
		// (literal encoding)
		// "The single operand of a DW_OP_constnu operation provides a 1, 2, 4, or 8-byte unsigned
		// integer constant, respectively"
		cons := uint64(expr[1])
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const1u",
			}, nil
		}, 2

	case 0x09:
		// DW_OP_const1s
		// (literal encoding)
		// "The single operand of a DW_OP_constns operation provides a 1, 2, 4, or 8-byte signed
		// integer constant, respectively"
		cons := uint64(expr[1])
		if cons&0x80 == 0x80 {
			cons |= 0xffffffffffffff00
		}
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const1s",
			}, nil
		}, 2

	case 0x0a:
		// DW_OP_const2u
		// (literal encoding)
		cons := uint64(expr[1])
		cons |= uint64(expr[2]) << 8
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const2u",
			}, nil
		}, 3

	case 0x0b:
		// DW_OP_const2s
		// (literal encoding)
		cons := uint64(expr[1])
		cons |= uint64(expr[2]) << 8
		if cons&0x8000 == 0x8000 {
			cons |= 0xffffffffffff0000
		}
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const2s",
			}, nil
		}, 3

	case 0x0c:
		// DW_OP_const4u
		// (literal encoding)
		cons := uint64(expr[1])
		cons |= uint64(expr[2]) << 8
		cons |= uint64(expr[3]) << 16
		cons |= uint64(expr[4]) << 24
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const4u",
			}, nil
		}, 5

	case 0x0d:
		// DW_OP_const4s
		// (literal encoding)
		cons := uint64(expr[1])
		cons |= uint64(expr[2]) << 8
		cons |= uint64(expr[3]) << 16
		cons |= uint64(expr[4]) << 24
		if cons&0x80000000 == 0x80000000 {
			cons |= 0xffffffff00000000
		}
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(cons),
				valueOk:  true,
				operator: "DW_OP_const4s",
			}, nil
		}, 5

	case 0x10:
		// DW_OP_constu
		// (literal encoding)
		// "The single operand of the DW_OP_constu operation provides an unsigned LEB128 integer
		// constant"
		value, n := leb128.DecodeULEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(value),
				valueOk:  true,
				operator: "DW_OP_constu",
			}, nil
		}, n + 1

	case 0x11:
		// DW_OP_consts
		// (literal encoding)
		// "The single operand of the DW_OP_constu operation provides an signed LEB128 integer
		// constant"
		value, n := leb128.DecodeSLEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(value),
				valueOk:  true,
				operator: "DW_OP_consts",
			}, nil
		}, n + 1

	case 0x12:
		// DW_OP_dup
		// (stack operations)
		// "The DW_OP_dup operation duplicates the value at the top of the stack"
		return func(loc *loclist) (location, error) {
			return loc.peek(), nil
		}, 1

	case 0x13:
		fallthrough
	case 0x14:
		fallthrough
	case 0x15:
		fallthrough
	case 0x16:
		fallthrough
	case 0x17:
		return nil, 0

	case 0x18:
		// DW_OP_xderef
		// (stack operations)
		// "The DW_OP_xderef operation provides an extended dereference mechanism. The entry at the
		// top of the stack is treated as an address. The second stack entry is treated as an
		// “address space identifier” for those architectures that support multiple address spaces.
		// The top two stack elements are popped, and a data item is retrieved through an
		// implementation-defined address calculation and pushed as the new stack top. The size of
		// the data retrieved from the dereferenced address is the size of an address on the target
		// machine"
		return nil, 0

	case 0x19:
		// DW_OP_abs
		// (arithmetic and logic operations)
		// "The DW_OP_abs operation pops the top stack entry, interprets it as a signed value and
		// pushes its absolute value. If the absolute value cannot be represented, the result is
		// undefined"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			value := a.value & 0x7fffffff
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_abs",
			}, nil
		}, 1

	case 0x1a:
		// DW_OP_and
		// (arithmetic and logic operations)
		// "The DW_OP_and operation pops the top two stack values, performs a bitwise and operation"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value & a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_and",
			}, nil
		}, 1

	case 0x1b:
		// DW_OP_div
		// (arithmetic and logic operations)
		// "The DW_OP_div operation pops the top two stack values, divides the former second entry
		// by the former top of the stack using signed division, and pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value / a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_div",
			}, nil
		}, 1

	case 0x1c:
		// DW_OP_minus
		// (arithmetic and logic operations)
		// "The DW_OP_minus operation pops the top two stack values, subtracts the former top of the
		// stack from the former second entry, and pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value - a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_minus",
			}, nil
		}, 1

	case 0x1d:
		// DW_OP_mod
		// (arithmetic and logic operations)
		// "The DW_OP_mod operation pops the top two stack values and pushes the result of the
		// calculation: former second stack entry modulo the former top of the stack"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value % a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_mod",
			}, nil
		}, 1

	case 0x1e:
		// DW_OP_mul
		// (arithmetic and logic operations)
		// "The DW_OP_mul operation pops the top two stack entries, multiplies them together, and
		// pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value * a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_mul",
			}, nil
		}, 1

	case 0x1f:
		// DW_OP_neg
		// (arithmetic and logic operations)
		// "The DW_OP_neg operation pops the top stack entry, interprets it as a signed value and
		// pushes its negation. If the negation cannot be represented, the result is undefined"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			value := uint32(-int32(a.value))
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_neg",
			}, nil
		}, 1

	case 0x20:
		// DW_OP_not
		// (arithmetic and logic operations)
		// "The DW_OP_not operation pops the top stack entry, and pushes its bitwise complement"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			value := ^a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_not",
			}, nil
		}, 1

	case 0x21:
		// DW_OP_or
		// (arithmetic and logic operations)
		// "The DW_OP_or operation pops the top two stack entries, performs a bitwise or operation
		// on the two, and pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value | a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_or",
			}, nil
		}, 1

	case 0x22:
		// DW_OP_plus
		// (arithmetic and logic operations)
		// "The DW_OP_plus operation pops the top two stack entries, adds them together, and pushes"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value + a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_plus",
			}, nil
		}, 1

	case 0x23:
		// DW_OP_plus_uconst
		// (arithmetic and logic operations)
		// "The DW_OP_plus_uconst operation pops the top stack entry, adds it to the unsigned LEB128
		// constant operand and pushes the result"
		value, n := leb128.DecodeULEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			return location{
				value:    uint32(value) + a.value,
				valueOk:  true,
				operator: "DW_OP_plus_uconst",
			}, nil
		}, n + 1

	case 0x24:
		// DW_OP_shl
		// (arithmetic and logic operations)
		// "The DW_OP_shl operation pops the top two stack entries, shifts the former second entry
		// left"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value << a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_shl",
			}, nil
		}, 1

	case 0x25:
		// DW_OP_shr
		// (arithmetic and logic operations)
		// "The DW_OP_shr operation pops the top two stack entries, shifts the former second entry
		// right logically (filling with zero bits) by the number of bits specified by the former
		// top of the stack, and pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value >> a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_shr",
			}, nil
		}, 1

	case 0x26:
		// DW_OP_shra
		// (arithmetic and logic operations)
		// "The DW_OP_shra operation pops the top two stack entries, shifts the former second entry
		// right arithmetically (divide the magnitude by 2, keep the same sign for the result) by
		// the number of bits specified by the former top of the stack, and pushes the result"
		// "DWARF4 Standard"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			signExtend := (b.value & 0x80000000) >> 31
			value := b.value >> a.value
			if signExtend == 0x01 {
				value |= ^uint32(0) << (32 - a.value)
			}
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_shra",
			}, nil
		}, 1

	case 0x27:
		// DW_OP_xor
		// (arithmetic and logic operations)
		// "The DW_OP_xor operation pops the top two stack entries, performs a bitwise exclusive-or
		// operation on the two, and pushes the result"
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			b, _ := loc.pop()
			value := b.value ^ a.value
			return location{
				value:    value,
				valueOk:  true,
				operator: "DW_OP_xor",
			}, nil
		}, 1

	case 0x28:
		// DW_OP_bra
		return nil, 0
	case 0x29:
		// DW_OP_eq
		return nil, 0
	case 0x2a:
		// DW_OP_ge
		return nil, 0
	case 0x2b:
		// DW_OP_gt
		return nil, 0
	case 0x2c:
		// DW_OP_le
		return nil, 0
	case 0x2d:
		// DW_OP_lt
		return nil, 0
	case 0x2e:
		// DW_OP_ne
		return nil, 0
	case 0x2f:
		// DW_OP_skip
		return nil, 0

	case 0x30:
		fallthrough
	case 0x31:
		fallthrough
	case 0x32:
		fallthrough
	case 0x33:
		fallthrough
	case 0x34:
		fallthrough
	case 0x35:
		fallthrough
	case 0x36:
		fallthrough
	case 0x37:
		fallthrough
	case 0x38:
		fallthrough
	case 0x39:
		fallthrough
	case 0x3a:
		fallthrough
	case 0x3b:
		fallthrough
	case 0x3c:
		fallthrough
	case 0x3d:
		fallthrough
	case 0x3e:
		fallthrough
	case 0x3f:
		fallthrough
	case 0x40:
		fallthrough
	case 0x41:
		fallthrough
	case 0x42:
		fallthrough
	case 0x43:
		fallthrough
	case 0x44:
		fallthrough
	case 0x45:
		fallthrough
	case 0x46:
		fallthrough
	case 0x47:
		fallthrough
	case 0x48:
		fallthrough
	case 0x49:
		fallthrough
	case 0x4a:
		fallthrough
	case 0x4b:
		fallthrough
	case 0x4c:
		fallthrough
	case 0x4d:
		fallthrough
	case 0x4e:
		fallthrough
	case 0x4f:
		// DW_OP_lit0, DW_OP_lit1, ..., DW_OP_lit31
		// (literal encoding)
		// "The DW_OP_litn operations encode the unsigned literal values from 0 through 31,
		// inclusive"
		lit := expr[0] - 0x30
		return func(loc *loclist) (location, error) {
			return location{
				value:    uint32(lit),
				valueOk:  true,
				operator: fmt.Sprintf("DW_OP_lit%d", lit),
			}, nil
		}, 1

	case 0x50:
		fallthrough
	case 0x51:
		fallthrough
	case 0x52:
		fallthrough
	case 0x53:
		fallthrough
	case 0x54:
		fallthrough
	case 0x55:
		fallthrough
	case 0x56:
		fallthrough
	case 0x57:
		fallthrough
	case 0x58:
		fallthrough
	case 0x59:
		fallthrough
	case 0x5a:
		fallthrough
	case 0x5b:
		fallthrough
	case 0x5c:
		fallthrough
	case 0x5d:
		fallthrough
	case 0x5e:
		fallthrough
	case 0x5f:
		fallthrough
	case 0x60:
		fallthrough
	case 0x61:
		fallthrough
	case 0x62:
		fallthrough
	case 0x63:
		fallthrough
	case 0x64:
		fallthrough
	case 0x65:
		fallthrough
	case 0x66:
		fallthrough
	case 0x67:
		fallthrough
	case 0x68:
		fallthrough
	case 0x69:
		fallthrough
	case 0x6a:
		fallthrough
	case 0x6b:
		fallthrough
	case 0x6c:
		fallthrough
	case 0x6d:
		fallthrough
	case 0x6e:
		fallthrough
	case 0x6f:
		// DW_OP_reg0, DW_OP_reg1, ..., DW_OP_reg31
		// (register location description)
		// "The DW_OP_regn operations encode the names of up to 32 registers, numbered from 0
		// through 31, inclusive. The object addressed is in register n"
		reg := expr[0] - 0x50
		return func(loc *loclist) (location, error) {
			value, ok := sec.coproc.CoProcRegister(int(reg))
			if !ok {
				return location{}, fmt.Errorf("unknown register: %d", reg)
			}
			return location{
				value:    value,
				valueOk:  true,
				operator: fmt.Sprintf("DW_OP_reg%d", reg),
			}, nil
		}, 1

	case 0x70:
		fallthrough
	case 0x71:
		fallthrough
	case 0x72:
		fallthrough
	case 0x73:
		fallthrough
	case 0x74:
		fallthrough
	case 0x75:
		fallthrough
	case 0x76:
		fallthrough
	case 0x77:
		fallthrough
	case 0x78:
		fallthrough
	case 0x79:
		fallthrough
	case 0x7a:
		fallthrough
	case 0x7b:
		fallthrough
	case 0x7c:
		fallthrough
	case 0x7d:
		fallthrough
	case 0x7e:
		fallthrough
	case 0x7f:
		fallthrough
	case 0x80:
		fallthrough
	case 0x81:
		fallthrough
	case 0x82:
		fallthrough
	case 0x83:
		fallthrough
	case 0x84:
		fallthrough
	case 0x85:
		fallthrough
	case 0x86:
		fallthrough
	case 0x87:
		fallthrough
	case 0x88:
		fallthrough
	case 0x89:
		fallthrough
	case 0x8a:
		fallthrough
	case 0x8b:
		fallthrough
	case 0x8c:
		fallthrough
	case 0x8d:
		fallthrough
	case 0x8e:
		fallthrough
	case 0x8f:
		// DW_OP_breg0, DW_OP_breg1, ..., DW_OP_breg31
		// (register based addressing)
		// "The single operand of the DW_OP_bregn operations provides a signed LEB128 offset from
		// the specified register"
		reg := expr[0] - 0x70
		offset, n := leb128.DecodeSLEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			regVal, ok := sec.coproc.CoProcRegister(int(reg))
			if !ok {
				return location{}, fmt.Errorf("unknown register: %d", reg)
			}

			// the general description for "register based addressing" says that "the following
			// operations push a value onto the stack that is the result of adding the contents of a
			// register to a given signed offset"
			address := int64(regVal) + offset

			return location{
				value:    uint32(address),
				valueOk:  false,
				operator: fmt.Sprintf("DW_OP_breg%d", reg),
			}, nil
		}, n + 1

	case 0x90:
		// DW_OP_regx
		// (register location description)
		reg, n := leb128.DecodeSLEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			value, ok := sec.coproc.CoProcRegister(int(reg))
			if !ok {
				return location{}, fmt.Errorf("unknown register: %d", reg)
			}
			return location{
				value:    value,
				valueOk:  false,
				operator: "DW_OP_regx",
			}, nil
		}, n + 1

	case 0x91:
		// DW_OP_fbreg
		// (register based addressing)
		// "The DW_OP_fbreg operation provides a signed LEB128 offset from the address specified by
		// the location description in the DW_AT_frame_base attribute of the current function. (This
		// is typically a “stack pointer” register plus or minus some offset. On more sophisticated
		// systems it might be a location list that adjusts the offset according to changes in the
		// stack pointer as the PC changes)"
		offset, n := leb128.DecodeSLEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			fb, err := loc.ctx.framebase()
			if err != nil {
				return location{}, err
			}
			address := int64(fb) + offset

			return location{
				value:    uint32(address),
				valueOk:  false,
				operator: "DW_OP_fbreg",
			}, nil
		}, n + 1

	case 0x93:
		// DW_OP_piece
		// (composite location descriptions)
		// "The DW_OP_piece operation takes a single operand, which is an unsigned LEB128 number.
		// The number describes the size in bytes of the piece of the object referenced by the preceding
		// simple location description. If the piece is located in a register, but does not occupy the entire
		// register, the placement of the piece within that register is defined by the ABI"
		size, n := leb128.DecodeULEB128(expr[1:])
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			v := a.value
			if !a.valueOk {
				var ok bool
				v, ok = loc.coproc.CoProcRead32bit(a.value)
				if !ok {
					return location{}, fmt.Errorf("unknown address: %08x", a.value)
				}
			}
			switch size {
			case 1:
				v &= 0x000000ff
			case 2:
				v &= 0x0000ffff
			case 3:
				v &= 0x00ffffff
			case 4:
				v &= 0xffffffff
			default:
				return location{}, fmt.Errorf("unknown piece size: %d", size)
			}

			return location{
				value:    v,
				valueOk:  false,
				operator: "DW_OP_piece",
			}, nil
		}, n + 1

	case 0x94:
		// DW_OP_deref_size
		// (stack operations)
		// "The DW_OP_deref_size operation behaves like the DW_OP_deref operation: it pops the top
		// stack entry and treats it as an address. The value retrieved from that address is pushed.
		// In the DW_OP_deref_size operation, however, the size in bytes of the data retrieved from
		// the dereferenced address is specified by the single operand. This operand is a 1-byte
		// unsigned integral constant whose value may not be larger than the size of an address on
		// the target machine. The data retrieved is zero extended to the size of an address on the
		// target machine before being pushed onto the expression stack."
		size := expr[1] // in bytes
		return func(loc *loclist) (location, error) {
			a, _ := loc.pop()
			address := uint64(a.value)

			value, ok := sec.coproc.CoProcRead32bit(uint32(address))
			if !ok {
				return location{}, fmt.Errorf("unknown address: %08x", address)
			}

			mask := ^((^int32(0)) << (size * 8))
			value &= uint32(mask)

			return location{
				address:   address,
				addressOk: true,
				value:     value,
				valueOk:   true,
				operator:  "DW_OP_deref_size",
			}, nil
		}, 2

	case 0x95:
		// DW_OP_xdref_size
		return nil, 0

	case 0x96:
		// DW_OP_nop
		// "The DW_OP_nop operation is a place holder. It has no effect on the location stack or any
		// of its values"
		return func(loc *loclist) (location, error) {
			return location{
				operator: "DW_OP_nop",
			}, nil
		}, 1

	case 0x9c:
		// DW_OP_call_frame_cfa
		// "The DW_OP_call_frame_cfa operation pushes the value of the CFA, obtained from the Call
		// Frame Information"
		//
		// NOTE: the context for the framebase function should hopefully point
		// to a frameSection instance
		return func(loc *loclist) (location, error) {
			fb, err := loc.ctx.framebase()
			if err != nil {
				return location{}, err
			}
			return location{
				value:    uint32(fb),
				valueOk:  true,
				operator: "DW_OP_call_frame_cfa",
			}, nil
		}, 1

	case 0x9f:
		// DW_OP_stack_value
		// (implicit location descriptions)
		// "The DW_OP_stack_value operation specifies that the object does not exist in memory but
		// its value is nonetheless known and is at the top of the DWARF expression stack. In this
		// form of location description, the DWARF expression represents the actual value of the
		// object, rather than its location. The DW_OP_stack_value operation terminates the
		// expression"
		return func(loc *loclist) (location, error) {
			res, ok := loc.pop()
			if !ok {
				return location{}, fmt.Errorf("stack empty")
			}
			res.valueOk = true
			res.operator = "DW_OP_stack_value"
			return res, nil
		}, 1
	}

	return nil, 0
}
