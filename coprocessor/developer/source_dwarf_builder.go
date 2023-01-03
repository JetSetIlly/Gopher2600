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
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/logger"
)

// the build struct is only used during construction of the debugging
// information to help the NewSource() function
type build struct {
	dwrf *dwarf.Data

	// ELF sections that help DWARF locate local variables in memory
	debug_loc   *loclistSection
	debug_frame *frameSection

	// the order in which we encountered the subprograms and inlined
	// subroutines is important
	order []*dwarf.Entry

	// all entries in the DWARF data
	entries map[dwarf.Offset]*dwarf.Entry

	// the parent compile unit for every dwarf offset. we record this because
	// sometimes a DWARF entry will reference another DWARF entry in a
	// different compile unit. this is important because acquiring a line
	// reader depends on the compile unit and the reason we need a line reader
	// is in order to match an AttrDeclFile with a file name
	compileUnits map[dwarf.Offset]*dwarf.Entry
}

func newBuild(dwrf *dwarf.Data, debug_loc *loclistSection, debug_frame *frameSection) (*build, error) {
	bld := &build{
		dwrf:         dwrf,
		debug_loc:    debug_loc,
		debug_frame:  debug_frame,
		entries:      make(map[dwarf.Offset]*dwarf.Entry),
		compileUnits: make(map[dwarf.Offset]*dwarf.Entry),
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

		bld.order = append(bld.order, entry)
		bld.entries[entry.Offset] = entry
		bld.compileUnits[entry.Offset] = compileUnit

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			compileUnit = entry
		}
	}

	return bld, nil
}

// buildTypes creates the types necessary to build variable information. in
// parituclar allocation of members to the "parent" composite type
func (bld *build) buildTypes(src *Source) error {
	resolveTypeDefs := func() error {
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagTypedef:
				baseType, err := bld.resolveType(e, src)
				if err != nil {
					return err
				}
				if baseType == nil {
					continue
				}

				// make a copy of the named type
				typ := func(b *SourceType) *SourceType {
					typ := *b
					return &typ
				}(baseType)

				// override the name field
				fld := e.AttrField(dwarf.AttrName)
				if fld == nil {
					continue
				}
				typ.Name = fld.Val.(string)

				src.types[e.Offset] = typ
			}
		}

		return nil
	}

	// basic types first because everything else is built on basic types
	for _, e := range bld.order {
		switch e.Tag {
		case dwarf.TagBaseType:
			var typ SourceType

			fld := e.AttrField(dwarf.AttrName)
			if fld == nil {
				continue
			}
			typ.Name = fld.Val.(string)

			fld = e.AttrField(dwarf.AttrByteSize)
			if fld == nil {
				continue
			}
			typ.Size = int(fld.Val.(int64))

			src.types[e.Offset] = &typ
		}
	}

	// typedefs of basic types
	err := resolveTypeDefs()
	if err != nil {
		return err
	}

	// two passes over pointer types, const types, and composite types
	for pass := 0; pass < 2; pass++ {
		// pointer types
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagPointerType:
				var typ SourceType

				typ.PointerType, err = bld.resolveType(e, src)
				if err != nil {
					return err
				}
				if typ.PointerType == nil {
					continue
				}

				typ.Name = fmt.Sprintf("%s *", typ.PointerType.Name)

				fld := e.AttrField(dwarf.AttrByteSize)
				if fld == nil {
					continue
				}
				typ.Size = int(fld.Val.(int64))

				src.types[e.Offset] = &typ
			}
		}

		// typedefs of pointer types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}

		// resolve composite types
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagUnionType:
				fallthrough
			case dwarf.TagStructType:
				var typ SourceType
				var name string

				fld := e.AttrField(dwarf.AttrName)
				if fld == nil {
					// allow anonymous structures. we sometimes see this when
					// structs are defined with typedef
					name = fmt.Sprintf("%x", e.Offset)
				} else {
					name = fld.Val.(string)
				}

				fld = e.AttrField(dwarf.AttrByteSize)
				if fld == nil {
					continue
				}
				typ.Size = int(fld.Val.(int64))

				src.types[e.Offset] = &typ

				// the name we store in the type is annotated with the composite category
				switch e.Tag {
				case dwarf.TagUnionType:
					typ.Name = fmt.Sprintf("union %s", name)
				case dwarf.TagStructType:
					typ.Name = fmt.Sprintf("struct %s", name)
				default:
					typ.Name = fmt.Sprintf("%s", name)
				}
			}
		}

		// allocate members to composite types
		var composite *SourceType
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagUnionType:
				fallthrough
			case dwarf.TagStructType:
				composite = src.types[e.Offset]

			case dwarf.TagMember:
				if composite == nil {
					// found a member without first finding a composite type. this
					// shouldn't happen
					continue
				}

				// members are basically like variables but with special address
				// handling
				memb, err := bld.resolveVariableDeclaration(e, src)
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
				fld := e.AttrField(dwarf.AttrDataMemberLoc)
				if fld != nil {
					switch fld.Class {
					case dwarf.ClassConstant:
						memb.loclist = bld.debug_loc.newLoclistJustContext(memb)
						memb.loclist.addOperator(func(loc *loclist) (location, error) {
							address := uint64(fld.Val.(int64))
							value, ok := bld.debug_loc.coproc.CoProcRead32bit(uint32(address))
							return location{
								address: address,
								value:   value,
								valueOk: ok,
							}, nil
						})
					case dwarf.ClassExprLoc:
						memb.loclist = bld.debug_loc.newLoclistJustContext(memb)
						r, _ := bld.debug_loc.decodeLoclistOperation(fld.Val.([]uint8))
						memb.loclist.addOperator(r)
					default:
						continue
					}
				}

				composite.Members = append(composite.Members, memb)

			default:
				composite = nil
			}
		}

		// remove any composites that have no members
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagUnionType:
				fallthrough
			case dwarf.TagStructType:
				if src.types[e.Offset] != nil && len(src.types[e.Offset].Members) == 0 {
					delete(src.types, e.Offset)
				}
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
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagArrayType:
				var err error
				arrayBaseType, err = bld.resolveType(e, src)
				if err != nil {
					return err
				}
				baseTypeOffset = e.Offset

			case dwarf.TagSubrangeType:
				if arrayBaseType == nil {
					// found a subrange without first finding an array type. this
					// shouldn't happen
					continue
				}

				fld := e.AttrField(dwarf.AttrUpperBound)
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

			default:
				arrayBaseType = nil
			}
		}

		// typedefs of array types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}

		// const types
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagConstType:
				baseType, err := bld.resolveType(e, src)
				if err != nil {
					return err
				}
				if baseType == nil {
					continue
				}

				typ := *baseType
				typ.Constant = true
				typ.Name = fmt.Sprintf("const %s", baseType.Name)

				src.types[e.Offset] = &typ
			}
		}

		// typedefs of const types
		err = resolveTypeDefs()
		if err != nil {
			return err
		}
	}

	return nil
}

