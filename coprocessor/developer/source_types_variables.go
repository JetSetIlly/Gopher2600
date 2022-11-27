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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// SourceVariable is a single local variable identified by the DWARF data.
type SourceVariableLocal struct {
	*SourceVariable

	// the address range for which the variable is valid
	StartAddress uint64
	EndAddress   uint64
}

func (varb *SourceVariableLocal) String() string {
	return fmt.Sprintf("%s %08x -> %08x", varb.Name, varb.StartAddress, varb.EndAddress)
}

// In returns true if the address of any of the instructions associated with
// the SourceLine are within the address range of the variable.
func (varb *SourceVariableLocal) In(ln *SourceLine) bool {
	for _, d := range ln.Disassembly {
		if d.Addr >= uint32(varb.StartAddress) && d.Addr <= uint32(varb.EndAddress) {
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

	// if unresolvable is true then an error was enounterd during a resolve()
	// sequence. when setting to the error will be logged and all future
	// attempts at resolution will silently fail
	unresolvable bool

	// location list resolves a Location
	loclist *loclist

	// most recent resolved value retrieved from emulation
	cachedLocation atomic.Value // Location

	// child variables of this variable. this includes array elements, struct
	// members and dereferenced variables
	children []*SourceVariable
}

func (varb *SourceVariable) String() string {
	return fmt.Sprintf("%s %s", varb.Type.Name, varb.Name)
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

	location, err := varb.DeclLine.Function.framebase.resolve()
	if err != nil {
		return 0, fmt.Errorf("framebase for function %s: %v", varb.DeclLine.Function.Name, err)
	}

	return location.address, nil
}

// resolve address/value
func (varb *SourceVariable) resolve() location {
	if varb.unresolvable {
		return location{}
	}

	loc, err := varb.loclist.resolve()
	if err != nil {
		varb.unresolvable = true
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
		varb.children = append(varb.children, deref)
		deref.addVariableChildren()
	}
}
