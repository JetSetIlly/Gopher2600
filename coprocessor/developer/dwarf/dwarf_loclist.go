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
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/logger"
)

type loclistSection struct {
	coproc    coprocessor.CartCoProc
	byteOrder binary.ByteOrder
	data      []uint8
}

func newLoclistSectionFromFile(ef *elf.File, coproc coprocessor.CartCoProc) (*loclistSection, error) {
	sec := ef.Section(".debug_loc")
	if sec == nil {
		return nil, fmt.Errorf("no .debug_loc section")
	}
	data, err := sec.Data()
	if err != nil {
		return nil, err
	}
	return newLoclistSection(data, ef.ByteOrder, coproc)
}

func newLoclistSection(data []uint8, byteOrder binary.ByteOrder, coproc coprocessor.CartCoProc) (*loclistSection, error) {
	sec := &loclistSection{
		data:      data,
		coproc:    coproc,
		byteOrder: byteOrder,
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty .debug_loc section")
	}

	return sec, nil
}

// loclistFramebase provides context to the location list. implemented by
// SourceVariable, SourceFunction and the frame section.
type loclistFramebase interface {
	framebase() (uint64, error)
}

type loclistStackClass int

const (
	stackClassNOP loclistStackClass = iota
	stackClassPush
	stackClassIsValue
	stackClassSingleAddress
	stackClassPiece
)

type loclistStack struct {
	class loclistStackClass
	value uint32
}

type loclistPiece struct {
	value     uint32
	isAddress bool
	size      uint32
}

type loclistOperator struct {
	operator string
	resolve  func(*loclist) (loclistStack, error)
}

type loclistResult struct {
	address    uint64
	hasAddress bool
	value      uint32

	// the result only has pieces for composite variable types (ie. structs, etc.)
	pieces []loclistPiece
}

type loclist struct {
	coproc coprocessor.CartCoProc
	ctx    loclistFramebase

	list []loclistOperator

	stack  []loclistStack
	pieces []loclistPiece

	singleLoc  bool
	loclistPtr int64

	// the derivation for the loclist is written to the io.Writer
	derivation io.Writer
}

func (sec *loclistSection) newLoclistJustContext(ctx loclistFramebase) *loclist {
	return &loclist{
		coproc:    sec.coproc,
		ctx:       ctx,
		singleLoc: true,
	}
}

func (sec *loclistSection) newLoclistFromSingleOperator(ctx loclistFramebase, expr []uint8) (*loclist, error) {
	loc := &loclist{
		coproc:    sec.coproc,
		ctx:       ctx,
		singleLoc: true,
	}
	op, n, err := sec.decodeLoclistOperation(expr)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("unhandled expression operator %02x", expr[0])
	}
	loc.list = append(loc.list, op)
	return loc, nil
}

type commitLoclist func(start, end uint64, loc *loclist)

func (sec *loclistSection) newLoclist(ctx loclistFramebase, ptr int64,
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

	loclistNumber := ptr

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
			coproc:     sec.coproc,
			ctx:        ctx,
			loclistPtr: loclistNumber,
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
				r, n, err := sec.decodeLoclistOperation(sec.data[ptr:])
				if err != nil {
					return err
				}
				if n == 0 {
					return fmt.Errorf("unhandled expression operator %02x", sec.data[ptr])
				}

				// add resolver to variable
				loc.addOperator(r)

				// reduce length value
				length -= n

				// advance sec pointer by length value
				ptr += int64(n)
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

		// update loclist number
		loclistNumber = ptr

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

func (loc *loclist) resolve() (loclistResult, error) {
	if loc.ctx == nil {
		return loclistResult{}, fmt.Errorf("no context [%x]", loc.loclistPtr)
	}

	if len(loc.list) == 0 {
		return loclistResult{}, fmt.Errorf("no loclist operations defined [%x]", loc.loclistPtr)
	}

	// clear lists
	loc.stack = loc.stack[:0]
	loc.pieces = loc.pieces[:0]

	// whether the top of the stack is a value or an address
	var isValue bool

	// resolve every entry in the loclist
	for i := range loc.list {
		s, err := loc.list[i].resolve(loc)
		if err != nil {
			return loclistResult{}, fmt.Errorf("%s: %w", loc.list[i].operator, err)
		}

		// process result according to the result class
		switch s.class {
		case stackClassNOP:
			// do nothing
		case stackClassPush:
			loc.stack = append(loc.stack, s)
		case stackClassIsValue:
			loc.stack = append(loc.stack, s)
			isValue = true
		case stackClassSingleAddress:
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
		case stackClassPiece:
			// all functionality of a piece operation is contained in the actual loclistOperator implementation
		}

		if loc.derivation != nil {
			loc.derivation.Write([]byte(fmt.Sprintf("%s %08x", loc.list[i].operator, s.value)))
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
		return loclistResult{}, fmt.Errorf("stack is empty [%x]", loc.loclistPtr)
	}

	// stack should only have one entry in it
	if len(loc.stack) > 1 {
		logger.Logf(logger.Allow, "dwarf", "loclist stack has more than one entry after resolve [%x]", loc.loclistPtr)
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

func (loc *loclist) peek() loclistStack {
	if len(loc.stack) == 0 {
		return loclistStack{}
	}
	return loc.stack[len(loc.stack)-1]
}

func (loc *loclist) pop() (loclistStack, bool) {
	l := len(loc.stack)
	if l == 0 {
		return loclistStack{}, false
	}
	s := loc.stack[l-1]
	loc.stack = loc.stack[:l-1]
	return s, true
}
