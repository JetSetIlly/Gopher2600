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
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi"
	"github.com/jetsetilly/gopher2600/logger"
)

// compile units are made up of many children. for convenience/speed we keep
// track of the children as an index rather than a tree.
type compileUnit struct {
	unit                    *dwarf.Entry
	children                map[dwarf.Offset]*dwarf.Entry
	unsupportedOptimisation string
}

// SourceFile is a single source file indentified by the DWARF data.
type SourceFile struct {
	Filename      string
	ShortFilename string
	Lines         []*SourceLine

	// the source file has at least one global variable if HasGlobals is true
	HasGlobals bool
}

// SourceDisasm is a single disassembled intruction from the ELF binary.
type SourceDisasm struct {
	Addr        uint32
	Opcode      uint16
	Instruction string

	Line *SourceLine
}

func (d *SourceDisasm) String() string {
	return fmt.Sprintf("%#08x %04x %s", d.Addr, d.Opcode, d.Instruction)
}

// SourceLine is a single line of source in a source file, identified by the
// DWARF data and loaded from the actual source file.
type SourceLine struct {
	// the actual file/line of the SourceLine
	File       *SourceFile
	LineNumber int

	// the function the line of source can be found within
	Function *SourceFunction

	// plain string of line
	PlainContent string

	// line divided into parts
	Fragments []SourceLineFragment

	// the generated assembly for this line. will be empty if line is a comment or otherwise unsused
	Disassembly []*SourceDisasm

	// some source lines will interleave their coproc instructions
	// (disassembly) with other source lines
	Interleaved bool

	// whether this source line has been responsible for an illegal access of memory
	IllegalAccess bool

	// cycle statisics for the line
	Stats Stats

	// kernel specific cycle statistics for the line. accumulated only once TV is stable
	StatsVBLANK   Stats
	StatsScreen   Stats
	StatsOverscan Stats
	StatsROMSetup Stats

	// which 2600 kernel has this line executed in
	Kernel InKernel
}

func (ln *SourceLine) String() string {
	return fmt.Sprintf("%s:%d", ln.File.Filename, ln.LineNumber)
}

// SourceFunction is a single function identified by the DWARF data.
type SourceFunction struct {
	Name string

	// first source line for each instance of the function
	DeclLine *SourceLine

	// cycle statisics for the function
	Stats Stats

	// kernel specific cycle statistics for the function. accumulated only once TV is stable
	StatsVBLANK   Stats
	StatsScreen   Stats
	StatsOverscan Stats
	StatsROMSetup Stats

	// which 2600 kernel has this function executed in
	Kernel InKernel
}

// SourceType is a single type identified by the DWARF data. Composite types
// are differentiated by the existance of member fields.
type SourceType struct {
	Name string

	// size of values of this type (in bytes)
	Size int

	// empty if type is not a composite type. see IsComposite() function
	Members []*SourceVariable

	// number of elements in the type. if count is more than zero then this
	// type is an array
	ElementCount int

	// the base type of all the elements in the type
	BaseType *SourceType
}

// Hex returns a format string to represent a value as a correctly padded
// hexadecinal number.
func (typ *SourceType) Hex() string {
	// other fields in the SourceType instance depend on the byte size
	switch typ.Size {
	case 1:
		return "%02x"
	case 2:
		return "%04x"
	case 4:
		return "%08x"
	}
	return "%x"
}

// Bin returns a format string to represent a value as a correctly padded
// binary number.
func (typ *SourceType) Bin() string {
	// other fields in the SourceType instance depend on the byte size
	switch typ.Size {
	case 1:
		return "%08b"
	case 2:
		return "%016b"
	case 4:
		return "%032b"
	}
	return "%b"
}

// Mask returns the mask value of the correct size for the type.
func (typ *SourceType) Mask() uint32 {
	switch typ.Size {
	case 1:
		return 0xff
	case 2:
		return 0xffff
	case 4:
		return 0xffffffff
	}
	return 0
}

