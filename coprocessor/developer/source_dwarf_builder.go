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
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/logger"
)

// associates a compile unit with an individual entry. this is important
// because retreiving the file list for an entry depends very much on the
// compile unit - we need to make sure we're using the correct compile unit.
type buildEntry struct {
	compileUnit *dwarf.Entry
	entry       *dwarf.Entry

	// additional information. context sensitive according to the entry type
	information string
}

type build struct {
	dwrf *dwarf.Data

	// debug_loc is the .debug_loc ELF section. it's required by DWARF for
	// finding local variables in memory
	debug_loc *elf.Section

	subprograms        map[dwarf.Offset]buildEntry
	inlinedSubroutines map[dwarf.Offset]buildEntry
	baseTypes          map[dwarf.Offset]buildEntry

	typedefs         map[dwarf.Offset]buildEntry
	compositeTypes   map[dwarf.Offset]buildEntry
	compositeMembers map[dwarf.Offset]buildEntry
	arrayTypes       map[dwarf.Offset]buildEntry
	arraySubranges   map[dwarf.Offset]buildEntry

	variables map[dwarf.Offset]buildEntry
	pointers  map[dwarf.Offset]buildEntry
	consts    map[dwarf.Offset]buildEntry

	// the order in which we encountered the subprograms and inlined
	// subroutines is important
	order []dwarf.Offset
}