func (bld *build) resolveType(v *dwarf.Entry, src *Source) (*SourceType, error) {
	fld := v.AttrField(dwarf.AttrType)
	if fld == nil {
		return nil, nil
	}

	typ, ok := src.types[fld.Val.(dwarf.Offset)]
	if !ok {
		return nil, nil
	}

	return typ, nil
}

func (bld *build) resolveVariableDeclaration(v *dwarf.Entry, src *Source) (*SourceVariable, error) {
	resolveSpec := func(v *dwarf.Entry) (*SourceVariable, error) {
		var varb SourceVariable

		// variable name
		fld := v.AttrField(dwarf.AttrName)
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
	fld := v.AttrField(dwarf.AttrSpecification)
	if fld != nil {
		var ok bool

		spec, ok := bld.entries[fld.Val.(dwarf.Offset)]
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
	fld = v.AttrField(dwarf.AttrDeclFile)
	if fld == nil {
		return nil, nil
	}
	declFile := fld.Val.(int64)

	fld = v.AttrField(dwarf.AttrDeclLine)
	if fld == nil {
		return nil, nil
	}
	declLine := fld.Val.(int64)

	lr, err := bld.dwrf.LineReader(bld.compileUnits[v.Offset])
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

// buildVariables populates variables map in the *Source tree. local variables
// will need to be relocated for relocatable ELF files
func (bld *build) buildVariables(src *Source, origin uint64) error {
	// keep track of the lexical range as we walk through the DWARF data in
	// order. if we need to add a variable to the list of locals and the DWARF
	// entry has a location attribute of class ExprLoc, then we use the most
	// recent lexical range as the resolvable range
	var lexStart [][]uint64
	var lexEnd [][]uint64
	var lexIdx int
	var lexSibling dwarf.Offset

	// default to zero for start/end addresses. this means we can access the
	// arrays without any special conditions
	lexStart = append(lexStart, []uint64{0})
	lexEnd = append(lexEnd, []uint64{0})

	// location lists use a base address of the current compilation unit when
	// constructing address ranges
	var compilationUnitAddress uint64

	// walk through the entire DWARF sequence in order. we'll only deal with
	// the entries that are of interest to us
	for _, e := range bld.order {
		// reset lexical block stack
		if e.Offset == lexSibling {
			if lexIdx > 0 {
				lexIdx--
			} else {
				// this should never happen unless the DWARF file is corrupt in some way
				logger.Logf("dwarf", "trying to end a lexical block without one being opened")
			}
		}

		switch e.Tag {
		case dwarf.TagCompileUnit:
			// the sibling entry indicates when a lexical block ends. if an
			// entry we're interested in (subprogram etc.) does not have a
			// sibling however, then that indicates that the lexical block is
			// at the end of the compilation unit and that the sibling is
			// implied
			//
			// when we encounter a compile unit tag therefore, we reset the
			// lexical block stack
			lexIdx = 0
			lexSibling = 0

			var l, h uint64

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				l = origin + uint64(fld.Val.(uint64))

				fld = e.AttrField(dwarf.AttrHighpc)
				if fld == nil {
					continue // for loop
				}

				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					h = l + uint64(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					h = uint64(fld.Val.(uint64))
				default:
				}

				lexIdx++
				lexStart = append(lexStart[:lexIdx], []uint64{l})
				lexEnd = append(lexEnd[:lexIdx], []uint64{h})

				// we can think of the compilationUnitAddress as a special form
				// of the lexical block and is used for location lists. we're
				// not interested in the high address in that context
				compilationUnitAddress = l

			} else {
				logger.Logf("dwarf", "no LowPC attribute for compilation unit")
				compilationUnitAddress = origin
			}

			continue // for loop

		case dwarf.TagSubprogram:
			fallthrough
		case dwarf.TagInlinedSubroutine:
			fallthrough
		case dwarf.TagLexDwarfBlock:
			var l, h uint64

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				l = origin + uint64(fld.Val.(uint64))

				fld = e.AttrField(dwarf.AttrHighpc)
				if fld == nil {
					continue // for loop
				}

				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					h = l + uint64(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					h = uint64(fld.Val.(uint64))
				default:
				}

				lexIdx++
				lexStart = append(lexStart[:lexIdx], []uint64{l})
				lexEnd = append(lexEnd[:lexIdx], []uint64{h})
			} else {
				fld = e.AttrField(dwarf.AttrRanges)
				if fld == nil {
					continue // for loop
				}

				rngs, err := bld.dwrf.Ranges(e)
				if err != nil {
					return err
				}

				var start []uint64
				var end []uint64
				for _, r := range rngs {
					start = append(start, r[0])
					end = append(end, r[1])
				}
				lexIdx++
				lexStart = append(lexStart[:lexIdx], start)
				lexEnd = append(lexEnd[:lexIdx], end)
			}

			// if there is no sibling for the lexical block then that indicates
			// that the block will end with the compilation unit
			fld = e.AttrField(dwarf.AttrSibling)
			if fld != nil {
				lexSibling = fld.Val.(dwarf.Offset)
			}

			continue // for loop

		case dwarf.TagFormalParameter:
			fallthrough
		case dwarf.TagVariable:
			// execute rest of for block

		default:
			// ignore all other DWARF tags
			continue // for loop
		}

		// get variable build entry
		v := bld.entries[e.Offset]

		// resolve name and type of variable
		var varb *SourceVariable
		var err error

		// check for abstract origin field. if it is present we resolve the
		// declartion using the DWARF entry indicated by the field. otherwise
		// we resolve using the current entry
		fld := v.AttrField(dwarf.AttrAbstractOrigin)
		if fld != nil {
			abstract, ok := bld.entries[fld.Val.(dwarf.Offset)]
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

		// adding children to the variable instance is done once all basic variables
		// have been added

		// variable actually exists in memory if it has a location attribute
		fld = v.AttrField(dwarf.AttrLocation)
		if fld != nil {
			switch fld.Class {
			case dwarf.ClassLocListPtr:
				var err error
				err = bld.debug_loc.newLoclist(varb, fld.Val.(int64), compilationUnitAddress,
					func(start, end uint64, loc *loclist) {
						v := *varb
						v.loclist = loc
						local := &SourceVariableLocal{
							SourceVariable: &v,
							Range: SourceRange{
								Start: start,
								End:   end,
							},
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

				if r, _ := bld.debug_loc.decodeLoclistOperationWithOrigin(fld.Val.([]uint8), origin); r != nil {
					varb.loclist = bld.debug_loc.newLoclistJustContext(varb)
					varb.loclist.addOperator(r)

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
						for i := range lexStart[lexIdx] {
							v := *varb
							local := &SourceVariableLocal{
								SourceVariable: &v,
								Range: SourceRange{
									Start: lexStart[lexIdx][i],
									End:   lexEnd[lexIdx][i],
								},
							}

							src.locals = append(src.locals, local)
							src.SortedLocals.Locals = append(src.SortedLocals.Locals, local)
						}
					}
				}
			}

		} else {
			// add local variable even if it has no location attribute
			if varb.DeclLine.Function.Name != stubIndicator {
				for i := range lexStart[lexIdx] {
					v := *varb
					local := &SourceVariableLocal{
						SourceVariable: &v,
						Range: SourceRange{
							Start: lexStart[lexIdx][i],
							End:   lexEnd[lexIdx][i],
						},
					}
					src.locals = append(src.locals, local)
					src.SortedLocals.Locals = append(src.SortedLocals.Locals, local)
				}
			}
		}
	}

	return nil
}

func (bld *build) buildFunctions(src *Source, origin uint64) error {
	resolveFramebase := func(e *dwarf.Entry) (*loclist, error) {
		var framebase *loclist

		fld := e.AttrField(dwarf.AttrFrameBase)
		if fld != nil {
			switch fld.Class {
			case dwarf.ClassExprLoc:
				var err error
				framebase, err = bld.debug_loc.newLoclistFromSingleOperator(src.debug_frame, fld.Val.([]uint8))
				if err != nil {
					return nil, err
				}
			case dwarf.ClassLocListPtr:
				err := bld.debug_loc.newLoclist(src.debug_frame, fld.Val.(int64), origin,
					func(_, _ uint64, loc *loclist) {
						framebase = loc
					})
				if err != nil {
					return nil, err
				}
			}
		}

		return framebase, nil
	}

	// resolve sets the return fn value for the entire function
	resolve := func(e *dwarf.Entry) (*SourceFunction, error) {
		lr, err := bld.dwrf.LineReader(bld.compileUnits[e.Offset])
		if err != nil {
			return nil, err
		}
		files := lr.Files()

		// name of entry
		fld := e.AttrField(dwarf.AttrName)
		if fld == nil {
			return nil, nil
		}
		name := fld.Val.(string)

		// declaration file
		fld = e.AttrField(dwarf.AttrDeclFile)
		if fld == nil {
			return nil, nil
		}
		filenum := fld.Val.(int64)

		// declaration line
		fld = e.AttrField(dwarf.AttrDeclLine)
		if fld == nil {
			return nil, nil
		}
		linenum := fld.Val.(int64)

		// framebase. non-abstract functions will not have a framebase
		// attribute. for those functions we can resolve it later
		framebase, err := resolveFramebase(e)
		if err != nil {
			logger.Logf("dwarf", "framebase for %s will be unreliable: %v", name, err)
		}

		// filename from file number
		filename := files[filenum].Name

		fn := &SourceFunction{
			Name:             name,
			DeclLine:         src.Files[filename].Content.Lines[linenum-1],
			framebaseLoclist: framebase,
		}

		// assign function to declaration line
		if !fn.DeclLine.Function.IsStub() && fn.DeclLine.Function.Name != fn.Name {
			logger.Logf("dwarf", "contentious function ownership for source line (%s)", fn.DeclLine)
			logger.Logf("dwarf", "%s and %s", fn.DeclLine.Function.Name, fn.Name)
		}
		fn.DeclLine.Function = fn

		return fn, nil
	}

	commit := func(fn *SourceFunction) {
		if _, ok := src.Functions[fn.Name]; !ok {
			src.Functions[fn.Name] = fn
			src.FunctionNames = append(src.FunctionNames, fn.Name)
		} else {
			// if function with the name already exists we simply add the Range
			// field to the existing function
			src.Functions[fn.Name].Range = append(src.Functions[fn.Name].Range, fn.Range...)
		}
	}

	var currentFrameBase *loclist

	for _, e := range bld.order {
		switch e.Tag {
		case dwarf.TagSubprogram:
			// check address against low/high fields
			var low uint64
			var high uint64

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld == nil {
				// it is possible for Subprograms to have no address fields.
				// the Subprograms are abstract and will be referred to by
				// either concrete Subprograms or concrete InlinedSubroutines
				continue // for loop
			}
			low = uint64(fld.Val.(uint64))

			fld = e.AttrField(dwarf.AttrHighpc)
			if fld == nil {
				return curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
			}

			switch fld.Class {
			case dwarf.ClassConstant:
				// dwarf-4
				high = low + uint64(fld.Val.(int64))
			case dwarf.ClassAddress:
				// dwarf-2
				high = uint64(fld.Val.(uint64))
			default:
				return curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
			}

			// reduce high value by one (otherwise the function reports as
			// sharing an address with an adjacent function)
			high -= 1

			// subprograms don't seem to ever have a range field (unlike
			// inlined subprograms)

			fld = e.AttrField(dwarf.AttrAbstractOrigin)
			if fld != nil {
				abstract, ok := bld.entries[fld.Val.(dwarf.Offset)]
				if !ok {
					return curated.Errorf("found inlined subroutine without abstract")
				}

				fn, err := resolve(abstract)
				if err != nil {
					return err
				}

				// start/end address of function
				fn.Range = append(fn.Range, SourceRange{
					Start: low + origin,
					End:   high + origin,
				})

				// try to acquire framebase for concrete subroutine. we don't expect
				// for the framebase to have been found already but we'll check it
				// to make sure in any case
				if fn.framebaseLoclist == nil {
					fn.framebaseLoclist, err = resolveFramebase(e)
					if err != nil {
						logger.Logf("dwarf", "framebase for %s will be unreliable: %v", fn.Name, err)
					}
				} else {
					logger.Logf("dwarf", "%s: concrete defintion for abstract function already has a framebase defintion!?", fn.Name)
				}

				// note framebase so that we can use it for inlined functions
				currentFrameBase = fn.framebaseLoclist

				commit(fn)

			} else {
				fn, err := resolve(e)
				if err != nil {
					return err
				}

				// start/end address of function
				fn.Range = append(fn.Range, SourceRange{
					Start: low + origin,
					End:   high + origin,
				})

				// note framebase so that we can use it for inlined functions
				currentFrameBase = fn.framebaseLoclist

				commit(fn)
			}

		case dwarf.TagInlinedSubroutine:
			// inlined subroutines have more complex memory placement
			commitInlinedSubroutine := func(low uint64, high uint64) error {
				fld := e.AttrField(dwarf.AttrAbstractOrigin)
				if fld == nil {
					return curated.Errorf("missing abstract origin for inlined subroutine")
				}

				abstract, ok := bld.entries[fld.Val.(dwarf.Offset)]
				if !ok {
					return curated.Errorf("found inlined subroutine without abstract")
				}

				fn, err := resolve(abstract)
				if err != nil {
					return err
				}

				// start/end address of function
				fn.Range = append(fn.Range, SourceRange{
					Start:  low + origin,
					End:    high + origin,
					Inline: true,
				})

				// inlined functions will not have a framebase attribute so we
				// use the most recent one found
				fn.framebaseLoclist = currentFrameBase

				commit(fn)

				return nil
			}

			var low uint64
			var high uint64

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				low = uint64(fld.Val.(uint64))

				// high PC
				fld = e.AttrField(dwarf.AttrHighpc)
				if fld == nil {
					return curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
				}

				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					high = low + uint64(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					high = uint64(fld.Val.(uint64))
				default:
					return curated.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
				}

				err := commitInlinedSubroutine(low, high)
				if err != nil {
					return err
				}

			} else {
				fld = e.AttrField(dwarf.AttrRanges)
				if fld == nil {
					continue // for loop
				}

				rngs, err := bld.dwrf.Ranges(e)
				if err != nil {
					return err
				}

				for _, r := range rngs {
					err := commitInlinedSubroutine(r[0], r[1])
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