// SourceVariable is a single variable identified by the DWARF data.
type SourceVariable struct {
	Name string

	// variable type (int, char, etc.)
	Type *SourceType

	// first source line for each instance of the function
	DeclLine *SourceLine

	// address in memory of the variable. if the variable is a member of
	// another type then the Address is an offset from the address of the
	// parent variable
	Address         uint64
	addressIsOffset bool
}

// IsComposite returns true if SourceType represents a composite type.
func (varb *SourceVariable) IsComposite() bool {
	return len(varb.Type.Members) > 0
}

// IsArray returns true if SourceType represents an array type.
func (varb *SourceVariable) IsArray() bool {
	return varb.Type.BaseType != nil && varb.Type.ElementCount > 0
}

// AddressIsOffset returns true if SourceVariable is member of another type
func (varb *SourceVariable) AddressIsOffset() bool {
	return varb.addressIsOffset
}

func (varb *SourceVariable) String() string {
	return fmt.Sprintf("%s %s => %#08x", varb.Type.Name, varb.Name, varb.Address)
}

// Source is created from available DWARF data that has been found in relation
// to and ELF file that looks to be related to the specified ROM.
//
// It is possible for the arrays/map fields to be empty
type Source struct {
	dwrf *dwarf.Data

	compileUnits []*compileUnit

	// if any of the compile units were compiled with GCC optimisation then
	// this string will contain an appropriate message. if string is empty then
	// the detected optimisation was acceptable (or there is no optimisation or
	// the compiler is unsupported)
	//
	// a GCC optimisation of -Os is okay
	//
	// optimisation can cause misleading or confusing information (albeit still
	// technically correct in terms of performance analysis)
	UnsupportedOptimisation string

	// disassembled binary
	Disassembly map[uint64]*SourceDisasm

	// all the files in all the compile units
	Files     map[string]*SourceFile
	Filenames []string

	// functions found in the compile units
	Functions     map[string]*SourceFunction
	FunctionNames []string

	// list of funcions sorted by FrameCycles field
	SortedFunctions SortedFunctions

	// types used in the source
	Types map[dwarf.Offset]*SourceType

	// all global variables in ll compile units
	Globals          map[string]*SourceVariable
	GlobalsByAddress map[uint64]*SourceVariable
	SortedGlobals    SortedVariables

	// the highest address of any variable (not just global variables, any
	// variable)
	VariableMemtop uint64

	// lines of source code found in the compile units
	linesByAddress map[uint64]*SourceLine

	// list of source lines sorted by FrameCycles field
	SortedLines SortedLines

	// sorted lines filtered by function name
	FunctionFilters []*FunctionFilter

	// cycle statisics for the entire program
	Stats Stats

	// kernel specific cycle statistics for the program. accumulated only once TV is stable
	StatsVBLANK   Stats
	StatsScreen   Stats
	StatsOverscan Stats
	StatsROMSetup Stats

	// flag to indicate whether the execution profile has changed since it was cleared
	//
	// cheap and easy way to prevent sorting too often - rather than sort after
	// every call to execute(), we can use this flag to sort only when we need
	// to in the GUI.
	//
	// probably not scalable but sufficient for our needs of a single GUI
	// running and using the statistics for only one reason
	ExecutionProfileChanged bool
}

