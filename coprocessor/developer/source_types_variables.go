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

	"github.com/jetsetilly/gopher2600/logger"
)

// SourceVariableLocal represents a single local variable identified by the
// DWARF data.
type SourceVariableLocal struct {
	*SourceVariable

	// the address range for which the variable is valid
	Range SourceRange
}

// the set of local variables can share a name but they cannot share a name and
// a declaration line. id() returns an identifier for the local variable
//
// note however that there may be multiple variables with the same id, these
// are the same variable but with different resolution information (resolve
// start/end and loclist)
func (local *SourceVariableLocal) id() string {
	return fmt.Sprintf("%s %s", local.Name, local.DeclLine)
}

// SourceVariable is a single variable identified by the DWARF data.
type SourceVariable struct {
	// name of variable
	Name string

	// variable type (int, char, etc.)
	Type *SourceType

	// first source line for each instance of the function
	DeclLine *SourceLine

	// location list resolves a Location. may be nil which indicates that the
	// variable can never be located
	loclist *loclist

	// if ErrorOnResolve is not nil then an error was enountered during a
	// resolve() sequence. the error will be logged when the field is first set
	// to true
	ErrorOnResolve error

	// most recent resolved value retrieved from emulation
	cachedLocation atomic.Value // Location

	// child variables of this variable. this includes array elements, struct
	// members and dereferenced variables
	children []*SourceVariable
}

func (varb *SourceVariable) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("%s = ", varb.decl()))
	if v, ok := varb.Value(); ok {
		s.WriteString(fmt.Sprintf(varb.Type.Hex(), v))
	} else {
		s.WriteString("unresolvable")
	}
	return s.String()
}

// decl returns the type-name and name pair
func (varb *SourceVariable) decl() string {
	return fmt.Sprintf("%s %s", varb.Type.Name, varb.Name)
}

// Address returns the location in memory of the variable referred to by
// SourceVariable
func (varb *SourceVariable) Address() (uint64, bool) {
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
	var r location
	var ok bool
	if r, ok = varb.cachedLocation.Load().(location); !ok {
		return 0, false
	}
	return r.value & varb.Type.Mask(), r.valueOk
}

// Derivation returns the sequence of results that led to the most recent value.
func (varb *SourceVariable) Derivation() []location {
	if varb.loclist == nil {
		return []location{}
	}
	return varb.loclist.derivation
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

// Update variable. It should be called periodically before using the return
// value from Address() or Value()
//
// Be careful to only call this from the emulation goroutine.
func (varb *SourceVariable) Update() {
	varb.cachedLocation.Store(varb.resolve())
	for _, c := range varb.children {
		c.Update()
	}
}

// resolve address/value
func (varb *SourceVariable) resolve() location {
	if varb.ErrorOnResolve != nil {
		return location{}
	}

	if varb.loclist == nil {
		varb.ErrorOnResolve = fmt.Errorf("there is no location to resolve")
		logger.Logf("dwarf", "%s: unresolvable: %v", varb.Name, varb.ErrorOnResolve)
		return location{}
	}

	loc, err := varb.loclist.resolve()
	if err != nil {
		varb.ErrorOnResolve = err
		logger.Logf("dwarf", "%s: unresolvable: %v", varb.Name, err)
		return location{}
	}

	return loc
}

// addVariableChildren populates the variable child array with SourceVariable
// instances that describe areas of memory related to the parent variable.
func (varb *SourceVariable) addVariableChildren(debug_loc *loclistSection) {
	if varb.Type.IsArray() {
		for i := 0; i < varb.Type.ElementCount; i++ {
			elem := &SourceVariable{
				Name:     fmt.Sprintf("%s[%d]", varb.Name, i),
				Type:     varb.Type.ElementType,
				DeclLine: varb.DeclLine,
			}
			elem.loclist = debug_loc.newLoclistJustContext(varb)

			if varb.loclist != nil {
				o := i
				elem.loclist.addOperator(func(_ *loclist) (location, error) {
					address, addressOk := varb.Address()
					address += uint64(o * varb.Type.ElementType.Size)
					value, ok := varb.loclist.coproc.CoProcRead32bit(uint32(address))
					return location{
						address:   address,
						addressOk: addressOk,
						value:     value,
						valueOk:   ok,
					}, nil
				})
			}

			varb.children = append(varb.children, elem)
			elem.addVariableChildren(debug_loc)
		}
	}

	if varb.Type.IsComposite() {
		var offset uint64
		for _, m := range varb.Type.Members {
			memb := &SourceVariable{
				Name:     m.Name,
				Type:     m.Type,
				DeclLine: varb.DeclLine,
			}
			memb.loclist = debug_loc.newLoclistJustContext(varb)

			if varb.loclist != nil {
				o := offset
				memb.loclist.addOperator(func(_ *loclist) (location, error) {
					address, addressOk := varb.Address()
					address += o
					value, ok := varb.loclist.coproc.CoProcRead32bit(uint32(address))
					return location{
						address:   address,
						addressOk: addressOk,
						value:     value,
						valueOk:   ok,
					}, nil
				})
			}

			varb.children = append(varb.children, memb)
			memb.addVariableChildren(debug_loc)

			offset += uint64(m.Type.Size)
		}
	}

	if varb.Type.IsPointer() {
		deref := &SourceVariable{
			Name:     fmt.Sprintf("*%s", varb.Name),
			Type:     varb.Type.PointerType,
			DeclLine: varb.DeclLine,
		}
		deref.loclist = debug_loc.newLoclistJustContext(varb)

		if varb.loclist != nil {
			deref.loclist.addOperator(func(_ *loclist) (location, error) {
				address, addressOk := varb.Value()
				value, ok := varb.loclist.coproc.CoProcRead32bit(address)
				return location{
					address:   uint64(address),
					addressOk: addressOk,
					value:     value,
					valueOk:   ok,
				}, nil
			})
		}

		varb.children = append(varb.children, deref)
		deref.addVariableChildren(debug_loc)
	}
}

// framebase implements the loclistFramebase interface
func (varb *SourceVariable) framebase() (uint64, error) {
	if varb.DeclLine == nil || varb.DeclLine.Function == nil {
		return 0, fmt.Errorf("no framebase")
	}

	fb, err := varb.DeclLine.Function.framebase()
	if err != nil {
		return 0, fmt.Errorf("framebase for function %s: %v", varb.DeclLine.Function.Name, err)
	}

	return fb, nil
}
