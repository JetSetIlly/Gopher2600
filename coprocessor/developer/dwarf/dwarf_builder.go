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
	"debug/dwarf"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/jetsetilly/gopher2600/logger"
)

// compile units are made up of many children. for convenience we keep track of
// all children in an index
type compileUnit struct {
	e *dwarf.Entry

	// name of the compilation unit. the dwarf.AttrName value from the dwarf.Entry
	name string

	// the index of all entries that are grouped under this compilation unit
	children map[dwarf.Offset]*dwarf.Entry

	// the base address for addresses found under this compilation unit
	origin uint64

	// the range of addresses covered by the compilation unit
	ranges [][2]uint64

	// whether this compilation unit appears to optimised
	optimisation bool
}

// the build struct is only used during construction of debugging information
type build struct {
	dwrf *dwarf.Data

	// it is sometimes useful to access an dwarf entry directly by an offset value.
	// references from one entry to another is done by offset so in those situations
	// an index saves searching. a more general solution would be to create a tree
	// from the dwarf data and limit the search, but this is okay
	idx map[dwarf.Offset]*dwarf.Entry

	// it's easier to traverse a linear array rather than dealing with the
	// dwardf.Data.Reader() on multiple occasions
	order []*dwarf.Entry

	// the parent compileunit for every dwarf entry. we directly record this
	// relationship because sometimes a DWARF entry will reference another DWARF
	// entry in a different compile unit and we need to know the compile unit of
	// that second entry. (the reason we need to know the compile unit is so we
	// can attain the line reader)
	parent map[dwarf.Offset]*compileUnit

	// an index of just the compile units. indexed by the offset of the compile unit
	units map[dwarf.Offset]*compileUnit

	// the type, globals and local variables discovered during the build process
	types   map[dwarf.Offset]*SourceType
	globals map[string]*SourceVariable
	locals  []*SourceVariableLocal
}

func newBuild(dwrf *dwarf.Data) (*build, error) {
	bld := &build{
		dwrf:    dwrf,
		idx:     make(map[dwarf.Offset]*dwarf.Entry),
		parent:  make(map[dwarf.Offset]*compileUnit),
		units:   make(map[dwarf.Offset]*compileUnit),
		types:   make(map[dwarf.Offset]*SourceType),
		globals: make(map[string]*SourceVariable),
	}

	r := bld.dwrf.Reader()
	for {
		entry, err := r.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // for loop
			}
			return nil, err
		}
		if entry == nil {
			break // for loop
		}

		bld.order = append(bld.order, entry)
		bld.idx[entry.Offset] = entry
	}

	return bld, nil
}

// build list of compilation units in the DWARF data
func (bld *build) buildCompilationUnits() error {
	var unit *compileUnit

	for _, e := range bld.order {
		switch e.Tag {
		case dwarf.TagCompileUnit:
			unit = &compileUnit{
				e:        e,
				children: make(map[dwarf.Offset]*dwarf.Entry),
			}

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				unit.origin = fld.Val.(uint64)

				fld := e.AttrField(dwarf.AttrHighpc)
				if fld != nil {
					var high uint64
					switch fld.Class {
					case dwarf.ClassConstant:
						// dwarf-4
						high = unit.origin + uint64(fld.Val.(int64))
					case dwarf.ClassAddress:
						// dwarf-2
						high = fld.Val.(uint64)
					default:
					}
					unit.ranges = append(unit.ranges, [2]uint64{unit.origin, high})

					fld = e.AttrField(dwarf.AttrRanges)
					if fld != nil {
						return fmt.Errorf("unexpected AttrRanges in compilation unit")
					}

				} else {
					fld = e.AttrField(dwarf.AttrRanges)
					if fld != nil {
						commitRange := func(low uint64, high uint64) {
							unit.ranges = append(unit.ranges, [2]uint64{low, high})
						}

						err := bld.processRanges(e, unit.origin, commitRange)
						if err != nil {
							return err
						}
					}
				}
			}

			fld = e.AttrField(dwarf.AttrName)
			if fld != nil {
				unit.name = fld.Val.(string)
			}

			// sub-optimal detection of whether the compileunit was generated with compiler
			// optimisation enabled
			fld = unit.e.AttrField(dwarf.AttrProducer)
			if fld != nil {
				s := fld.Val.(string)
				unit.optimisation = strings.HasPrefix(s, "GNU") && strings.Contains(s, " -O")
			}

			bld.units[e.Offset] = unit
		default:
			bld.parent[e.Offset] = unit
		}
	}

	return nil
}