// NewSource is the preferred method of initialisation for the Source type.
//
// If no ELF file or valid DWARF data can be found in relation to the pathToROM
// argument, the function will return nil with an error.
//
// Once the ELF and DWARF file has been identified then Source will always be
// non-nil but with the understanding that the fields may be empty.
func NewSource(pathToROM string) (*Source, error) {
	src := &Source{
		Disassembly:      make(map[uint64]*SourceDisasm),
		Files:            make(map[string]*SourceFile),
		Filenames:        make([]string, 0, 10),
		Functions:        make(map[string]*SourceFunction),
		FunctionNames:    make([]string, 0, 10),
		Types:            make(map[dwarf.Offset]*SourceType),
		Globals:          make(map[string]*SourceVariable),
		GlobalsByAddress: make(map[uint64]*SourceVariable),
		SortedGlobals: SortedVariables{
			Variables: make([]*SourceVariable, 0, 100),
		},
		SortedFunctions: SortedFunctions{
			Functions: make([]*SourceFunction, 0, 100),
		},
		linesByAddress: make(map[uint64]*SourceLine),
		SortedLines: SortedLines{
			Lines: make([]*SourceLine, 0, 100),
		},
		ExecutionProfileChanged: true,
	}

	// open ELF file
	elf := findELF(pathToROM)
	if elf == nil {
		return nil, curated.Errorf("dwarf: compiled ELF file not found")
	}
	defer elf.Close()

	// disassemble every word in the ELF file
	for _, p := range elf.Progs {
		o := p.Open()

		addr := p.ProgHeader.Vaddr

		b := make([]byte, 2)
		for {
			n, err := o.Read(b)
			if n != 2 {
				break
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, curated.Errorf("dwarf: compiled ELF file not found")
			}

			opcode := elf.ByteOrder.Uint16(b)
			disasm := arm7tdmi.Disassemble(opcode)

			// put the disassembly entry in the
			src.Disassembly[addr] = &SourceDisasm{
				Addr:        uint32(addr),
				Opcode:      opcode,
				Instruction: disasm.String(),
			}

			addr += 2
		}
	}

	var err error

	// get DWARF information from ELF file
	src.dwrf, err = elf.DWARF()
	if err != nil {
		return nil, curated.Errorf("dwarf: %v", err)
	}

	bld, err := newBuild(src.dwrf)
	if err != nil {
		return nil, curated.Errorf("dwarf: %v", err)
	}

	// readSourceFile() will shorten the filepath of a source file using the
	// pathToROM string. however, symbolic links can confuse this so we expand
	// all symbolic links in readSourceFile() and need to do the same with the
	// pathToROM value
	var pathToROM_nosymlinks string
	pathToROM_nosymlinks, err = filepath.EvalSymlinks(pathToROM)
	if err != nil {
		pathToROM_nosymlinks = pathToROM
	}

	// compile units are made up of many files. the files and filenames are in
	// the fields below
	r := src.dwrf.Reader()

	// most recent compile unit we've seen
	var unit *compileUnit

	for {
		entry, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break // for loop
			}
			return nil, curated.Errorf("dwarf: %v", err)
		}
		if entry == nil {
			break // for loop
		}
		if entry.Offset == 0 {
			continue // for loop
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			unit = &compileUnit{
				unit:     entry,
				children: make(map[dwarf.Offset]*dwarf.Entry),
			}

			// assuming DWARF never has duplicate compile unit entries
			src.compileUnits = append(src.compileUnits, unit)

			r, err := src.dwrf.LineReader(entry)
			if err != nil {
				return nil, curated.Errorf("dwarf: %v", err)
			}

			// loop through files in the compilation unit. entry 0 is always nil
			for _, f := range r.Files()[1:] {
				sf, err := readSourceFile(f.Name, pathToROM_nosymlinks)
				if err != nil {
					logger.Logf("dwarf", "%v", err)
				} else {
					// add file to list if we've not see it already
					if _, ok := src.Files[sf.Filename]; !ok {
						src.Files[sf.Filename] = sf
						src.Filenames = append(src.Filenames, sf.Filename)
					}
				}
			}

			// check optimisation directive
			fld := entry.AttrField(dwarf.AttrProducer)
			if fld != nil {
				producer := fld.Val.(string)
				if strings.HasPrefix(producer, "GNU") {
					idx := strings.Index(producer, " -O")
					if idx > -1 {
						idx += 3
						if idx < len(producer) {
							switch producer[idx] {
							case 's':
							case ' ':
							default:
								unit.unsupportedOptimisation = fmt.Sprintf("binary compiled with unsupported optimisation (-O%c)", producer[idx])
							}
						}
					}
				}
			}

		default:
			unit.children[entry.Offset] = entry
		}
	}

	// if any of the units have unsupported optimsation then indicate that the
	// entire source has the same property
	for _, u := range src.compileUnits {
		if len(u.children) > 0 && u.unsupportedOptimisation != "" {
			src.UnsupportedOptimisation = u.unsupportedOptimisation
			logger.Logf("dwarf", unit.unsupportedOptimisation)
		}
	}

	// find reference for every meaningful source line and link to disassembly
	for _, e := range src.compileUnits {
		// read every line in the compile unit
		r, err := src.dwrf.LineReader(e.unit)
		if err != nil {
			continue // for loop
		}

		// allocation of asm entries to a SourceLine is a step behind the
		// dwarf.LineEntry. this is because of how dwarf data is stored
		//
		// the instructions associated with a source line include those at
		// addresses between the address given in an dwarf.LineEntry and the
		// address in the next entry
		var workingSourceLine *SourceLine
		var workingAddress uint64

		for {
			var le dwarf.LineEntry

			err := r.Next(&le)
			if err != nil {
				if err == io.EOF {
					break // for loop
				}
				logger.Logf("dwarf", "%v", err)
				workingSourceLine = nil
			}

			// check source matches what is in the DWARF data
			if src.Files[le.File.Name] == nil {
				logger.Logf("dwarf", "file not available for linereader: %s", le.File.Name)
				continue
			}
			if le.Line-1 > len(src.Files[le.File.Name].Lines) {
				return nil, curated.Errorf("dwarf: current source is unrelated to ELF/DWARF data (1)")
			}

			// if the entry is the end of a sequence then assign nil to workingSourceLine
			if le.EndSequence {
				workingSourceLine = nil
				continue // for loop
			}

			// if workingSourceLine is valid ...
			if workingSourceLine != nil {

				// ... and there are addresses to process ...
				if le.Address-workingAddress > 0 {

					// ... then find the function for the working address
					foundFunc, err := bld.findFunction(workingAddress)
					if err != nil {
						return nil, curated.Errorf("dwarf: %v", err)
					}

					if foundFunc == nil {
						logger.Logf("dwarf", "cannot find function for address: %08x", workingAddress)
						continue // for loop
					}

					// check source matches what is in the DWARF data
					if src.Files[foundFunc.filename] == nil {
						return nil, curated.Errorf("dwarf: current source is unrelated to ELF/DWARF data (2)")
					}
					if int(foundFunc.linenum-1) > len(src.Files[foundFunc.filename].Lines) {
						return nil, curated.Errorf("dwarf: current source is unrelated to ELF/DWARF data (3)")
					}

					// associate function with workingSourceLine making sure we
					// use the existing function if it exists
					if f, ok := src.Functions[foundFunc.name]; ok {
						workingSourceLine.Function = f
					} else {
						src.Functions[foundFunc.name] = &SourceFunction{
							Name:     foundFunc.name,
							DeclLine: src.Files[foundFunc.filename].Lines[foundFunc.linenum-1],
						}
						src.FunctionNames = append(src.FunctionNames, foundFunc.name)
						workingSourceLine.Function = src.Functions[foundFunc.name]
					}

					// add disassembly to source line and build a linesByAddress map
					for addr := workingAddress; addr < le.Address; addr += 2 {
						// look for address in disassembly
						if d, ok := src.Disassembly[addr]; ok {
							// add disassembly to the list of instructions for the workingSourceLine
							workingSourceLine.Disassembly = append(workingSourceLine.Disassembly, d)

							// link diassembly back to source line
							d.Line = workingSourceLine

							// associate the address with the workingSourceLine
							src.linesByAddress[addr] = workingSourceLine
						}
					}
				}

				// we've done with working source line
				workingSourceLine = nil
			}

			// defer current line entry
			workingSourceLine = src.Files[le.File.Name].Lines[le.Line-1]
			workingAddress = le.Address
		}
	}

	// interleaved instructions check
	disasmCt := 0.0
	interleaveCt := 0.0
	for _, ln := range src.linesByAddress {
		if len(ln.Disassembly) > 0 {
			disasmCt++
			addr := ln.Disassembly[0].Addr
			for _, d := range ln.Disassembly[1:] {
				if d.Addr > addr+2 {
					interleaveCt++
					ln.Interleaved = true
					break // disasm loop
				}
			}
		}
	}

	// sanity check of functions list
	if len(src.Functions) != len(src.FunctionNames) {
		return nil, curated.Errorf("dwarf: unmatched function definitions")
	}

	// assemble sorted functions list
	for _, fn := range src.Functions {
		src.SortedFunctions.Functions = append(src.SortedFunctions.Functions, fn)
	}

	// sort list of filenames and functions. these wont' be sorted again
	sort.Strings(src.Filenames)
	sort.Strings(src.FunctionNames)

	// assemble sorted source lines
	//
	// we must make sure that we don't duplicate a source line entry: src.Lines
	// is indexed by address. however, more than one address may point to a
	// single SourceLine
	//
	// to prevent adding a SourceLine more than once we keep an "observed" map
	// indexed by (and this is important) the pointer address of the SourceLine
	// and not the execution address
	observed := make(map[*SourceLine]bool)
	for _, ln := range src.linesByAddress {
		if _, ok := observed[ln]; !ok {
			observed[ln] = true
			src.SortedLines.Lines = append(src.SortedLines.Lines, ln)
		}
	}

	// build types
	err = bld.buildTypes(src)
	if err != nil {
		return nil, curated.Errorf("dwarf: %v", err)
	}

	// build variables
	err = bld.buildVariables(src)
	if err != nil {
		return nil, curated.Errorf("dwarf: %v", err)
	}

	// sort sorted lines
	sort.Sort(src.SortedLines)

	// sorted functions
	sort.Sort(src.SortedFunctions)

	// sorted functions
	sort.Sort(src.SortedGlobals)

	// log summary
	logger.Logf("dwarf", "identified %d functions in %d compile units", len(src.Functions), len(src.compileUnits))
	logger.Logf("dwarf", "highest memory address occupied by a variable (%08x)", src.VariableMemtop)

	return src, nil
}

