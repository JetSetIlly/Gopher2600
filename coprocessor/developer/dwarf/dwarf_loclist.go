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
	"encoding/binary"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/logger"
)

type loclistDecoder struct {
	coproc    coprocessor.CartCoProc
	byteOrder binary.ByteOrder
	data      []uint8
}

// newLoclistDecoder will always return a new instance of loclistDecoder but it
// may come with a warning about the .debug_loc section being empty
func newLoclistDecoder(data []uint8, byteOrder binary.ByteOrder, coproc coprocessor.CartCoProc) (*loclistDecoder, error) {
	sec := &loclistDecoder{
		data:      data,
		coproc:    coproc,
		byteOrder: byteOrder,
	}
	if len(data) == 0 {
		return sec, fmt.Errorf("empty .debug_loc section")
	}
	return sec, nil
}

// framebaseResolver provides context to the location list. implemented by
// SourceVariable, SourceFunction and the frame section.
type framebaseResolver interface {
	resolveFramebase(derivation io.Writer) (uint64, error)
}

type loclistStackItemClass int

const (
	stackItemClassNOP loclistStackItemClass = iota
	stackItemClassPush
	stackItemClassValue
	stackItemClassAddress
	stackItemClassPiece
	stackItemClassBranch
)

type loclistStackItem struct {
	class loclistStackItemClass

	// meaning of value depends on the value of the class field
	value uint32
}

type loclistPiece struct {
	value     uint32
	isAddress bool
	size      uint32
}

type loclistOperator struct {
	operator string
	size     int

	// resoluve function returns an item to put on the stack and an error
	resolve func(*loclist, io.Writer) (loclistStackItem, error)
}

type loclistResult struct {
	address    uint64
	hasAddress bool
	value      uint32

	// the result only has pieces for composite variable types (ie. structs, etc.)
	pieces []loclistPiece
}

type loclistClass int

const (
	classExpr loclistClass = iota
	classPtr
	classFramebase
)

type loclist struct {
	coproc coprocessor.CartCoProc
	fb     framebaseResolver

	list []loclistOperator

	stack  []loclistStackItem
	pieces []loclistPiece

	class loclistClass
	ptr   int64
}

func (sec *loclistDecoder) newLoclistJustFramebase(fb framebaseResolver) *loclist {
	return &loclist{
		coproc: sec.coproc,
		fb:     fb,
		class:  classFramebase,
	}
}

// newLoclistFromExpr only supports single operator expressions for now (because that's all I've
// encountered)
func (sec *loclistDecoder) newLoclistFromExpr(fb framebaseResolver, expr []uint8) (*loclist, error) {
	loc := &loclist{
		coproc: sec.coproc,
		fb:     fb,
		class:  classExpr,
	}
	op, err := sec.decodeLoclistOperation(expr)
	if err != nil {
		return nil, err
	}
	if op.size == 0 {
		return nil, fmt.Errorf("unhandled expression operator %#02x", expr[0])
	}
	if op.size*8 < len(expr) {
		// only expressions with one operator are supported for now
		return nil, fmt.Errorf("unexpected number of bytes in loclist expression: %d", len(expr))
	}
	loc.list = append(loc.list, op)
	return loc, nil
}

type commitLoclist func(start, end uint64, loc *loclist)