func newBuild(dwrf *dwarf.Data, debug_loc *elf.Section) (*build, error) {
	bld := &build{
		dwrf:               dwrf,
		debug_loc:          debug_loc,
		subprograms:        make(map[dwarf.Offset]buildEntry),
		inlinedSubroutines: make(map[dwarf.Offset]buildEntry),
		baseTypes:          make(map[dwarf.Offset]buildEntry),
		typedefs:           make(map[dwarf.Offset]buildEntry),
		compositeTypes:     make(map[dwarf.Offset]buildEntry),
		compositeMembers:   make(map[dwarf.Offset]buildEntry),
		arrayTypes:         make(map[dwarf.Offset]buildEntry),
		arraySubranges:     make(map[dwarf.Offset]buildEntry),
		variables:          make(map[dwarf.Offset]buildEntry),
		pointers:           make(map[dwarf.Offset]buildEntry),
		consts:             make(map[dwarf.Offset]buildEntry),
		order:              make([]dwarf.Offset, 0, 100),
	}

	var compileUnit *dwarf.Entry

	r := bld.dwrf.Reader()
	for {
		entry, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break // for loop
			}
			return nil, err
		}
		if entry == nil {
			break // for loop
		}
		if entry.Offset == 0 {
			continue // for loop
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			compileUnit = entry

		case dwarf.TagInlinedSubroutine:
			if compileUnit == nil {
				return nil, curated.Errorf("found inlined subroutine tag without compile unit")
			} else {
				bld.inlinedSubroutines[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagSubprogram:
			if compileUnit == nil {
				return nil, curated.Errorf("found subprogram tag without compile unit")
			} else {
				bld.subprograms[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagLexDwarfBlock:
			// do nothing with lexical blocks

		case dwarf.TagTypedef:
			if compileUnit == nil {
				return nil, curated.Errorf("found base/pointer type tag without compile unit")
			} else {
				bld.typedefs[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagBaseType:
			if compileUnit == nil {
				return nil, curated.Errorf("found base/pointer type tag without compile unit")
			} else {
				bld.baseTypes[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagUnionType:
			if compileUnit == nil {
				return nil, curated.Errorf("found union type tag without compile unit")
			} else {
				bld.compositeTypes[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
					information: "union",
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagStructType:
			if compileUnit == nil {
				return nil, curated.Errorf("found struct type tag without compile unit")
			} else {
				bld.compositeTypes[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
					information: "struct",
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagMember:
			if compileUnit == nil {
				return nil, curated.Errorf("found member tag without compile unit")
			} else {
				bld.compositeMembers[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagArrayType:
			if compileUnit == nil {
				return nil, curated.Errorf("found array type tag without compile unit")
			} else {
				bld.arrayTypes[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagSubrangeType:
			if compileUnit == nil {
				return nil, curated.Errorf("found subrange type tag without compile unit")
			} else {
				bld.arraySubranges[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagVariable:
			if compileUnit == nil {
				return nil, curated.Errorf("found variable tag without compile unit")
			} else {
				bld.variables[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagFormalParameter:
			if compileUnit == nil {
				return nil, curated.Errorf("found formal parameter tag without compile unit")
			} else {
				bld.variables[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagPointerType:
			if compileUnit == nil {
				return nil, curated.Errorf("found pointer type tag without compile unit")
			} else {
				bld.pointers[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}

		case dwarf.TagConstType:
			if compileUnit == nil {
				return nil, curated.Errorf("found const type tag without compile unit")
			} else {
				bld.consts[entry.Offset] = buildEntry{
					compileUnit: compileUnit,
					entry:       entry,
				}
				bld.order = append(bld.order, entry.Offset)
			}
		}
	}

	return bld, nil
}

// buildTypes creates the types necessary to build variable information. in
// parituclar allocation of members to the "parent" composite type
func (bld *build) buildTypes(src *Source) error {
	resolveTypeDefs := func() error {
		for _, v := range bld.typedefs {
			baseType, err := bld.resolveType(v, src)
			if err != nil {
				return err
			}
			if baseType == nil {
				continue
			}

			// make a copy of the named type
			typ := func(btyp *SourceType) *SourceType {
				typ := *btyp
				return &typ
			}(baseType)

			// override the name field
			fld := v.entry.AttrField(dwarf.AttrName)
			if fld == nil {
				continue
			}
			typ.Name = fld.Val.(string)

			src.types[v.entry.Offset] = typ
		}

		return nil
	}

	// basic types first because everything else is built on basic types
	for _, v := range bld.baseTypes {
		var typ SourceType

		fld := v.entry.AttrField(dwarf.AttrName)
		if fld == nil {
			continue
		}
		typ.Name = fld.Val.(string)

		fld = v.entry.AttrField(dwarf.AttrByteSize)
		if fld == nil {
			continue
		}
		typ.Size = int(fld.Val.(int64))

		src.types[v.entry.Offset] = &typ
	}

	// typedefs of basic types
	err := resolveTypeDefs()
	if err != nil {
		return err
	}

	// two passes over pointer types, const types, and composite types
	for pass := 0; pass < 2; pass++ {
		// pointer types
		for _, v := range bld.pointers {
			var typ SourceType

			typ.PointerType, err = bld.resolveType(v, src)
			if err != nil {
				return err
			}
			if typ.PointerType == nil {
				continue
			}

			typ.Name = fmt.Sprintf("%s *", typ.PointerType.Name)

			fld := v.entry.AttrField(dwarf.AttrByteSize)
			if fld == nil {
				continue
			}
			typ.Size = int(fld.Val.(int64))

			src.types[v.entry.Offset] = &typ
		}

		// typedefs of pointer types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}

		// resolve composite types
		for _, v := range bld.compositeTypes {
			var typ SourceType
			var name string

			fld := v.entry.AttrField(dwarf.AttrName)
			if fld == nil {
				// allow anonymous structures. we sometimes see this when
				// structs are defined with typedef
				name = fmt.Sprintf("%x", v.entry.Offset)
			} else {
				name = fld.Val.(string)
			}

			fld = v.entry.AttrField(dwarf.AttrByteSize)
			if fld == nil {
				continue
			}
			typ.Size = int(fld.Val.(int64))

			src.types[v.entry.Offset] = &typ

			// the name we store in the type is annotated with the composite
			// category.
			//
			// this may be language sensitive but we're assuming the use of C for now
			typ.Name = fmt.Sprintf("%s %s", v.information, name)
		}

		// allocate members to composite types
		var composite *SourceType
		for _, off := range bld.order {
			if v, ok := bld.compositeTypes[off]; ok {
				composite = src.types[v.entry.Offset]
			} else if v, ok := bld.compositeMembers[off]; ok {
				if composite == nil {
					// found a member without first finding a composite type. this
					// shouldn't happen
					continue
				}

				if v.entry.Offset == 0x274 {
				}

				// members are basically like variables but with special address
				// handling
				memb, err := bld.resolveVariableDeclaration(v, src)
				if err != nil {
					return err
				}
				if memb == nil {
					continue
				}

				// look for data member location field. if it's not present
				// then it doesn't matter, the member address will be kept at
				// zero and will still be considered an offset address. absence
				// of the data member location field is the case with union
				// types
				fld := v.entry.AttrField(dwarf.AttrDataMemberLoc)
				if fld != nil {
					switch fld.Class {
					case dwarf.ClassConstant:
						memb.loclist = newLoclistJustContext(memb)
						memb.loclist.addOperator(func(loc *loclist) (location, error) {
							address := uint64(fld.Val.(int64))
							value, ok := loc.ctx.coproc().CoProcRead32bit(uint32(address))
							return location{
								address: address,
								value:   value,
								valueOk: ok,
							}, nil
						})
					case dwarf.ClassExprLoc:
						memb.loclist = newLoclistJustContext(memb)
						r, _ := decodeDWARFoperation(fld.Val.([]uint8), 0)
						memb.loclist.addOperator(r)
					default:
						continue
					}
				}

				composite.Members = append(composite.Members, memb)
			} else {
				composite = nil
			}
		}

		// remove any composites that have no members
		for off := range bld.compositeTypes {
			if src.types[off] != nil && len(src.types[off].Members) == 0 {
				delete(src.types, off)
			}
		}

		// typedefs of composite types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}

		// build array types
		var arrayBaseType *SourceType
		var baseTypeOffset dwarf.Offset
		for _, off := range bld.order {
			if v, ok := bld.arrayTypes[off]; ok {
				var err error
				arrayBaseType, err = bld.resolveType(v, src)
				if err != nil {
					return err
				}
				baseTypeOffset = v.entry.Offset
			} else if v, ok := bld.arraySubranges[off]; ok {
				if arrayBaseType == nil {
					// found a subrange without first finding an array type. this
					// shouldn't happen
					continue
				}

				fld := v.entry.AttrField(dwarf.AttrUpperBound)
				if fld == nil {
					continue
				}
				num := fld.Val.(int64) + 1

				src.types[baseTypeOffset] = &SourceType{
					Name:         fmt.Sprintf("[%d] %s", num, arrayBaseType.Name),
					Size:         arrayBaseType.Size * int(num),
					ElementType:  arrayBaseType,
					ElementCount: int(num),
				}
			} else {
				arrayBaseType = nil
			}
		}

		// typedefs of array types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}

		// const types
		for _, v := range bld.consts {
			baseType, err := bld.resolveType(v, src)
			if err != nil {
				return err
			}
			if baseType == nil {
				continue
			}

			typ := *baseType
			typ.Constant = true
			typ.Name = fmt.Sprintf("const %s", baseType.Name)

			src.types[v.entry.Offset] = &typ
		}

		// typedefs of const types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}
	}

	return nil
}

func (bld *build) resolveType(v buildEntry, src *Source) (*SourceType, error) {
	fld := v.entry.AttrField(dwarf.AttrType)
	if fld == nil {
		return nil, nil
	}

	typ, ok := src.types[fld.Val.(dwarf.Offset)]
	if !ok {
		return nil, nil
	}

	return typ, nil
}

func (bld *build) resolveVariableDeclaration(v buildEntry, src *Source) (*SourceVariable, error) {
	resolveSpec := func(v buildEntry) (*SourceVariable, error) {
		var varb SourceVariable

		// variable name
		fld := v.entry.AttrField(dwarf.AttrName)
		if fld == nil {
			return nil, nil
		}
		varb.Name = fld.Val.(string)

		// variable type
		var err error
		varb.Type, err = bld.resolveType(v, src)
		if err != nil {
			return nil, err
		}
		if varb.Type == nil {
			return nil, nil
		}

		return &varb, nil
	}

	var varb SourceVariable

	// check for specification field. if it is present we resolve the
	// specification using with the DWARF entry indicated by the field.
	// otherwise we resolve using the current entry
	fld := v.entry.AttrField(dwarf.AttrSpecification)
	if fld != nil {
		var ok bool

		spec, ok := bld.variables[fld.Val.(dwarf.Offset)]
		if !ok {
			return nil, nil
		}

		s, err := resolveSpec(spec)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, nil
		}
		varb.Name = s.Name
		varb.Type = s.Type
	} else {
		s, err := resolveSpec(v)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, nil
		}
		varb.Name = s.Name
		varb.Type = s.Type
	}

	// variable location in the source
	fld = v.entry.AttrField(dwarf.AttrDeclFile)
	if fld == nil {
		return nil, nil
	}
	declFile := fld.Val.(int64)

	fld = v.entry.AttrField(dwarf.AttrDeclLine)
	if fld == nil {
		return nil, nil
	}
	declLine := fld.Val.(int64)

	lr, err := bld.dwrf.LineReader(v.compileUnit)
	if err != nil {
		return nil, err
	}
	files := lr.Files()

	file := src.Files[files[declFile].Name]
	if file == nil {
		return nil, nil
	}
	varb.DeclLine = file.Content.Lines[declLine-1]

	return &varb, nil
}

// buildVariables populates variables map in the *Source tree
func (bld *build) buildVariables(src *Source, origin uint64) error {
	for _, v := range bld.variables {
		// resolve name and type of variable
		var varb *SourceVariable
		var err error

		// check for abstract origin field. if it is present we resolve the
		// declartion using the DWARF entry indicated by the field. otherwise
		// we resolve using the current entry
		fld := v.entry.AttrField(dwarf.AttrAbstractOrigin)
		if fld != nil {
			abstract, ok := bld.variables[fld.Val.(dwarf.Offset)]
			if !ok {
				return curated.Errorf("found concrete variable without abstract")
			}

			varb, err = bld.resolveVariableDeclaration(abstract, src)
			if err != nil {
				return err
			}
		} else {
			varb, err = bld.resolveVariableDeclaration(v, src)
			if err != nil {
				return err
			}
		}

		// nothing found when resolving the declaration
		if varb == nil {
			continue // for loop
		}

		// do not include variables of constant type (pointers to constant type are fine)
		if varb.Type.Constant {
			continue // for loop
		}

		// do not include variables of array type where the elements are of
		// constant type
		if varb.Type.ElementType != nil && varb.Type.Constant {
			continue // for loop
		}

		// point variable to CartCoProcDeveloper
		varb.Cart = src.cart

		// variable actually exists in memory if it has a location attribute
		fld = v.entry.AttrField(dwarf.AttrLocation)
		if fld != nil {
			switch fld.Class {
			case dwarf.ClassLocListPtr:
				var err error
				err = newLoclist(varb, bld.debug_loc, fld.Val.(int64), func(start, end uint64, loc *loclist) {
					varb.loclist = loc
					local := &SourceVariableLocal{
						SourceVariable:  varb,
						resolvableStart: start,
						resolvableEnd:   end,
					}

					src.locals = append(src.locals, local)
					src.SortedLocals.Locals = append(src.SortedLocals.Locals, local)
				})
				if err != nil {
					logger.Logf("dwarf", "%s: %v", varb.Name, err)
				}

			case dwarf.ClassExprLoc:
				// Single location description "They are sufficient for describing the location of any object
				// as long as its lifetime is either static or the same as the lexical block that owns it,
				// and it does not move during its lifetime"
				// page 26 of "DWARF4 Standard"

				// set address resolve function
				r, n := decodeDWARFoperation(fld.Val.([]uint8), origin)
				if n != 0 {
					varb.loclist = newLoclistJustContext(varb)
					varb.loclist.addOperator(r)

					// determine highest address occupied by any variable in the program
					hiAddress := varb.resolve().address + uint64(varb.Type.Size)
					if hiAddress > src.VariableMemtop {
						src.VariableMemtop = hiAddress
					}

					// add variable to list of global variables if there is no
					// parent function otherwise we treat the variable as a
					// local variable
					if varb.DeclLine.Function.Name == stubIndicator {
						// list of global variables for all compile units
						src.globals[varb.Name] = varb
						src.GlobalsByAddress[varb.resolve().address] = varb
						src.SortedGlobals.Variables = append(src.SortedGlobals.Variables, varb)

						// note that the file has at least one global variables
						varb.DeclLine.File.HasGlobals = true
					} else {
						varb.loclist = newLoclistJustContext(varb)
						varb.loclist.addOperator(r)

						local := &SourceVariableLocal{
							SourceVariable: varb,
						}

						src.locals = append(src.locals, local)
						src.SortedLocals.Locals = append(src.SortedLocals.Locals, local)
					}
				}
			}

		} else {
			// add local variable without any loclist. this enables the user to
			// see that the variable exists for a function even though we don't
			// know how to locate it in memory
			if varb.DeclLine.Function.Name != stubIndicator {
				local := &SourceVariableLocal{
					SourceVariable: varb,
				}
				src.locals = append(src.locals, local)
				src.SortedLocals.Locals = append(src.SortedLocals.Locals, local)
			}
		}

		// add children (array elements etc.)
		varb.addVariableChildren()
	}

	return nil
}

type foundFunction struct {
	filename string
	linenum  int64
	name     string

	// not all functions have a framebase
	//
	// the loclist will have been built without a context it must be manually
	// added before use
	framebase *loclist
}

func (bld *build) findFunction(src *Source, addr uint64) (*foundFunction, error) {
	var ret *foundFunction

	resolveFramebase := func(b buildEntry) (*loclist, error) {
		var framebase *loclist

		fld := b.entry.AttrField(dwarf.AttrFrameBase)
		if fld != nil {
			switch fld.Class {
			case dwarf.ClassExprLoc:
				var err error
				framebase, err = newLoclistFromSingleOperator(nil, fld.Val.([]uint8))
				if err != nil {
					return nil, err
				}
			case dwarf.ClassLocListPtr:
				err := newLoclist(nil, bld.debug_loc, fld.Val.(int64), func(_, _ uint64, loc *loclist) {
					framebase = loc
				})
				if err != nil {
					return nil, err
				}
			}
		}

		return framebase, nil
	}

	resolve := func(b buildEntry) (*foundFunction, error) {
		lr, err := bld.dwrf.LineReader(b.compileUnit)
		if err != nil {
			return nil, err
		}
		files := lr.Files()

		// name of entry
		fld := b.entry.AttrField(dwarf.AttrName)
		if fld == nil {
			return nil, nil
		}
		name := fld.Val.(string)

		// declaration file
		fld = b.entry.AttrField(dwarf.AttrDeclFile)
		if fld == nil {
			return nil, nil
		}
		filenum := fld.Val.(int64)

		// declaration line
		fld = b.entry.AttrField(dwarf.AttrDeclLine)
		if fld == nil {
			return nil, nil
		}
		linenum := fld.Val.(int64)

		// framebase. non-abstract functions will not have a framebase
		// attribute. for those functions we can resolve it later
		framebase, err := resolveFramebase(b)
		if err != nil {
			logger.Logf("dwarf", "framebase for %s will be unreliable: %v", name, err)
		}

		return &foundFunction{
			filename:  files[filenum].Name,
			linenum:   linenum,
			name:      name,
			framebase: framebase,
		}, nil
	}

	for _, off := range bld.order {
		var low uint64
		var high uint64

		if subp, ok := bld.subprograms[off]; ok {
			entry := subp.entry

			// check address against low/high fields. compare to
			// InlinedSubroutines where address range can be given by either
			// low/high fields OR a Range field. for Subprograms, there is
			// never a Range field.

			fld := entry.AttrField(dwarf.AttrLowpc)
			if fld == nil {
				// it is possible for Subprograms to have no address fields.
				// the Subprograms are abstract and will be referred to by
				// either concrete Subprograms or concrete InlinedSubroutines
				continue // for loop
			}
			low = uint64(fld.Val.(uint64))

			fld = entry.AttrField(dwarf.AttrHighpc)
			if fld == nil {
				return nil, curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine: %08x", addr)
			}

			switch fld.Class {
			case dwarf.ClassConstant:
				// dwarf-4
				high = low + uint64(fld.Val.(int64))
			case dwarf.ClassAddress:
				// dwarf-2
				high = uint64(fld.Val.(uint64))
			default:
				return nil, curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine: %08x", addr)
			}

			if addr < low || addr >= high {
				continue // for loop
			}

			fld = entry.AttrField(dwarf.AttrAbstractOrigin)
			if fld != nil {
				abstract, ok := bld.subprograms[fld.Val.(dwarf.Offset)]
				if !ok {
					return nil, curated.Errorf("found inlined subroutine without abstract: %08x", addr)
				}

				r, err := resolve(abstract)
				if err != nil {
					return nil, err
				}
				if r != nil {
					ret = r
				}

				// try to acquire framebase for concrete subroutine. we don't expect
				// for the framebase to have been found already but we'll check it
				// to make sure in any case
				if ret.framebase == nil {
					framebase, err := resolveFramebase(subp)
					if err != nil {
						logger.Logf("dwarf", "framebase for %s will be unreliable: %v", ret.name, err)
					}
					ret.framebase = framebase
				} else {
					logger.Logf("dwarf", "%s: concrete defintion for abstract function already has a framebase defintion!?", ret.name)
				}
			} else {
				r, err := resolve(subp)
				if err != nil {
					return nil, err
				}
				if r != nil {
					ret = r
				}
			}

		} else if inl, ok := bld.inlinedSubroutines[off]; ok {
			entry := inl.entry
			fld := entry.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				low = uint64(fld.Val.(uint64))

				// high PC
				fld = entry.AttrField(dwarf.AttrHighpc)
				if fld == nil {
					return nil, curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine: %08x", addr)
				}

				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					high = low + uint64(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					high = uint64(fld.Val.(uint64))
				default:
					return nil, curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine: %08x", addr)
				}

				if addr < low || addr >= high {
					continue // for loop
				}
			} else {
				fld = entry.AttrField(dwarf.AttrRanges)
				if fld == nil {
					continue // for loop
				}

				rngs, err := bld.dwrf.Ranges(entry)
				if err != nil {
					return nil, err
				}

				match := false
				for _, r := range rngs {
					if addr >= r[0] && addr < r[1] {
						low = r[0]
						high = r[1]
						match = true
						break
					}
				}
				if !match {
					continue // for loop
				}
			}

			fld = entry.AttrField(dwarf.AttrAbstractOrigin)
			if fld == nil {
				return nil, curated.Errorf("missing AttrAbstractOrigin: %08x", addr)
			}

			abstract, ok := bld.subprograms[fld.Val.(dwarf.Offset)]
			if !ok {
				return nil, curated.Errorf("found inlined subroutine without abstract: %08x", addr)
			}

			r, err := resolve(abstract)
			if err != nil {
				return nil, err
			}
			if r != nil {
				ret = r
			}

			// try to acquire framebase for inline subroutine. we don't expect
			// for the framebase to have been found already but we'll check it
			// to make sure in any case
			if ret.framebase == nil {
				framebase, err := resolveFramebase(inl)
				if err != nil {
					logger.Logf("dwarf", "framebase for %s will be unreliable: %v", ret.name, err)
				}
				ret.framebase = framebase
			} else {
				logger.Logf("dwarf", "%s: concrete defintion for abstract function already has a framebase defintion!?", ret.name)
			}
		}
	}

	return ret, nil
}
