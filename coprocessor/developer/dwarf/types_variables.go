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
	"strings"
	"sync/atomic"
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

	hasConstantValue bool
	constantValue    uint32

	// location list resolves a Location. may be nil which indicates that the
	// variable can never be located
	loclist *loclist

	// if Error is not nil then an error was enountered during a resolve()
	// sequence. the error will be logged when the field is first set to true
	Error error

	// most recent resolved value retrieved from emulation
	cachedLocation atomic.Value // loclistResult

	// child variables of this variable. this includes array elements, struct
	// members and dereferenced variables
	children []*SourceVariable
}

func (varb *SourceVariable) String() string {
	if !varb.IsValid() {
		return fmt.Sprintf("%s %s is not locatable", varb.Type.Name, varb.Name)
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("%s %s", varb.Type.Name, varb.Name))
	if varb.Error != nil {
		s.WriteString(" has error")
	} else if varb.Type.IsArray() {
		s.WriteString(" is array")
	} else if varb.Type.IsComposite() {
		s.WriteString(" is composite")
	} else if varb.Type.IsPointer() {
		s.WriteString(" = ")
		s.WriteString(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
	} else {
		s.WriteString(" = ")
		if varb.Type.Conversion != nil {
			c, _ := varb.Type.Conversion(varb.Value())
			s.WriteString(c)
		} else {
			s.WriteString(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
		}
	}
	return s.String()
}

// Address returns the location in memory of the variable referred to by
// SourceVariable
func (varb *SourceVariable) Address() (uint64, bool) {
	var r loclistResult
	var ok bool
	if r, ok = varb.cachedLocation.Load().(loclistResult); !ok {
		return 0, false
	}
	return r.address, r.hasAddress
}

// Value returns the current value of a SourceVariable
func (varb *SourceVariable) Value() uint32 {
	if varb.hasConstantValue {
		return varb.constantValue
	}

	var r loclistResult
	var ok bool
	if r, ok = varb.cachedLocation.Load().(loclistResult); !ok {
		return 0
	}
	return r.value & varb.Type.Mask()
}

// hasPieces returns the number of pieces, if any, in the resolved variable.
func (varb *SourceVariable) hasPieces() int {
	var r loclistResult
	var ok bool
	if r, ok = varb.cachedLocation.Load().(loclistResult); !ok {
		return 0
	}
	return len(r.pieces)
}

// piece returns a loclistPiece at the index specified. returns true if
// loclistPiece is value and false if not
//
// use this and hasPieces when resolving members of a composite variable
// (structs, etc.)
func (varb *SourceVariable) piece(idx int) (loclistPiece, bool) {
	var r loclistResult
	var ok bool
	if r, ok = varb.cachedLocation.Load().(loclistResult); !ok {
		return loclistPiece{}, false
	}
	if idx >= len(r.pieces) {
		return loclistPiece{}, false
	}
	return r.pieces[idx], true
}

// IsValid returns true if the variable has a valid or resolvable value
func (varb *SourceVariable) IsValid() bool {
	return varb.loclist != nil || varb.hasConstantValue
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
	varb.cachedLocation.Store(varb.resolve(nil))
	for _, c := range varb.children {
		c.Update()
	}
}

// WriteDerivation outputs the derivation of a varibale to the io.Writer. If the
// derivation encounters an error the error is returned
//
// Note that the basic information about the variable is not output by this
// function. The String() function provides that information
func (varb *SourceVariable) WriteDerivation(derive io.Writer) error {
	if derive == nil {
		return nil
	}

	if varb.hasConstantValue {
		derive.Write(fmt.Appendf(nil, "constant value %08x", varb.constantValue))
	}

	if varb.loclist == nil {
		return nil
	}
	varb.resolve(derive)
	return varb.Error
}

// resolve address/value
func (varb *SourceVariable) resolve(derive io.Writer) loclistResult {
	if varb.hasConstantValue {
		return loclistResult{
			value: varb.constantValue,
		}
	}

	if varb.loclist == nil {
		varb.Error = fmt.Errorf("there is no location to resolve")
		return loclistResult{}
	}

	r, err := varb.loclist.resolve(derive)
	if err != nil {
		varb.Error = err
		return loclistResult{}
	}

	if r.hasAddress {
		v, ok := varb.loclist.coproc.Peek(uint32(r.address))
		if !ok {
			varb.Error = fmt.Errorf("error resolving address %08x", r.address)
			return loclistResult{}
		}
		r.value = v
	}

	return r
}

// addVariableChildren populates the variable child array with SourceVariable
// instances that describe areas of memory related to the parent variable.
func (varb *SourceVariable) addVariableChildren(debug_loc *loclistDecoder) {
	if varb.Type.IsArray() {
		for i := 0; i < varb.Type.ElementCount; i++ {
			elem := &SourceVariable{
				Name:     fmt.Sprintf("%s[%d]", varb.Name, i),
				Type:     varb.Type.ElementType,
				DeclLine: varb.DeclLine,
			}
			elem.loclist = debug_loc.newLoclistJustFramebase(varb)

			if varb.loclist != nil {
				o := i
				elem.loclist.addOperator(loclistOperator{
					resolve: func(_ *loclist, _ io.Writer) (loclistStack, error) {
						address, ok := varb.Address()
						if !ok {
							return loclistStack{}, fmt.Errorf("no base address for array")
						}
						address += uint64(o * varb.Type.ElementType.Size)
						return loclistStack{
							class: stackClassSingleAddress,
							value: uint32(address),
						}, nil
					},
					operator: "array offset",
				})
			}

			varb.children = append(varb.children, elem)
			elem.addVariableChildren(debug_loc)
		}
	}

	if varb.Type.IsComposite() {
		for i, m := range varb.Type.Members {
			memb := &SourceVariable{
				Name:     m.Name,
				Type:     m.Type,
				DeclLine: varb.DeclLine,
			}
			memb.loclist = debug_loc.newLoclistJustFramebase(varb)

			if varb.loclist != nil {
				offset := m.resolve(nil).value
				idx := i
				memb.loclist.addOperator(loclistOperator{
					resolve: func(_ *loclist, _ io.Writer) (loclistStack, error) {
						np := varb.hasPieces()
						if np == 0 {
							address, ok := varb.Address()
							if !ok {
								return loclistStack{}, fmt.Errorf("no base address for composite variable")
							}
							address += uint64(offset)
							return loclistStack{
								class: stackClassSingleAddress,
								value: uint32(address),
							}, nil
						}

						p, ok := varb.piece(idx)
						if !ok {
							return loclistStack{}, fmt.Errorf("no piece information for this member")
						}

						if p.isAddress {
							return loclistStack{
								class: stackClassSingleAddress,
								value: p.value,
							}, nil
						}

						return loclistStack{
							class: stackClassIsValue,
							value: p.value,
						}, nil
					},
					operator: "member offset",
				})
			}

			varb.children = append(varb.children, memb)
			memb.addVariableChildren(debug_loc)
		}
	}

	if varb.Type.IsPointer() {
		deref := &SourceVariable{
			Name:     fmt.Sprintf("*%s", varb.Name),
			Type:     varb.Type.PointerType,
			DeclLine: varb.DeclLine,
		}
		deref.loclist = debug_loc.newLoclistJustFramebase(varb)

		deref.loclist.addOperator(loclistOperator{
			resolve: func(_ *loclist, _ io.Writer) (loclistStack, error) {
				return loclistStack{
					class: stackClassSingleAddress,
					value: varb.Value(),
				}, nil
			},
			operator: "pointer dereference",
		})

		varb.children = append(varb.children, deref)
		deref.addVariableChildren(debug_loc)
	}
}

// resolveFramebase implements the loclistResolver interface
func (varb *SourceVariable) resolveFramebase(derviation io.Writer) (uint64, error) {
	if varb.DeclLine == nil || varb.DeclLine.Function == nil {
		return 0, fmt.Errorf("no framebase")
	}

	fb, err := varb.DeclLine.Function.resolveFramebase(derviation)
	if err != nil {
		return 0, fmt.Errorf("framebase for %s: %w", varb.DeclLine.Function.Name, err)
	}

	return fb, nil
}