// find source line for program counter. shouldn't be called too often because
// it's expensive.
//
// the src.LinesByAddress is an alternative source of this information but this
// function is good in cases where LinesByAddress won't have been initialised
// yet.
func (src *Source) findSourceLine(addr uint32) (*SourceLine, error) {
	for _, e := range src.compileUnits {
		r, err := src.dwrf.LineReader(e.unit)
		if err != nil {
			return nil, err
		}

		var entry dwarf.LineEntry

		err = r.SeekPC(uint64(addr), &entry)
		if err != nil {
			if err == dwarf.ErrUnknownPC {
				// not in this compile unit
				continue // for loop
			}
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}

		if src.Files[entry.File.Name] == nil {
			return nil, fmt.Errorf("%s not in list of files", entry.File.Name)
		}

		return src.Files[entry.File.Name].Lines[entry.Line-1], nil
	}

	return nil, nil
}

func (src *Source) executionProfile(addr uint32, ct float32, kernel InKernel) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	line, ok := src.linesByAddress[uint64(addr)]
	if ok {
		line.Stats.count += ct
		line.Function.Stats.count += ct
		src.Stats.count += ct

		line.Kernel |= kernel
		line.Function.Kernel |= kernel
		line.Function.DeclLine.Kernel |= kernel

		switch kernel {
		case InVBLANK:
			line.StatsVBLANK.count += ct
			line.Function.StatsVBLANK.count += ct
			src.StatsVBLANK.count += ct
		case InScreen:
			line.StatsScreen.count += ct
			line.Function.StatsScreen.count += ct
			src.StatsScreen.count += ct
		case InOverscan:
			line.StatsOverscan.count += ct
			line.Function.StatsOverscan.count += ct
			src.StatsOverscan.count += ct
		case InROMSetup:
			line.StatsROMSetup.count += ct
			line.Function.StatsROMSetup.count += ct
			src.StatsROMSetup.count += ct
		}
	}
}

