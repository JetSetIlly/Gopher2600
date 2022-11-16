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
	resolvedLast Resolved

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
func (varb *SourceVariable) Address() uint64 {
	varb.Cart.PushFunction(func() {
		varb.cachedResolve.Store(varb.resolve())
	})

	var r Resolved
	var ok bool
	if r, ok = varb.cachedResolve.Load().(Resolved); !ok {
		return 0
	}

	return r.address
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
	return varb.resolvedLast
}

// resolve address/value
func (varb *SourceVariable) resolve() Resolved {
	varb.resolvedLast = Resolved{}
	for i := range varb.resolver {
		varb.resolvedLast = varb.resolver[i](varb)
	}
	return varb.resolvedLast
}

// add another resolver to the desclaration expresion
func (varb *SourceVariable) addResolver(r resolver) {
	varb.resolver = append(varb.resolver, r)
}
