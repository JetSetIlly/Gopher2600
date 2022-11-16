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

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"

type resolveCoproc interface {
	coproc() mapper.CartCoProc
	framebase() uint64
	lastResolved() Resolved
}

type Resolved struct {
	address uint64
	value   uint32
	valueOk bool
}

type resolver func(resolveCoproc) Resolved

// decode DWARF data of ClassExprLoc
//
// returns nil and zero if expression cannot be handled
func decodeSingleLocationDescription(expr []uint8) (resolver, int) {
	// expression location operators reference
	//
	// "DWARF Debugging Information Format Version 4", page 17, section 2.5.1.1
	// table of value on page 153, section 7.7.1 DWARF Expressions

	switch expr[0] {
	case 0x03:
		return func(r resolveCoproc) Resolved {
			// DW_OP_addr
			// constant address
			address := uint64(expr[1])
			address |= uint64(expr[2]) << 8
			address |= uint64(expr[3]) << 16
			address |= uint64(expr[4]) << 24
			value, ok := r.coproc().CoProcRead32bit(uint32(address))
			return Resolved{
				address: address,
				value:   value,
				valueOk: ok,
			}
		}, 5
	case 0x23:
		// DW_OP_plus_uconst
		// ULEB128 to be added to previous value on the stack
		return func(r resolveCoproc) Resolved {
			address := decodeULEB128(expr[1:5])
			value, ok := r.coproc().CoProcRead32bit(uint32(address))
			return Resolved{
				address: address,
				value:   value,
				valueOk: ok,
			}
		}, 5
	case 0x91:
		// DW_OP_fbreg
		return func(r resolveCoproc) Resolved {
			address := r.framebase() + decodeSLEB128(expr[1:5])
			value, ok := r.coproc().CoProcRead32bit(uint32(address))
			return Resolved{
				address: address,
				value:   value,
				valueOk: ok,
			}
		}, 5
	case 0x50:
		// DW_OP_reg0
		return func(r resolveCoproc) Resolved {
			reg := 0
			value := r.coproc().CoProcRegister(reg)
			return Resolved{
				address: uint64(reg),
				value:   value,
				valueOk: true,
			}
		}, 1
	case 0x51:
		// DW_OP_reg1
		return func(r resolveCoproc) Resolved {
			reg := 1
			value := r.coproc().CoProcRegister(reg)
			return Resolved{
				address: uint64(reg),
				value:   value,
				valueOk: true,
			}
		}, 1
	case 0x52:
		// DW_OP_reg2
		return func(r resolveCoproc) Resolved {
			reg := 2
			value := r.coproc().CoProcRegister(reg)
			return Resolved{
				address: uint64(reg),
				value:   value,
				valueOk: true,
			}
		}, 1
	case 0x53:
		// DW_OP_reg3
		return func(r resolveCoproc) Resolved {
			reg := 3
			value := r.coproc().CoProcRegister(reg)
			return Resolved{
				address: uint64(reg),
				value:   value,
				valueOk: true,
			}
		}, 1
	}

	return nil, 0
}

// some ClassExprLoc operands are expressed as unsigned LEB128 values.
// algorithm taken from page 218 of "DWARF4 Standard", figure 46
func decodeULEB128(encoded []uint8) uint64 {
	var result uint64
	var shift uint64
	for _, v := range encoded {
		result |= (uint64(v & 0x7f)) << shift
		if v&0x80 == 0x00 {
			break
		}
		shift += 7
	}
	return result
}

// some ClassExprLoc operands are expressed as signed LEB128 values
// algorithm taken from page 218 of "DWARF4 Standard", figure 47
func decodeSLEB128(encoded []uint8) uint64 {
	var result uint64
	var shift uint64
	const size = 32

	var v uint8
	for _, v = range encoded {
		result |= (uint64(v & 0x7f)) << shift
		shift += 7
		if v&0x80 == 0x00 {
			break
		}
	}

	// sign extend last byte from the encoded slice
	if (shift < size) && v&0x80 == 0x80 {
		result |= ((^uint64(0)) >> shift) << shift
	}

	return result
}