func (src *Source) newFrame() {
	// calling newFrame() on stats in a specific order. first the program, then
	// the functions and then the source lines.

	src.Stats.newFrame(nil, nil)
	src.StatsVBLANK.newFrame(nil, nil)
	src.StatsScreen.newFrame(nil, nil)
	src.StatsOverscan.newFrame(nil, nil)
	src.StatsROMSetup.newFrame(nil, nil)

	for _, fn := range src.Functions {
		fn.Stats.newFrame(&src.Stats, nil)
		fn.StatsVBLANK.newFrame(&src.StatsVBLANK, nil)
		fn.StatsScreen.newFrame(&src.StatsScreen, nil)
		fn.StatsOverscan.newFrame(&src.StatsOverscan, nil)
		fn.StatsROMSetup.newFrame(&src.StatsROMSetup, nil)
	}

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, ln := range src.SortedLines.Lines {
		ln.Stats.newFrame(&src.Stats, &ln.Function.Stats)
		ln.StatsVBLANK.newFrame(&src.StatsVBLANK, &ln.Function.StatsVBLANK)
		ln.StatsScreen.newFrame(&src.StatsScreen, &ln.Function.StatsScreen)
		ln.StatsOverscan.newFrame(&src.StatsOverscan, &ln.Function.StatsOverscan)
		ln.StatsROMSetup.newFrame(&src.StatsROMSetup, &ln.Function.StatsROMSetup)
	}
}