// read the source files used by each compilation units
func (bld *build) buildSourceFiles(src *Source) error {
	for _, u := range bld.units {
		// read each file referenced by the compilation unit
		r, err := bld.dwrf.LineReader(u.e)
		if err == nil {
			for _, f := range r.Files()[1:] {
				if _, ok := src.Files[f.Name]; !ok {
					sf, err := readSourceFile(f.Name, src.path, &src.AllLines)
					if err != nil {
						logger.Log(logger.Allow, "dwarf", err)
					} else {
						src.Files[sf.Filename] = sf
						src.Filenames = append(src.Filenames, sf.Filename)
						src.FilesByShortname[sf.ShortFilename] = sf
						src.ShortFilenames = append(src.ShortFilenames, sf.ShortFilename)
					}
				}
			}
		}
	}
	return nil
}

// buildTypes creates the types necessary to build variable information. in
// parituclar allocation of members to the "parent" composite type
func (bld *build) buildTypes(src *Source) error {
	resolveTypeDefs := func() error {
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagTypedef:
				baseType, err := bld.resolveType(e)
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

				bld.types[e.Offset] = typ
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

			bld.types[e.Offset] = &typ
		}
	}

	// three passes over more complex types
	for pass := 0; pass < 3; pass++ {
		err := resolveTypeDefs()
		if err != nil {
			return err
		}

		// pointer types
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagPointerType:
				var typ SourceType

				typ.PointerType, err = bld.resolveType(e)
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

				bld.types[e.Offset] = &typ
			}
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

				bld.types[e.Offset] = &typ

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
				composite = bld.types[e.Offset]

			case dwarf.TagMember:
				if composite == nil {
					// found a member without first finding a composite type. this
					// shouldn't happen
					continue
				}

				// members are basically like variables but with special address
				// handling
				memb, err := bld.resolveVariableDeclaration(e, nil, src)
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
						memb.loclist = src.debugLoc.newLoclistJustFramebase(memb)
						address := fld.Val.(int64)
						memb.loclist.addOperator(loclistOperator{
							resolve: func(loc *loclist, _ io.Writer) (loclistStack, error) {
								return loclistStack{
									class: stackClassIsValue,
									value: uint32(address),
								}, nil
							},
							operator: "member offset",
						})
					case dwarf.ClassExprLoc:
						memb.loclist = src.debugLoc.newLoclistJustFramebase(memb)
						expr := fld.Val.([]uint8)
						r, n, err := src.debugLoc.decodeLoclistOperation(expr)
						if err != nil {
							return err
						}
						if n == 0 {
							return fmt.Errorf("unhandled expression operator %02x", expr[0])
						}
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
				if bld.types[e.Offset] != nil && len(bld.types[e.Offset].Members) == 0 {
					delete(bld.types, e.Offset)
				}
			}
		}

		// build array types
		var arrayBaseType *SourceType
		var baseTypeOffset dwarf.Offset
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagArrayType:
				var err error
				arrayBaseType, err = bld.resolveType(e)
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

				var elementCount int

				fld := e.AttrField(dwarf.AttrUpperBound)
				if fld == nil {
					continue
				}
				elementCount = int(fld.Val.(int64) + 1)

				bld.types[baseTypeOffset] = &SourceType{
					Name:         fmt.Sprintf("%s", arrayBaseType.Name),
					Size:         arrayBaseType.Size * elementCount,
					ElementType:  arrayBaseType,
					ElementCount: elementCount,
				}

			default:
				arrayBaseType = nil
			}
		}

		// const types
		for _, e := range bld.order {
			switch e.Tag {
			case dwarf.TagConstType:
				baseType, err := bld.resolveType(e)
				if err != nil {
					return err
				}
				if baseType == nil {
					continue
				}

				typ := *baseType
				if !typ.Constant {
					typ.Constant = true
					typ.Name = fmt.Sprintf("const %s", baseType.Name)
				}

				bld.types[e.Offset] = &typ
			}
		}
	}

	// customise known types
	var conversion func(typ *SourceType)
	conversion = func(typ *SourceType) {
		if strings.Contains(typ.Name, "float") {
			if typ.Conversion == nil {
				typ.Conversion = func(v uint32) (string, any) {
					return "%f", math.Float32frombits(v)
				}
			}
		}
		for _, m := range typ.Members {
			if typ != m.Type {
				conversion(m.Type)
			}
		}
		if typ.ElementType != nil {
			conversion(typ.ElementType)
		}
		if typ.PointerType != nil {
			conversion(typ.PointerType)
		}
	}
	for _, typ := range bld.types {
		conversion(typ)
	}

	return nil
}

