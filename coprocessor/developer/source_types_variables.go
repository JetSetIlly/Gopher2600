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
	"sync/atomic"
)

// addressResolution allows a SourceVariable instance to retrieve its own
// value from a coprocessor's memory (including registers if necssary)
type addressResolution interface {
	// read 8, 16 or 32 bit values from the address. the address should be in
	// the range given in one of the CartStaticSegment returned by the
	// Segments() function.
	Read8bit(addr uint32) (uint8, bool)
	Read16bit(addr uint32) (uint16, bool)
	Read32bit(addr uint32) (uint32, bool)
}

// SourceVariable is a single variable identified by the DWARF data.
type SourceVariable struct {
	Cart CartCoProcDeveloper

	// name of variable
	Name string

	// variable type (int, char, etc.)
	Type *SourceType

	// first source line for each instance of the function
	DeclLine *SourceLine

	// if addressResolve is not nil then the result of the function is the
	// address and not the address field
	addressResolve func(addressResolution) uint64

	// origin address of variable
	origin uint64

	// most recent values retreived from emulation
	cachedValue   atomic.Value // uint32
	cachedValueOk atomic.Value // bool
	cachedAddress atomic.Value // uint64

	// child variables of this variable. this includes array elements, struct
	// members and dereferenced variables
	children []*SourceVariable
}

// address differs from Address() in that the results from the emulation are
// immediate
//
// only use this function when we sure that we are inside the emulation
// goroutine
func (varb *SourceVariable) address() uint64 {
	if varb.addressResolve == nil {
		return 0
	}
	if varb.Cart == nil {
		return 0
	}

	bus := varb.Cart.GetStaticBus()
	return varb.addressResolve(bus.GetStatic()) + varb.origin
}

// Address returns the location in memory of the variable referred to by
// SourceVariable
func (varb *SourceVariable) Address() uint64 {
	varb.Cart.PushFunction(func() {
		varb.cachedAddress.Store(varb.address())
	})

	var address uint64

	var ok bool
	if address, ok = varb.cachedAddress.Load().(uint64); !ok {
		return 0
	}

	return uint64(address)
}

// setAddress updates the SourceVariable to refer to the variable at the
// specified address
//
// address argument should be type uint64 or "func(addressResolution) uint64"
func (varb *SourceVariable) setAddress(address interface{}) {
	switch a := address.(type) {
	case uint64:
		varb.addressResolve = func(_ addressResolution) uint64 {
			return a
		}
	case func(addressResolution) uint64:
		varb.addressResolve = a
	default:
		panic(fmt.Sprintf("unsupported address argument %T", address))
	}
}

func (varb *SourceVariable) String() string {
	return fmt.Sprintf("%s %s => %#08x", varb.Type.Name, varb.Name, varb.Address())
}

// Value returns the current value of a SourceVariable. It's good practice to
// use this function to read memory, rather than reading memory directly,
// because it will handle any required address resolution before accessing the
// memory.
func (varb *SourceVariable) Value() (uint32, bool) {
	if varb.Cart == nil {
		return 0, false
	}

	varb.Cart.PushFunction(func() {
		var val uint32
		var valOk bool

		addr := uint32(varb.Address())

		switch varb.Type.Size {
		case 1:
			var v uint8
			bus := varb.Cart.GetStaticBus()
			v, valOk = bus.GetStatic().Read8bit(addr)
			val = uint32(v) & varb.Type.Mask()
		case 2:
			var v uint16
			bus := varb.Cart.GetStaticBus()
			v, valOk = bus.GetStatic().Read16bit(addr)
			val = uint32(v) & varb.Type.Mask()
		case 4:
			var v uint32
			bus := varb.Cart.GetStaticBus()
			v, valOk = bus.GetStatic().Read32bit(addr)
			val = uint32(v) & varb.Type.Mask()
		default:
		}

		varb.cachedValue.Store(val)
		varb.cachedValueOk.Store(valOk)
	})

	var val uint32
	var valOk bool

	var ok bool
	if val, ok = varb.cachedValue.Load().(uint32); !ok {
		return 0, false
	}

	if valOk, ok = varb.cachedValueOk.Load().(bool); !ok {
		return 0, false
	}

	return val, valOk
}

// NumChildren returns the number of children for this variable
func (varb *SourceVariable) NumChildren() int {
	return len(varb.children)
}

// Child returns the i'th child of the variable. A child can be an array
// element, composite member or dereferenced variable, as appropriate for the
// variables SourceType
//
// Count from zero. Returns nil if no such child exists
func (varb *SourceVariable) Child(i int) *SourceVariable {
	if i >= len(varb.children) {
		return nil
	}
	return varb.children[i]
}