func (sec *loclistDecoder) newLoclistFromPtr(fb framebaseResolver, ptr int64,
	compilationUnitAddress uint64, commit commitLoclist) error {

	// "Location lists, which are used to describe objects that have a limited lifetime or change
	// their location during their lifetime. Location lists are more completely described below."
	// page 26 of "DWARF4 Standard"
	//
	// "Location lists are used in place of location expressions whenever the object whose location is
	// being described can change location during its lifetime. Location lists are contained in a separate
	// object file section called .debug_loc . A location list is indicated by a location attribute whose
	// value is an offset from the beginning of the .debug_loc section to the first byte of the list for the
	// object in question"
	// page 30 of "DWARF4 Standard"
	//
	// "loclistptr: This is an offset into the .debug_loc section (DW_FORM_sec_offset). It consists
	// of an offset from the beginning of the .debug_loc section to the first byte of the data making up
	// the location list for the compilation unit. It is relocatable in a relocatable object file, and
	// relocated in an executable or shared object. In the 32-bit DWARF format, this offset is a 4-
	// byte unsigned value; in the 64-bit DWARF format, it is an 8-byte unsigned value (see
	// Section 7.4)"
	// page 148 of "DWARF4 Standard"

	// "The applicable base address of a location list entry is determined by the closest preceding base
	// address selection entry (see below) in the same location list. If there is no such selection entry,
	// then the applicable base address defaults to the base address of the compilation unit (see
	// Section 3.1.1)"
	//
	// "A base address selection entry affects only the list in which it is contained"
	// page 31 of "DWARF4 Standard"
	baseAddress := compilationUnitAddress

	if int(ptr)+8 > len(sec.data) {
		return fmt.Errorf("loclist ptr beyond scope of .debug_loc data: %d > %d", ptr, len(sec.data))
	}

	// start and end address. this will be updated at the end of every for loop iteration
	startAddress := uint64(sec.byteOrder.Uint32(sec.data[ptr:]))
	ptr += 4
	endAddress := uint64(sec.byteOrder.Uint32(sec.data[ptr:]))
	ptr += 4

	// "The end of any given location list is marked by an end of list entry, which consists of a 0 for the
	// beginning address offset and a 0 for the ending address offset. A location list containing only an
	// end of list entry describes an object that exists in the source code but not in the executable
	// program". page 31 of "DWARF4 Standard"
	for !(startAddress == 0x0 && endAddress == 0x0) {
		loc := &loclist{
			coproc: sec.coproc,
			fb:     fb,
			class:  classPtr,

			// we've already read the start and end address information so we
			// adjust the ptr by the length of two 32byte values
			ptr: ptr - 8,
		}

		// "A base address selection entry consists of:
		// 1. The value of the largest representable address offset (for example, 0xffffffff when the size of
		// an address is 32 bits).
		// 2. An address, which defines the appropriate base address for use in interpreting the beginning
		// and ending address offsets of subsequent entries of the location list"
		// page 31 of "DWARF4 Standard"
		if startAddress == 0xffffffff {
			baseAddress = endAddress
		} else {
			// reduce end address by one. this is because the value we've read "marks the
			// first address past the end of the address range over which the location is
			// valid" (page 30 of "DWARF4 Standard")
			endAddress -= 1

			// length of expression
			length := int(sec.byteOrder.Uint16(sec.data[ptr:]))
			ptr += 2

			// loop through stack operations
			for length > 0 {
				op, err := sec.decodeLoclistOperation(sec.data[ptr:])
				if err != nil {
					return err
				}
				if op.size == 0 {
					return fmt.Errorf("unhandled expression operator %#02x", sec.data[ptr])
				}

				// add resolver to variable
				loc.addOperator(op)

				// reduce length value
				length -= op.size

				// advance sec pointer by length value
				ptr += int64(op.size)
			}

			// "A location list entry (but not a base address selection or end of list entry) whose beginning
			// and ending addresses are equal has no effect because the size of the range covered by such
			// an entry is zero". page 31 of "DWARF4 Standard"
			//
			// "The ending address must be greater than or equal to the beginning address"
			// page 30 of "DWARF4 Standard"
			if startAddress < endAddress {
				commit(startAddress+baseAddress, endAddress+baseAddress, loc)
			}
		}

		// bounds check
		if int(ptr)+8 > len(sec.data) {
			return fmt.Errorf("loclist ptr beyond scope of .debug_loc data: %d > %d", ptr, len(sec.data))
		}

		// read next address range
		startAddress = uint64(sec.byteOrder.Uint32(sec.data[ptr:]))
		ptr += 4
		endAddress = uint64(sec.byteOrder.Uint32(sec.data[ptr:]))
		ptr += 4
	}

	return nil
}

func (loc *loclist) addOperator(r loclistOperator) {
	loc.list = append(loc.list, r)
}

