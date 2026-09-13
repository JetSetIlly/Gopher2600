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

package dwarf

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf/leb128"
)

// decode loclist DWARF expression operation. the expr argument is the operation
// stream. the first entry in the slice is the operator, remaining entries in
// the slice contain the operands for the operator. entries in the slice may be
// unused.
//
// the function returns a resolver function and the number of bytes consumed in
// the expr slice
//
// returns empty loclistOperator and zero if expression cannot be handled
func (sec *loclistDecoder) decodeLoclistOperation(expr []uint8) (loclistOperator, error) {
	// expression location operators reference
	//
	// "DWARF Debugging Information Format Version 4", page 17 to 24
	//
	// also the table of values on pages 163-167, "section 7.7.1 DWARF Expressions"

	switch expr[0] {
	case 0x03:
		// DW_OP_addr
		// (literal encoding)
		// "The DW_OP_addr operation has a single operand that encodes a machine address and whose
		// size is the size of an address on the target machine."
		address := sec.byteOrder.Uint32(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: address,
				}, nil
			},
			size:     5,
			operator: "DW_OP_addr",
		}, nil

	case 0x06:
		// DW_OP_deref
		// (stack operations)
		// "The DW_OP_deref operation pops the top stack entry and treats it as an address. The
		// value retrieved from that address is pushed. The size of the data retrieved from the
		// dereferenced address is the size of an address on the target machine"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				value, ok := sec.coproc.Peek(a.value)
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown address: %08x", a.value)
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_deref",
		}, nil

	case 0x08:
		// DW_OP_const1u
		// (literal encoding)
		// "The single operand of a DW_OP_constnu operation provides a 1, 2, 4, or 8-byte unsigned
		// integer constant, respectively"
		cons := uint32(expr[1])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     2,
			operator: "DW_OP_const1u",
		}, nil

	case 0x09:
		// DW_OP_const1s
		// (literal encoding)
		// "The single operand of a DW_OP_constns operation provides a 1, 2, 4, or 8-byte signed
		// integer constant, respectively"
		cons := uint32(expr[1])
		if cons&0x80 == 0x80 {
			cons |= 0xffffff00
		}
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     2,
			operator: "DW_OP_const1s",
		}, nil

	case 0x0a:
		// DW_OP_const2u
		// (literal encoding)
		cons := uint32(sec.byteOrder.Uint16(expr[1:]))
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     3,
			operator: "DW_OP_const2u",
		}, nil

	case 0x0b:
		// DW_OP_const2s
		// (literal encoding)
		cons := uint32(sec.byteOrder.Uint16(expr[1:]))
		if cons&0x8000 == 0x8000 {
			cons |= 0xffff0000
		}
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     3,
			operator: "DW_OP_const2s",
		}, nil

	case 0x0c:
		// DW_OP_const4u
		// (literal encoding)
		cons := sec.byteOrder.Uint32(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     5,
			operator: "DW_OP_const4u",
		}, nil

	case 0x0d:
		// DW_OP_const4s
		// (literal encoding)
		cons := sec.byteOrder.Uint32(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: cons,
				}, nil
			},
			size:     5,
			operator: "DW_OP_const4s",
		}, nil

	case 0x10:
		// DW_OP_constu
		// (literal encoding)
		// "The single operand of the DW_OP_constu operation provides an unsigned LEB128 integer
		// constant"
		value, n := leb128.DecodeULEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: uint32(value),
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_constu",
		}, nil

	case 0x11:
		// DW_OP_consts
		// (literal encoding)
		// "The single operand of the DW_OP_constu operation provides an signed LEB128 integer
		// constant"
		value, n := leb128.DecodeSLEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: uint32(value),
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_consts",
		}, nil

	case 0x12:
		// DW_OP_dup
		// (stack operations)
		// "The DW_OP_dup operation duplicates the value at the top of the stack"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				loc.push(a)
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_dup",
		}, nil

	case 0x13:
		// DW_OP_drop
		// (stack operations)
		// "The DW_OP_drop operation pops the value at the top of the stack"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				_, _ = loc.pop()
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_drop",
		}, nil

	case 0x14:
		// DW_OP_over
		// (stack operations)
		// "The DW_OP_over operation duplicates the entry currently second in the stack at the top of
		// the stack. This is equivalent to a DW_OP_pick operation, with index 1"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				loc.push(b)
				loc.push(a)
				loc.push(b)
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_over",
		}, nil

	case 0x15:
		// DW_OP_pick
		// (stack operations)
		// "The single operand of the DW_OP_pick operation provides a 1-byte index. A copy of the
		// stack entry with the specified index (0 through 255, inclusive) is pushed onto the stack"
		n := int(expr[1]) // depth of pick
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				if a, ok := loc.peek(n); ok {
					loc.push(a)
				}
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     2,
			operator: "DW_OP_pick",
		}, nil

	case 0x16:
		// DW_OP_swap
		// (stack operations)
		// "The DW_OP_swap operation swaps the top two stack entries. The entry at the top of the
		// stack becomes the second stack entry, and the second entry becomes the top of the stack"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				loc.push(a)
				loc.push(b)
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_swap",
		}, nil

	case 0x17:
		// DW_OP_rot
		// (stack operations)
		// "The DW_OP_rot operation rotates the first three stack entries. The entry at the top of
		// the stack becomes the third stack entry, the second entry becomes the top of the stack,
		// and the third entry becomes the second entry"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				c, _ := loc.pop()
				loc.push(a)
				loc.push(c)
				loc.push(b)
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_swap",
		}, nil

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
		return loclistOperator{}, nil

	case 0x19:
		// DW_OP_abs
		// (arithmetic and logic operations)
		// "The DW_OP_abs operation pops the top stack entry, interprets it as a signed value and
		// pushes its absolute value. If the absolute value cannot be represented, the result is
		// undefined"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				value := a.value & 0x7fffffff
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_abs",
		}, nil

	case 0x1a:
		// DW_OP_and
		// (arithmetic and logic operations)
		// "The DW_OP_and operation pops the top two stack values, performs a bitwise and operation"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value & a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_and",
		}, nil

	case 0x1b:
		// DW_OP_div
		// (arithmetic and logic operations)
		// "The DW_OP_div operation pops the top two stack values, divides the former second entry
		// by the former top of the stack using signed division, and pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value / a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_div",
		}, nil

	case 0x1c:
		// DW_OP_minus
		// (arithmetic and logic operations)
		// "The DW_OP_minus operation pops the top two stack values, subtracts the former top of the
		// stack from the former second entry, and pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value - a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_minus",
		}, nil

	case 0x1d:
		// DW_OP_mod
		// (arithmetic and logic operations)
		// "The DW_OP_mod operation pops the top two stack values and pushes the result of the
		// calculation: former second stack entry modulo the former top of the stack"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value % a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_mod",
		}, nil

	case 0x1e:
		// DW_OP_mul
		// (arithmetic and logic operations)
		// "The DW_OP_mul operation pops the top two stack entries, multiplies them together, and
		// pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value * a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_mul",
		}, nil

	case 0x1f:
		// DW_OP_neg
		// (arithmetic and logic operations)
		// "The DW_OP_neg operation pops the top stack entry, interprets it as a signed value and
		// pushes its negation. If the negation cannot be represented, the result is undefined"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				value := uint32(-int32(a.value))
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_neg",
		}, nil

	case 0x20:
		// DW_OP_not
		// (arithmetic and logic operations)
		// "The DW_OP_not operation pops the top stack entry, and pushes its bitwise complement"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				value := ^a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_not",
		}, nil

	case 0x21:
		// DW_OP_or
		// (arithmetic and logic operations)
		// "The DW_OP_or operation pops the top two stack entries, performs a bitwise or operation
		// on the two, and pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value | a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_or",
		}, nil

	case 0x22:
		// DW_OP_plus
		// (arithmetic and logic operations)
		// "The DW_OP_plus operation pops the top two stack entries, adds them together, and pushes"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value + a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_plus",
		}, nil

	case 0x23:
		// DW_OP_plus_uconst
		// (arithmetic and logic operations)
		// "The DW_OP_plus_uconst operation pops the top stack entry, adds it to the unsigned LEB128
		// constant operand and pushes the result"
		value, n := leb128.DecodeULEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				return loclistStackItem{
					class: stackItemClassPush,
					value: uint32(value) + a.value,
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_plus_uconst",
		}, nil

	case 0x24:
		// DW_OP_shl
		// (arithmetic and logic operations)
		// "The DW_OP_shl operation pops the top two stack entries, shifts the former second entry
		// left"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value << a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_shl",
		}, nil

	case 0x25:
		// DW_OP_shr
		// (arithmetic and logic operations)
		// "The DW_OP_shr operation pops the top two stack entries, shifts the former second entry
		// right logically (filling with zero bits) by the number of bits specified by the former
		// top of the stack, and pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value >> a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_shr",
		}, nil

	case 0x26:
		// DW_OP_shra
		// (arithmetic and logic operations)
		// "The DW_OP_shra operation pops the top two stack entries, shifts the former second entry
		// right arithmetically (divide the magnitude by 2, keep the same sign for the result) by
		// the number of bits specified by the former top of the stack, and pushes the result"
		// "DWARF4 Standard"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				signExtend := (b.value & 0x80000000) >> 31
				value := b.value >> a.value
				if signExtend == 0x01 {
					value |= ^uint32(0) << (32 - a.value)
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_shra",
		}, nil

	case 0x27:
		// DW_OP_xor
		// (arithmetic and logic operations)
		// "The DW_OP_xor operation pops the top two stack entries, performs a bitwise exclusive-or
		// operation on the two, and pushes the result"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				value := b.value ^ a.value
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_xor",
		}, nil

	case 0x28:
		// DW_OP_bra
		// (control flow operations)
		// "DW_OP_bra is a conditional branch. Its single operand is a 2-byte signed integer
		// constant. This operation pops the top of stack. If the value popped is not the constant
		// 0, the 2-byte constant operand is the number of bytes of the DWARF expression to skip
		// forward or backward from the current operation, beginning after the 2-byte constant"
		jmp := int32(int16(sec.byteOrder.Uint16(expr[1:])))
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				if a.value == 0 {
					b, _ := loc.pop()
					return b, nil
				}
				return loclistStackItem{
					class: stackItemClassBranch,
					value: uint32(jmp),
				}, nil
			},
			size:     3,
			operator: "DW_OP_bra",
		}, nil

	case 0x29:
		// DW_OP_eq
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) == int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_eq",
		}, nil
	case 0x2a:
		// DW_OP_ge
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) >= int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_ge",
		}, nil
	case 0x2b:
		// DW_OP_gt
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) > int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_gt",
		}, nil
	case 0x2c:
		// DW_OP_le
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) <= int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_le",
		}, nil
	case 0x2d:
		// DW_OP_lt
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) < int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_lt",
		}, nil
	case 0x2e:
		// DW_OP_ne
		// (control flow operations)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				b, _ := loc.pop()
				var value uint32
				if int32(b.value) != int32(a.value) {
					value = 1
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     1,
			operator: "DW_OP_ne",
		}, nil
	case 0x2f:
		// DW_OP_skip
		// (control flow operations)
		// "DW_OP_skip is an unconditional branch. Its single operand is a 2-byte signed integer
		// constant. The 2-byte constant is the number of bytes of the DWARF expression to skip
		// forward or backward from the current operation, beginning after the 2-byte constant"
		jmp := int32(int16(sec.byteOrder.Uint16(expr[1:])))
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassBranch,
					value: uint32(jmp),
				}, nil
			},
			size:     3,
			operator: "DW_OP_skip",
		}, nil

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
		lit := uint32(expr[0] - 0x30)
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassPush,
					value: lit,
				}, nil
			},
			size:     1,
			operator: fmt.Sprintf("DW_OP_lit%d", lit),
		}, nil

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
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				value, ok := sec.coproc.Register(int(reg))
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown register: %d", reg)
				}
				return loclistStackItem{
					class: stackItemClassValue,
					value: value,
				}, nil
			},
			size:     1,
			operator: fmt.Sprintf("DW_OP_reg%d", reg),
		}, nil

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
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				regVal, ok := sec.coproc.Register(int(reg))
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown register: %d", reg)
				}
				address := uint32(int64(regVal) + offset)

				return loclistStackItem{
					class: stackItemClassPush,
					value: address,
				}, nil
			},
			size:     n + 1,
			operator: fmt.Sprintf("DW_OP_breg%d", reg),
		}, nil

	case 0x90:
		// DW_OP_regx
		// (register location description)
		// "The DW_OP_regx operation has a single unsigned LEB128 literal operand that encodes the
		// name of a register"
		reg, n := leb128.DecodeULEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				value, ok := sec.coproc.Register(int(reg))
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown register: %d", reg)
				}
				return loclistStackItem{
					class: stackItemClassValue,
					value: value,
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_regx",
		}, nil

	case 0x91:
		// DW_OP_fbreg
		// (register based addressing)
		// "The DW_OP_fbreg operation provides a signed LEB128 offset from the address specified by
		// the location description in the DW_AT_frame_base attribute of the current function. (This
		// is typically a “stack pointer” register plus or minus some offset. On more sophisticated
		// systems it might be a location list that adjusts the offset according to changes in the
		// stack pointer as the PC changes)"
		offset, n := leb128.DecodeSLEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, derivation io.Writer) (loclistStackItem, error) {
				fb, err := loc.fb.resolveFramebase(derivation)
				if err != nil {
					return loclistStackItem{}, err
				}
				address := int64(fb) + offset

				return loclistStackItem{
					class: stackItemClassPush,
					value: uint32(address),
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_fbreg",
		}, nil

	case 0x92:
		// DW_OP_bregx
		// (register based addressing)
		// "DW_OP_bregx
		// "The DW_OP_bregx operation has two operands: a register which is specified by an unsigned
		// LEB128 number, followed by a signed LEB128 offset"
		reg, n := leb128.DecodeULEB128(expr[1:])
		offset, m := leb128.DecodeSLEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				regVal, ok := sec.coproc.Register(int(reg))
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown register: %d", reg)
				}
				address := uint32(int64(regVal) + offset)

				return loclistStackItem{
					class: stackItemClassPush,
					value: address,
				}, nil
			},
			size:     m + n + 1,
			operator: "DW_OP_bregx",
		}, nil

	case 0x93:
		// DW_OP_piece
		// (composite location descriptions)
		// "The DW_OP_piece operation takes a single operand, which is an unsigned LEB128 number.
		// The number describes the size in bytes of the piece of the object referenced by the preceding
		// simple location description. If the piece is located in a register, but does not occupy the entire
		// register, the placement of the piece within that register is defined by the ABI"
		size, n := leb128.DecodeULEB128(expr[1:])
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				v := a.value
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
					return loclistStackItem{}, fmt.Errorf("unknown piece size %d", size)
				}

				p := loclistPiece{
					value: v,
					size:  uint32(size),
				}

				// set isAddress flag
				switch a.class {
				case stackItemClassNOP:
				case stackItemClassPush:
					p.isAddress = true
				case stackItemClassValue:
					p.isAddress = false
				default:
					return loclistStackItem{}, fmt.Errorf("unhandled stack entry: %v", a.class)
				}

				// add to list of pieces
				loc.pieces = append(loc.pieces, p)

				return loclistStackItem{
					class: stackItemClassPiece,
				}, nil
			},
			size:     n + 1,
			operator: "DW_OP_piece",
		}, nil

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
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				a, _ := loc.pop()
				address := uint64(a.value)

				value, ok := sec.coproc.Peek(uint32(address))
				if !ok {
					return loclistStackItem{}, fmt.Errorf("unknown address: %08x", address)
				}

				mask := ^((^int32(0)) << (size * 8))
				value &= uint32(mask)

				return loclistStackItem{
					class: stackItemClassPush,
					value: value,
				}, nil
			},
			size:     2,
			operator: "DW_OP_deref_size",
		}, nil

	case 0x95:
		// DW_OP_xdref_size
		// (stack operations)
		return loclistOperator{}, nil

	case 0x96:
		// DW_OP_nop
		// (special operations)
		// "The DW_OP_nop operation is a place holder. It has no effect on the location stack or any
		// of its values"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassNOP,
				}, nil
			},
			size:     1,
			operator: "DW_OP_nop",
		}, nil

	case 0x9c:
		// DW_OP_call_frame_cfa
		// (stack operations)
		// "The DW_OP_call_frame_cfa operation pushes the value of the CFA, obtained from the Call
		// Frame Information"
		return loclistOperator{
			resolve: func(loc *loclist, derivation io.Writer) (loclistStackItem, error) {
				fb, err := loc.fb.resolveFramebase(derivation)
				if err != nil {
					return loclistStackItem{}, err
				}
				return loclistStackItem{
					class: stackItemClassPush,
					value: uint32(fb),
				}, nil
			},
			size:     1,
			operator: "DW_OP_call_frame_cfa",
		}, nil

	case 0x9d:
		// DW_OP_bit_piece
		return loclistOperator{}, nil

	case 0x9e:
		// DW_OP_implicit_value
		// (implicit location descriptions)
		// "The DW_OP_implicit_value operation specifies an immediate value using two operands: an
		// unsigned LEB128 length, followed by a block representing the value in the memory
		// representation of the target machine. The length operand gives the length in bytes of the
		// block"
		length, n := leb128.DecodeULEB128(expr[1:])
		var val uint32
		switch length {
		case 1:
			val = uint32(expr[1+n])
		case 2:
			val = uint32(sec.byteOrder.Uint16(expr[1+n:]))
		case 4:
			val = sec.byteOrder.Uint32(expr[1+n:])
		default:
			return loclistOperator{}, fmt.Errorf("unsupported length value for DW_OP_implicit_value")
		}
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				return loclistStackItem{
					class: stackItemClassValue,
					value: val,
				}, nil
			},
			size:     int(length) + n + 1,
			operator: "DW_OP_implicit_value",
		}, nil

	case 0x9f:
		// DW_OP_stack_value
		// (implicit location descriptions)
		// "The DW_OP_stack_value operation specifies that the object does not exist in memory but
		// its value is nonetheless known and is at the top of the DWARF expression stack. In this
		// form of location description, the DWARF expression represents the actual value of the
		// object, rather than its location. The DW_OP_stack_value operation terminates the
		// expression"
		return loclistOperator{
			resolve: func(loc *loclist, _ io.Writer) (loclistStackItem, error) {
				res, ok := loc.pop()
				if !ok {
					return loclistStackItem{}, fmt.Errorf("stack empty")
				}
				res.class = stackItemClassValue
				return res, nil
			},
			size:     1,
			operator: "DW_OP_stack_value",
		}, nil

	case 0xf0:
		// DW_OP_GNU_uninit
		//
		// this is the only GNU DWARF extension we implement. we do so because this operator is
		// sometimes emitted even when -gstrict-dwarf is specified when compiling
		return loclistOperator{}, nil
	}

	return loclistOperator{}, fmt.Errorf("%w: unsupported expression operator %#02x", UnsupportedDWARF, expr[0])
}