func readSourceFile(filename string, pathToROM_nosymlinks string) (*SourceFile, error) {
	var err error

	fl := SourceFile{
		Filename: filename,
	}

	// read file data
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// split files into lines and parse into fragments
	var fp fragmentParser
	for i, s := range strings.Split(string(b), "\n") {
		l := &SourceLine{
			File:         &fl,
			LineNumber:   i + 1,
			Function:     &SourceFunction{Name: UnknownFunction},
			PlainContent: s,
		}
		fl.Lines = append(fl.Lines, l)
		fp.parseLine(l)
	}

	// evaluate symbolic links for the source filenam. pathToROM_nosymlinks has
	// already been processed so the comparison later should work in all
	// instance
	var filename_nosymlinks string
	filename_nosymlinks, err = filepath.EvalSymlinks(filename)
	if err != nil {
		filename_nosymlinks = filename
	}

	if strings.HasPrefix(filename_nosymlinks, pathToROM_nosymlinks) {
		fl.ShortFilename = filename_nosymlinks[len(pathToROM_nosymlinks)+1:]
	} else {
		fl.ShortFilename = filename_nosymlinks
	}

	return &fl, nil
}

func findELF(pathToROM string) *elf.File {
	const (
		elfFile            = "armcode.elf"
		elfFile_older      = "custom2.elf"
		elfFile_jetsetilly = "main.elf"
	)

	// current working directory
	od, err := elf.Open(elfFile)
	if err == nil {
		return od
	}

	// same directory as binary
	od, err = elf.Open(filepath.Join(pathToROM, elfFile))
	if err == nil {
		return od
	}

	// main sub-directory
	od, err = elf.Open(filepath.Join(pathToROM, "main", elfFile))
	if err == nil {
		return od
	}

	// main/bin sub-directory
	od, err = elf.Open(filepath.Join(pathToROM, "main", "bin", elfFile))
	if err == nil {
		return od
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	od, err = elf.Open(filepath.Join(pathToROM, "custom", "bin", elfFile_older))
	if err == nil {
		return od
	}

	// jetsetilly source tree
	od, err = elf.Open(filepath.Join(pathToROM, "arm", elfFile_jetsetilly))
	if err == nil {
		return od
	}

	return nil
}

// ResetStatistics resets all performance statistics.
func (src *Source) ResetStatistics() {
	for i := range src.Functions {
		src.Functions[i].Kernel = InKernelAll
		src.Functions[i].Stats.reset()
		src.Functions[i].StatsVBLANK.reset()
		src.Functions[i].StatsScreen.reset()
		src.Functions[i].StatsOverscan.reset()
		src.Functions[i].StatsROMSetup.reset()
	}
	for i := range src.linesByAddress {
		src.linesByAddress[i].Kernel = InKernelAll
		src.linesByAddress[i].Stats.reset()
		src.linesByAddress[i].StatsVBLANK.reset()
		src.linesByAddress[i].StatsScreen.reset()
		src.linesByAddress[i].StatsOverscan.reset()
		src.linesByAddress[i].StatsROMSetup.reset()
	}
	src.Stats.reset()
	src.StatsVBLANK.reset()
	src.StatsScreen.reset()
	src.StatsOverscan.reset()
	src.StatsROMSetup.reset()
}
