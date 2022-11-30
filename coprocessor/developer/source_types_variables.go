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
	"strings"
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// SourceVariable is a single local variable identified by the DWARF data.
type SourceVariableLocal struct {
	*SourceVariable

	LexStart uint64
	LexEnd   uint64

	// the address range for which the variable is valid
	ResolvableStart uint64
	ResolvableEnd   uint64
}

func (varb *SourceVariableLocal) String() string {
	return varb.SourceVariable.String()
}

// In returns true if the address of any of the instructions associated with
// the SourceLine are within the lexical address range of the variable.
func (varb *SourceVariableLocal) In(ln *SourceLine) bool {
	for _, d := range ln.Disassembly {
		if d.Addr >= uint32(varb.LexStart) && d.Addr <= uint32(varb.LexEnd) {
			return true
		}
		if d.Addr >= uint32(varb.ResolvableStart) && d.Addr <= uint32(varb.ResolvableEnd) {
			return true
		}
	}
	return false
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

	// location list resolves a Location. may be nil
	loclist *loclist

	// if Unresolvable is true then an error was enounterd during a resolve()
	// sequence. when setting to the error will be logged and all future
	// attempts at resolution will silently fail
	Unresolvable bool

	// most recent resolved value retrieved from emulation
	cachedLocation atomic.Value // Location

	// child variables of this variable. this includes array elements, struct
	// members and dereferenced variables
	children []*SourceVariable
}

func (varb *SourceVariable) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("%s %s = ", varb.Type.Name, varb.Name))
	if v, ok := varb.Value(); ok {
		s.WriteString(fmt.Sprintf(varb.Type.Hex(), v))
	} else {
		s.WriteString("unresolvable")
	}
	return s.String()
}

// Address returns the location in memory of the variable referred to by
// SourceVariable
func (varb *SourceVariable) Address() (uint64, bool) {
	varb.Cart.PushFunction(func() {
		varb.cachedLocation.Store(varb.resolve())
	})

	var r location
	var ok bool
	if r, ok = varb.cachedLocation.Load().(location); !ok {
		return 0, false
	}

	return r.address, r.addressOk
}

// Value returns the current value of a SourceVariable. It's good practice to
// use this function to read memory, rather than reading memory directly using
// the result of Address(), because it will handle any required address
// resolution before accessing the memory.
func (varb *SourceVariable) Value() (uint32, bool) {
	varb.Cart.PushFunction(func() {
		varb.cachedLocation.Store(varb.resolve())
	})

	var r location
	var ok bool
	if r, ok = varb.cachedLocation.Load().(location); !ok {
		return 0, false
	}

	return r.value & varb.Type.Mask(), r.valueOk
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

// coproc implements the resolver interface
func (varb *SourceVariable) coproc() mapper.CartCoProc {
	return varb.Cart.GetCoProc()
}

// coproc implements the resolver interface
func (varb *SourceVariable) framebase() (uint64, error) {
	if varb.DeclLine == nil || varb.DeclLine.Function == nil {
		return 0, fmt.Errorf("no framebase")
	}

	if varb.DeclLine.Function.framebaseList == nil {
		return 0, fmt.Errorf("no framebase for function %s", varb.DeclLine.Function.Name)
	}

	location, err := varb.DeclLine.Function.framebaseList.resolve()
	if err != nil {
		return 0, fmt.Errorf("framebase for function %s: %v", varb.DeclLine.Function.Name, err)
	}

	return uint64(location.value), nil
}

// resolve address/value
func (varb *SourceVariable) resolve() location {
	if varb.Unresolvable || varb.loclist == nil {
		return location{}
	}

	loc, err := varb.loclist.resolve()
	if err != nil {
		varb.Unresolvable = true
		logger.Logf("dwarf", "%s: unresolvable: %v", varb.Name, err)
		return location{}
	}

	return loc
}

// addVariableChildren populates the variable child array with SourceVariable
// instances that describe areas of memory related to the parent variable.
func (varb *SourceVariable) addVariableChildren() {
	if varb.Type.IsArray() {
		for i := 0; i < varb.Type.ElementCount; i++ {
			elem := &SourceVariable{
				Cart:     varb.Cart,
				Name:     fmt.Sprintf("%s[%d]", varb.Name, i),
				Type:     varb.Type.ElementType,
				DeclLine: varb.DeclLine,
			}
			elem.loclist = newLoclistJustContext(varb)

			if varb.loclist != nil {
				o := i
				elem.loclist.addOperator(func(_ *loclist) (location, error) {
					r := varb.loclist.lastResolved()
					address := r.address + uint64(o*varb.Type.ElementType.Size)
					value, ok := varb.Cart.GetCoProc().CoProcRead32bit(uint32(address))
					return location{
						address:   address,
						addressOk: r.addressOk, // address is a derivative of lastResolved().address
						value:     value,
						valueOk:   ok,
					}, nil
				})
			}

			varb.children = append(varb.children, elem)
			elem.addVariableChildren()
		}
	}

	if varb.Type.IsComposite() {
		var offset uint64
		for _, m := range varb.Type.Members {
			memb := &SourceVariable{
				Cart:     varb.Cart,
				Name:     m.Name,
				Type:     m.Type,
				DeclLine: varb.DeclLine,
			}
			memb.loclist = newLoclistJustContext(varb)

			if varb.loclist != nil {
				o := offset
				memb.loclist.addOperator(func(_ *loclist) (location, error) {
					r := varb.loclist.lastResolved()
					address := r.address + o
					value, ok := varb.Cart.GetCoProc().CoProcRead32bit(uint32(address))
					return location{
						address:   address,
						addressOk: r.addressOk, // address is a derivative of lastResolved().address
						value:     value,
						valueOk:   ok,
					}, nil
				})
			}

			varb.children = append(varb.children, memb)
			memb.addVariableChildren()

			offset += uint64(m.Type.Size)
		}
	}

	if varb.Type.IsPointer() {
		deref := &SourceVariable{
			Cart:     varb.Cart,
			Name:     fmt.Sprintf("*%s", varb.Name),
			Type:     varb.Type.PointerType,
			DeclLine: varb.DeclLine,
		}
		deref.loclist = newLoclistJustContext(varb)

		if varb.loclist != nil {
			deref.loclist.addOperator(func(_ *loclist) (location, error) {
				r := varb.loclist.lastResolved()
				address := uint64(r.value)
				value, ok := varb.Cart.GetCoProc().CoProcRead32bit(uint32(address))
				return location{
					address:   address,
					addressOk: r.valueOk, // address is a derivative of lastResolved().value
					value:     value,
					valueOk:   ok,
				}, nil
			})
		}

		varb.children = append(varb.children, deref)
		deref.addVariableChildren()
	}
}
