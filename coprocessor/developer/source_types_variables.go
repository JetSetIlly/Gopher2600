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
)

// SourceVariable is a single local variable identified by the DWARF data.
type SourceVariableLocal struct {
	*SourceVariable

	// the address range for which the variable is valid
	StartAddress uint64
	EndAddress   uint64
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

	// DWARF expression resolver. see resolve(), addResolver() and
	// lastResolved(). external packages use Address() or Value()
	resolver     []resolver
	resolveStack []Resolved

	// most recent resolved value retrieved from emulation
	cachedResolve atomic.Value // Resolved

	// origin address of variable
	origin uint64

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
		varb.cachedResolve.Store(varb.resolve())
	})

	var r Resolved
	var ok bool
	if r, ok = varb.cachedResolve.Load().(Resolved); !ok {
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
		varb.cachedResolve.Store(varb.resolve())
	})

	var r Resolved
	var ok bool
	if r, ok = varb.cachedResolve.Load().(Resolved); !ok {
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
func (varb *SourceVariable) framebase() uint64 {
	if varb.DeclLine == nil || varb.DeclLine.Function == nil || varb.DeclLine.Function.frameBase == nil {
		return 0
	}
	return varb.DeclLine.Function.frameBase(varb).address
}

// lastResolved implements the resolver interface
func (varb *SourceVariable) lastResolved() Resolved {
	if len(varb.resolveStack) == 0 {
		return Resolved{}
	}
	return varb.resolveStack[len(varb.resolveStack)-1]
}

func (varb *SourceVariable) pop() (Resolved, bool) {
	l := len(varb.resolveStack)
	if l == 0 {
		return Resolved{}, false
	}
	r := varb.resolveStack[l-1]
	varb.resolveStack = varb.resolveStack[:l-1]
	return r, true
}

// resolve address/value
func (varb *SourceVariable) resolve() Resolved {
	varb.resolveStack = varb.resolveStack[:0]
	for i := range varb.resolver {
		r := varb.resolver[i](varb)
		if r.addressOk || r.valueOk {
			varb.resolveStack = append(varb.resolveStack, r)
		}
	}

	if len(varb.resolveStack) == 0 {
		return Resolved{}
	}

	return varb.resolveStack[len(varb.resolveStack)-1]
}

// add another resolver to the desclaration expresion
func (varb *SourceVariable) addResolver(r resolver) {
	varb.resolver = append(varb.resolver, r)
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

			o := i
			elem.addResolver(func(r resolveCoproc) Resolved {
				address := varb.lastResolved().address + uint64(o*varb.Type.ElementType.Size)
				value, ok := r.coproc().CoProcRead32bit(uint32(address))
				return Resolved{
					address:   address,
					addressOk: true,
					value:     value,
					valueOk:   ok,
				}
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

			o := offset
			memb.addResolver(func(r resolveCoproc) Resolved {
				address := varb.lastResolved().address + o
				value, ok := r.coproc().CoProcRead32bit(uint32(address))
				return Resolved{
					address:   address,
					addressOk: true,
					value:     value,
					valueOk:   ok,
				}
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
		deref.addResolver(func(r resolveCoproc) Resolved {
			a := varb.lastResolved().value
			address := uint64(a)
			value, ok := r.coproc().CoProcRead32bit(uint32(address))
			return Resolved{
				address:   address,
				addressOk: true,
				value:     value,
				valueOk:   ok,
			}
		})
		varb.children = append(varb.children, deref)
		deref.addVariableChildren()
	}
}