func (bld *build) resolveType(v *dwarf.Entry) (*SourceType, error) {
	if v == nil {
		return nil, nil
	}

	fld := v.AttrField(dwarf.AttrType)
	if fld == nil {
		return nil, nil
	}

	typ, ok := bld.types[fld.Val.(dwarf.Offset)]
	if !ok {
		return nil, nil
	}

	return typ, nil
}

func (bld *build) resolveVariableDeclaration(v *dwarf.Entry, t *dwarf.Entry, src *Source) (*SourceVariable, error) {
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

		// prefer type from 't' entry
		if t != nil {
			varb.Type, err = bld.resolveType(t)
			if err != nil {
				return nil, err
			}
		}

		// if type has not been resolved use 'v' entry
		if varb.Type == nil {
			varb.Type, err = bld.resolveType(v)
			if err != nil {
				return nil, err
			}

			// return nothing if there is still no type field
			if varb.Type == nil {
				return nil, nil
			}
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

		spec, ok := bld.idx[fld.Val.(dwarf.Offset)]
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

	lr, err := bld.dwrf.LineReader(bld.parent[v.Offset].e)
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
func (bld *build) buildVariables(src *Source) error {
	// location lists use a base address of the current compilation unit when
	// constructing address ranges
	var compilationUnitAddress uint64

	// stack of entries representing the tree. the newest entry is always at
	// index zero
	var stack []*dwarf.Entry

	// if the length of the stack is equal to this value then that means any
	// variable is in global scope
	const globalDepth = 1

	// walk through the entire DWARF sequence in order. we'll only deal with
	// the entries that are of interest to us
	for _, e := range bld.order {
		// increase depth if the entry has children. the moment when we choose
		// to increase the depth value impacts what is meant by globalDepth (see
		// constant value of same name)
		if e.Children {
			stack = append([]*dwarf.Entry{e}, stack...)
		}

		switch e.Tag {
		case 0:
			// reduce depth value if this is a null entry
			if len(stack) > 0 {
				stack = stack[1:]
			}
		case dwarf.TagCompileUnit:
			// basic compilation address is the address adjustment value. this
			// will be changed depending on the presence of AttrLowpc
			compilationUnitAddress = bld.units[e.Offset].origin
			continue // for loop

		case dwarf.TagVariable, dwarf.TagFormalParameter:
			// execute rest of for block

		default:
			// ignore all other DWARF tags
			continue // for loop
		}

		// resolve name and type of variable
		var varb *SourceVariable
		var err error

		// check for abstract origin field or specification field
		//
		// if either is present we resolve the abstract declartion otherwise we
		// resolve using the current entry
		fld := e.AttrField(dwarf.AttrAbstractOrigin)
		if fld == nil {
			fld = e.AttrField(dwarf.AttrSpecification)
		}

		if fld != nil {
			v, ok := bld.idx[fld.Val.(dwarf.Offset)]
			if !ok {
				return fmt.Errorf("found concrete variable without abstract")
			}

			varb, err = bld.resolveVariableDeclaration(v, e, src)
			if err != nil {
				return err
			}
		} else {
			varb, err = bld.resolveVariableDeclaration(e, nil, src)
			if err != nil {
				return err
			}
		}

		// nothing found when resolving the declaration
		if varb == nil {
			continue // for loop
		}

		// make sure the variable has a type
		if varb.Type == nil {
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

		// add variable to list of globals if aprropriate. returns true if the
		// variable has been added and false if it is not a global
		addGlobal := func(varb *SourceVariable) bool {
			if len(stack) != globalDepth {
				return false
			}

			g, ok := bld.globals[varb.Name]

			if !ok || (!g.IsValid() && varb.IsValid()) {
				bld.globals[varb.Name] = varb
			}

			// note that the file has at least one global variable
			varb.DeclLine.File.HasGlobals = true

			return true
		}

		// add local variable
		addLocal := func(varb *SourceVariable) {
			findRange := func(e *dwarf.Entry) (uint64, uint64, bool) {
				var low, high uint64
				fld := e.AttrField(dwarf.AttrLowpc)
				if fld != nil {
					low = fld.Val.(uint64)

					fld = e.AttrField(dwarf.AttrHighpc)
					if fld == nil {
						return 0, 0, false
					}

					switch fld.Class {
					case dwarf.ClassConstant:
						// dwarf-4
						high = low + uint64(fld.Val.(int64))
					case dwarf.ClassAddress:
						// dwarf-2
						high = fld.Val.(uint64)
					default:
					}

					// "high address is the first location past the last instruction
					// associated with the entity"
					// page 34 of "DWARF4 Standard"
					high--
				} else {
					fld = e.AttrField(dwarf.AttrRanges)
					if fld == nil {
						return 0, 0, false
					}

					var start []uint64
					var end []uint64

					commitRange := func(low uint64, high uint64) {
						start = append(start, low)
						end = append(end, high)
					}

					err := bld.processRanges(e, compilationUnitAddress, commitRange)
					if err != nil {
						return 0, 0, false
					}
				}
				return low, high, true
			}

			var low, high uint64
			var foundRange bool
			for _, e := range stack {
				low, high, foundRange = findRange(e)
				if foundRange {
					break // for loop
				}
			}
			if !foundRange {
				logger.Logf(logger.Allow, "dwarf", "orphaned local variable: %s", varb.Name)
			}

			cp := *varb
			local := &SourceVariableLocal{
				SourceVariable: &cp,
				Range: SourceRange{
					Start: low,
					End:   high,
				},
			}

			bld.locals = append(bld.locals, local)
		}

		// variable actually exists if it has a location or constant value attribute

		constfld := e.AttrField(dwarf.AttrConstValue)
		if constfld != nil {
			varb.hasConstantValue = true

			switch constfld.Val.(type) {
			case int64:
				varb.constantValue = uint32(constfld.Val.(int64))
			case []uint8:
				// eg. a float value
				varb.constantValue = src.debugLoc.byteOrder.Uint32(constfld.Val.([]uint8))
			default:
				logger.Logf(logger.Allow, "dwarf", "unhandled DW_AT_const_value type %T", constfld.Val)
				continue // for loop
			}

			if !addGlobal(varb) {
				addLocal(varb)
			}
		} else {
			locfld := e.AttrField(dwarf.AttrLocation)
			if locfld != nil {
				switch locfld.Class {
				case dwarf.ClassLocListPtr:
					ptrCommit := func(start, end uint64, loc *loclist) {
						cp := *varb
						cp.loclist = loc
						local := &SourceVariableLocal{
							SourceVariable: &cp,
							Range: SourceRange{
								Start: start,
								End:   end,
							},
						}
						bld.locals = append(bld.locals, local)
					}

					err := src.debugLoc.newLoclistFromPtr(varb, locfld.Val.(int64), compilationUnitAddress, ptrCommit)
					if err != nil {
						if errors.Is(err, UnsupportedDWARF) {
							return err
						}
						logger.Logf(logger.Allow, "dwarf", "%s: %v", varb.Name, err)
					}

					if !addGlobal(varb) {
						addLocal(varb)
					}

				case dwarf.ClassExprLoc:
					// Single location description "They are sufficient for describing the location of any object
					// as long as its lifetime is either static or the same as the lexical block that owns it,
					// and it does not move during its lifetime"
					// page 26 of "DWARF4 Standard"

					var err error
					varb.loclist, err = src.debugLoc.newLoclistFromExpr(src.debugFrame, locfld.Val.([]uint8))
					if err != nil {
						return err
					}

					if !addGlobal(varb) {
						addLocal(varb)
					}
				}
			}
		}
	}

	return nil
}

func (bld *build) buildFunctions(src *Source) error {
	// location lists use a base address of the current compilation unit when
	// constructing address ranges
	var compilationUnitAddress uint64

	resolveFramebase := func(e *dwarf.Entry) (*loclist, error) {
		var framebase *loclist

		fld := e.AttrField(dwarf.AttrFrameBase)
		if fld != nil {
			switch fld.Class {
			case dwarf.ClassExprLoc:
				var err error
				framebase, err = src.debugLoc.newLoclistFromExpr(src.debugFrame, fld.Val.([]uint8))
				if err != nil {
					return nil, err
				}

			case dwarf.ClassLocListPtr:
				err := src.debugLoc.newLoclistFromPtr(src.debugFrame, fld.Val.(int64), compilationUnitAddress,
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

	// resolve the dwarf entry e for the filename and framebase. if the
	// framebase is in another dwarf entry then that can be specified with the
	// fb paramenter. fb can be nil
	//
	// may return nil for both error and SourceFunction value. therefore, the
	// caller should check for an error first and if that is nil, check the
	// SourceFunction value before working with it
	resolve := func(e *dwarf.Entry, fb *dwarf.Entry) (*SourceFunction, error) {
		// if fb is nil then use e by default
		if fb == nil {
			fb = e
		}

		lr, err := bld.dwrf.LineReader(bld.parent[e.Offset].e)
		if err != nil {
			return nil, err
		}
		files := lr.Files()

		// name of entry might be the entry we've been given or in a specification
		fld := e.AttrField(dwarf.AttrName)
		if fld == nil {
			sp := e.AttrField(dwarf.AttrSpecification)
			if sp != nil {
				fld = bld.idx[sp.Val.(dwarf.Offset)].AttrField(dwarf.AttrName)
			}
		}
		if fld == nil {
			return nil, fmt.Errorf("no function name")
		}
		name := fld.Val.(string)

		// linkage name may be different to the name of the function
		linkageName := name
		fld = e.AttrField(dwarf.AttrLinkageName)
		if fld == nil {
			fld = e.AttrField(dwarf.AttrSpecification)
			if fld != nil {
				fld = bld.idx[fld.Val.(dwarf.Offset)].AttrField(dwarf.AttrLinkageName)
				if fld != nil {
					linkageName = fld.Val.(string)
				}
			}
		} else {
			linkageName = fld.Val.(string)
		}

		// declaration file
		fld = e.AttrField(dwarf.AttrDeclFile)
		if fld == nil {
			fld = e.AttrField(dwarf.AttrSpecification)
			if fld == nil {
				return nil, fmt.Errorf("no source file for %s", name)
			}
			fld = bld.idx[fld.Val.(dwarf.Offset)].AttrField(dwarf.AttrDeclFile)
			if fld == nil {
				return nil, fmt.Errorf("no source file for %s", name)
			}
		}
		filenum := fld.Val.(int64)

		// declaration line
		fld = e.AttrField(dwarf.AttrDeclLine)
		if fld == nil {
			fld = e.AttrField(dwarf.AttrSpecification)
			if fld == nil {
				return nil, fmt.Errorf("no line number for %s", name)
			}
			fld = bld.idx[fld.Val.(dwarf.Offset)].AttrField(dwarf.AttrDeclLine)
			if fld == nil {
				return nil, fmt.Errorf("no line number for %s", name)
			}
		}
		linenum := fld.Val.(int64)

		// framebase. non-abstract functions will not have a framebase
		// attribute. for those functions we can resolve it later
		framebase, err := resolveFramebase(fb)
		if err != nil {
			logger.Logf(logger.Allow, "dwarf", "framebase for %s will be unreliable: %v", name, err)
		}

		// filename from file number
		filename := files[filenum].Name

		if src.Files[filename] == nil {
			return nil, fmt.Errorf("no file named %s", filename)
		}

		fn := &SourceFunction{
			Name:        name,
			linkageName: linkageName,
			DeclLine:    src.Files[filename].Content.Lines[linenum-1],
			framebase:   framebase,
		}

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
			fn = src.Functions[fn.Name]
		}

		// assign function to declaration line
		if !fn.DeclLine.Function.IsStub() && fn.DeclLine.Function.Name != fn.Name {
			logger.Logf(logger.Allow, "dwarf", "contentious function ownership for source line (%s)", fn.DeclLine)
			logger.Logf(logger.Allow, "dwarf", "%s and %s", fn.DeclLine.Function.Name, fn.Name)
		}
		fn.DeclLine.Function = fn
	}

	// the framebase location list to use when preparing inline functions
	var currentFrameBase *loclist

	for _, e := range bld.order {
		switch e.Tag {
		case dwarf.TagCompileUnit:
			compilationUnitAddress = bld.units[e.Offset].origin
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
			low = fld.Val.(uint64)

			fld = e.AttrField(dwarf.AttrHighpc)
			if fld == nil {
				return fmt.Errorf("AttrLowpc without AttrHighpc for Subprogram")
			}

			switch fld.Class {
			case dwarf.ClassConstant:
				// dwarf-4
				high = low + uint64(fld.Val.(int64))
			case dwarf.ClassAddress:
				// dwarf-2
				high = fld.Val.(uint64)
			default:
				return fmt.Errorf("AttrLowpc without AttrHighpc for Subprogram")
			}

			// "high address is the first location past the last instruction
			// associated with the entity"
			// page 34 of "DWARF4 Standard"
			high--

			// subprograms don't seem to ever have a range field (unlike
			// inlined subprograms)

			fld = e.AttrField(dwarf.AttrAbstractOrigin)
			if fld == nil {
				fld = e.AttrField(dwarf.AttrSpecification)
			}
			if fld != nil {
				av, ok := bld.idx[fld.Val.(dwarf.Offset)]
				if !ok {
					return fmt.Errorf("found inlined subroutine without abstract")
				}

				var fn *SourceFunction
				var err error

				if fld.Attr == dwarf.AttrSpecification {
					fn, err = resolve(e, av)
					if err != nil {
						logger.Log(logger.Allow, "dwarf", err)
						continue // build order loop
					}
				} else {
					fn, err = resolve(av, nil)
					if err != nil {
						logger.Log(logger.Allow, "dwarf", err)
						continue // build order loop
					}
				}

				// start/end address of function
				fn.Range = append(fn.Range, SourceRange{
					Start: low,
					End:   high,
				})

				// try to acquire framebase for concrete subroutine. we don't expect
				// for the framebase to have been found already but we'll check it
				// to make sure in any case
				if fn.framebase == nil {
					fn.framebase, err = resolveFramebase(e)
					if err != nil {
						logger.Logf(logger.Allow, "dwarf", "framebase for %s will be unreliable: %v", fn.Name, err)
					}
				} else {
					logger.Logf(logger.Allow, "dwarf", "%s: concrete defintion for abstract function already has a framebase defintion!?", fn.Name)
				}

				// note framebase so that we can use it for inlined functions
				currentFrameBase = fn.framebase

				commit(fn)
				continue // for bld.order
			}

			fn, err := resolve(e, nil)
			if err != nil {
				logger.Log(logger.Allow, "dwarf", err)
				continue // build order loop
			}

			// start/end address of function
			fn.Range = append(fn.Range, SourceRange{
				Start: low,
				End:   high,
			})

			// note framebase so that we can use it for inlined functions
			currentFrameBase = fn.framebase

			commit(fn)

		case dwarf.TagInlinedSubroutine:
			// inlined subroutines have more complex memory placement
			commitInlinedSubroutine := func(low uint64, high uint64) error {
				fld := e.AttrField(dwarf.AttrAbstractOrigin)
				if fld == nil {
					return fmt.Errorf("missing abstract origin for inlined subroutine")
				}

				av, ok := bld.idx[fld.Val.(dwarf.Offset)]
				if !ok {
					return fmt.Errorf("found inlined subroutine without abstract")
				}

				fn, err := resolve(av, nil)
				if err != nil {
					return err
				}

				// start/end address of function
				fn.Range = append(fn.Range, SourceRange{
					Start:  low,
					End:    high,
					Inline: true,
				})

				// inlined functions will not have a framebase attribute so we
				// use the most recent one found
				fn.framebase = currentFrameBase

				commit(fn)

				return nil
			}

			var low uint64
			var high uint64

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				low = fld.Val.(uint64)

				// high PC
				fld = e.AttrField(dwarf.AttrHighpc)
				if fld == nil {
					return fmt.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
				}

				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					high = low + uint64(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					high = fld.Val.(uint64)
				default:
					return fmt.Errorf("AttrLowpc without AttrHighpc for InlinedSubroutine")
				}

				// "high address is the first location past the last instruction
				// associated with the entity"
				// page 34 of "DWARF4 Standard"
				high--

				err := commitInlinedSubroutine(low, high)
				if err != nil {
					logger.Log(logger.Allow, "dwarf", err)
					continue // build order loop
				}

			} else {
				fld = e.AttrField(dwarf.AttrRanges)
				if fld == nil {
					continue // for loop
				}

				commitRange := func(low uint64, high uint64) {
					err := commitInlinedSubroutine(low, high)
					if err != nil {
						logger.Log(logger.Allow, "dwarf", err)
					}
				}

				err := bld.processRanges(e, compilationUnitAddress, commitRange)
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// process ranges by calling the supplied commit function for every range entry.
// the compilationUnitAddress will be the base address of each entry, as
// described in the DWARF 4 standard
func (bld *build) processRanges(e *dwarf.Entry, compilationUnitAddress uint64, commit func(uint64, uint64)) error {
	rngs, err := bld.dwrf.Ranges(e)
	if err != nil {
		return err
	}

	// "The applicable rangeBase address of a range list entry is determined by the closest
	// preceding rangeBase address selection entry (see below) in the same range list. If
	// there is no such selection entry, then the applicable rangeBase address defaults to
	// the rangeBase address of the compilation unit"
	// page 39 of "DWARF4 Standard"
	rangeBase := compilationUnitAddress

	for _, r := range rngs {
		// "A base address selection entry consists of:
		// 1. The value of the largest representable address offset (for example, 0xffffffff when the size of
		// an address is 32 bits).
		// 2. An address, which defines the appropriate base address for use in interpreting the beginning
		// and ending address offsets of subsequent entries of the location list."
		// page 39 of "DWARF4 Standard"
		if r[0] == 0xffffffff {
			// not sure if the adjustment is required or if it should be
			// executable origin address rather than the compilation unit
			// address
			rangeBase = compilationUnitAddress + r[1]
			continue
		}

		// ignore range entries which are empty
		if r[0] == r[1] {
			continue
		}

		low := rangeBase + r[0]
		high := rangeBase + r[1]

		// "[high address] marks the first address past the end of the address range.The ending address
		// must be greater than or equal to the beginning address"
		// page 39 of "DWARF4 Standard"
		high--

		commit(low, high)
	}
	return nil
}