func (loc *loclist) resolve(derivation io.Writer) (loclistResult, error) {
	if loc.fb == nil {
		return loclistResult{}, fmt.Errorf("no context [%x]", loc.ptr)
	}

	if len(loc.list) == 0 {
		return loclistResult{}, fmt.Errorf("no loclist operations defined [%x]", loc.ptr)
	}

	// clear lists
	loc.stack = loc.stack[:0]
	loc.pieces = loc.pieces[:0]

	// whether the top of the stack is a value or an address
	var isValue bool

	// resolve every entry in the loclist
	var i int
	for i < len(loc.list) {
		s, err := loc.list[i].resolve(loc, derivation)
		if err != nil {
			return loclistResult{}, fmt.Errorf("%s: %w", loc.list[i].operator, err)
		}

		if derivation != nil {
			fmt.Fprintf(derivation, "%s %08x\n", loc.list[i].operator, s.value)
		}

		// process result according to the result class
		switch s.class {
		case stackItemClassNOP:
			// do nothing

		case stackItemClassPush:
			loc.push(s)

		case stackItemClassValue:
			loc.push(s)
			isValue = true

		case stackItemClassAddress:
			r := loclistResult{
				address:    uint64(s.value),
				hasAddress: true,
			}

			var ok bool
			r.value, ok = loc.coproc.Peek(s.value)
			if !ok {
				return loclistResult{}, fmt.Errorf("%s: error resolving address %08x", loc.list[i].operator, s.value)
			}

			return r, nil

		case stackItemClassPiece:
			// all functionality of a piece operation is contained in the actual loclistOperator implementation

		case stackItemClassBranch:
			jmp := int(int16(s.value))
			j := i + 1
			for j < len(loc.list) {
				jmp -= loc.list[j].size
				j++
				if jmp == 0 {
					break // for loop
				}
				if jmp < 0 {
					return loclistResult{}, fmt.Errorf("%s: unusable value for branch [%d]", loc.list[i].operator, jmp)
				}
			}
			if jmp != 0 {
				return loclistResult{}, fmt.Errorf("%s: unexpected end for branch [%d]", loc.list[i].operator, jmp)
			}
			if j == i {
				return loclistResult{}, fmt.Errorf("%s: branch jumps back to itself", loc.list[i].operator)
			}
			if j >= len(loc.list) {
				return loclistResult{}, fmt.Errorf("%s: branch has jumped too far", loc.list[i].operator)
			}
			i = j

		default:
			return loclistResult{}, fmt.Errorf("%s: unhandled stackItemClass [%d]", loc.list[i].operator, s.class)
		}

		// i has already been set for branches
		if s.class != stackItemClassBranch {
			i++
		}
	}

	// return assembled pieces
	if len(loc.pieces) > 0 {
		var r loclistResult
		r.pieces = append(r.pieces, loc.pieces...)
		return r, nil
	}

	// no pieces so just use top of stack

	if len(loc.stack) == 0 {
		return loclistResult{}, fmt.Errorf("stack is empty [%x]", loc.ptr)
	}

	// stack should only have one entry in it
	if len(loc.stack) > 1 {
		logger.Logf(logger.Allow, "dwarf", "loclist stack has more than one entry after resolve [%x]", loc.ptr)
	}

	// top of stack is the result
	s := loc.stack[len(loc.stack)-1]

	// is the top of the stack a valid value or is it an address
	if isValue {
		r := loclistResult{
			value: s.value,
		}
		return r, nil
	}

	// top of the stack is an address. how this address is interpreted depends
	// on context
	return loclistResult{
		address:    uint64(s.value),
		hasAddress: true,
	}, nil
}

func (loc *loclist) peek(n int) (loclistStackItem, bool) {
	if n < len(loc.stack) {
		return loc.stack[len(loc.stack)-n], true
	}
	return loclistStackItem{}, false
}

func (loc *loclist) pop() (loclistStackItem, bool) {
	l := len(loc.stack)
	if l == 0 {
		return loclistStackItem{}, false
	}
	s := loc.stack[l-1]
	loc.stack = loc.stack[:l-1]
	return s, true
}

func (loc *loclist) push(b loclistStackItem) {
	loc.stack = append(loc.stack, b)
}
