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

// SourceFile is a single source file indentified by the DWARF data.
type SourceFile struct {
	Filename      string
	ShortFilename string
	Lines         []*SourceLine
}

// SourceFunction is a single function identified by the DWARF data.
type SourceFunction struct {
	Name string

	// first source line for each instance of the function
	DeclLine *SourceLine

	// cycle statisics related to the SourceFunction
	Stats SourceStats
}

// SourceDisasm is a single disassembled intruction from the ELF binary.
type SourceDisasm struct {
	Addr        uint32
	Opcode      uint16
	Instruction string
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

	// the content of the line
	Content string

	// the function the line of source can be found within
	Function *SourceFunction

	// the generated assembly for this line. will be empty if line is a comment or otherwise unsused
	Disassembly []*SourceDisasm

	// the number of times the line has been responsible for an illegal access.
	IllegalCount int

	// cycle statisics related to the SourceLine
	Stats SourceStats
}

func (ln *SourceLine) String() string {
	return fmt.Sprintf("%s:%d", ln.File.Filename, ln.LineNumber)
}

// compile units are made up of many children. for convenience/speed we keep
// track of the children as an index rather than a tree.
type compileUnit struct {
	unit                    *dwarf.Entry
	children                map[dwarf.Offset]*dwarf.Entry
	unsupportedOptimisation string
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

	// Functions found in the compile units
	Functions     map[string]*SourceFunction
	FunctionNames []string

	// list of funcions sorted by FrameCycles field
	SortedFunctions SortedFunctions

	// lines of source code found in the compile units
	Lines map[uint32]*SourceLine

	// list of source lines sorted by FrameCycles field
	SortedLines SortedLines

	// sorted lines filtered by function name
	FunctionFilter        string
	FunctionFilteredLines SortedLines

	// numer of cycles in the entire program represented by the source since the last update
	cyclesCount float32
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
		Disassembly:   make(map[uint64]*SourceDisasm),
		Files:         make(map[string]*SourceFile),
		Filenames:     make([]string, 0, 10),
		Functions:     make(map[string]*SourceFunction),
		FunctionNames: make([]string, 0, 10),
		SortedFunctions: SortedFunctions{
			Functions: make([]*SourceFunction, 0, 100),
		},
		Lines: make(map[uint32]*SourceLine),
		SortedLines: SortedLines{
			Lines: make([]*SourceLine, 0, 100),
		},
		FunctionFilter: "",
		FunctionFilteredLines: SortedLines{
			Lines: make([]*SourceLine, 0, 100),
		},
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
		return src, curated.Errorf("dwarf: %v", err)
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
				return src, curated.Errorf("dwarf: %v", err)
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

			// working source line and address. we'll proces these at the next line entry
			if src.Files[le.File.Name] == nil {
				workingSourceLine = nil
				continue // for loop
			}

			// if the entry is the end of a sequence then assign nil to workingSourceLine
			if le.EndSequence {
				workingSourceLine = nil
				continue // for loop
			}

			// if workingSourceLine is non-nil then lookup the assembly for each
			// instruction between the address of workingSourceLine and the address
			// of the current line entry
			if workingSourceLine != nil {
				for addr := workingAddress; addr < le.Address; addr += 2 {
					// look for address in disassembly
					if d, ok := src.Disassembly[addr]; ok {
						// add disassembly to the list of instructions for the workingSourceLine
						workingSourceLine.Disassembly = append(workingSourceLine.Disassembly, d)

						// associate the address with the workingSourceLine
						src.Lines[uint32(addr)] = workingSourceLine

						// special handling of unconditional branch
						//
						// we sometimes see this at the end of a code block
						// and is important to check for so that the for
						// doesn't go wandering into the weeds.
						//
						// as far as I can tell there is no other way of
						// checking to see if the end of block of
						// instructions has been reached
						//
						// TODO: improve detection of end of code block during DWARF parsing
						if len(d.Instruction) > 3 && d.Instruction[:3] == "BAL" {
							break // for addr loop
						}
					}
				}

				// we've done with working source line
				workingSourceLine = nil
			}

			// new working source line and working address
			workingSourceLine = src.Files[le.File.Name].Lines[le.Line-1]
			workingAddress = le.Address

			// find function name for line entry
			foundFunc, err := bld.findFunction(le.Address)
			if err != nil {
				return nil, curated.Errorf("dwarf: %v", err)
			}
			if foundFunc == nil {
				continue // for loop
			}

			var srcFunc *SourceFunction

			if f, ok := src.Files[foundFunc.filename]; ok {
				srcFunc = &SourceFunction{
					Name:     foundFunc.name,
					DeclLine: f.Lines[foundFunc.linenum-1],
				}
			}
			if srcFunc == nil {
				continue // for loop
			}

			// if function can't be found then log error and continue
			if srcFunc.Name == UnknownFunction {
				logger.Logf("dwarf", "no function for line entry: %s", workingSourceLine.String())
				workingSourceLine = nil
				continue // for loop
			}

			// if function already exists use that function instance.
			// otherwise, add the function to the map and the list of function
			// names
			if _, ok := src.Functions[srcFunc.Name]; ok {
				srcFunc = src.Functions[srcFunc.Name]
			} else {
				src.Functions[srcFunc.Name] = srcFunc
				src.FunctionNames = append(src.FunctionNames, srcFunc.Name)
			}

			// associate function with workingSourceLine
			workingSourceLine.Function = srcFunc
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
	for _, ln := range src.Lines {
		if _, ok := observed[ln]; !ok {
			observed[ln] = true
			src.SortedLines.Lines = append(src.SortedLines.Lines, ln)
		}
	}

	// sort sorted lines
	sort.Sort(src.SortedLines)

	// sorted functions
	sort.Sort(src.SortedFunctions)

	// log summary
	logger.Logf("dwarf", "identified %d functions in %d compile units", len(src.Functions), len(src.compileUnits))

	return src, nil
}

// returns function or error. if error is nil then SourceFunction will be valid.
//
// if function cannot be found the SourceFunction.Name will be UnknownFunction.
// test for that rather than nil.
//
// this function is inherently slow and is only really useful for one shot
// lookups, very occassionaly. during the preperation of the Source instance
// the findFunction() in the build type is preferred.
func (src *Source) findFunction(addr uint64) (*SourceFunction, error) {
	found := &SourceFunction{Name: UnknownFunction}
	return found, nil

	// resolveAbstract is a helper function that creates a SourceFunction and
	// assigns it to the ret variable (created above). it is commone to both
	// Subprogram and InlinedSubroutine tagged entries
	//
	// significantly it handles the AbstractOrigin attribute correctly
	resolveAbstract := func(entry *dwarf.Entry) error {
		var err error

		// read abstract entry (using a different reader) if appropriate tag is found
		fld := entry.AttrField(dwarf.AttrAbstractOrigin)
		if fld != nil {
			r := src.dwrf.Reader()
			r.Seek(fld.Val.(dwarf.Offset))
			entry, err = r.Next()
			if err != nil {
				return err
			}
		}

		// from here similar to TagSubprogram

		// the list of files for the compile unit. where we get the files
		// from depends on whether current entry has an abstract origin or
		// not
		var files []*dwarf.LineFile

		// check which compile unit the abstract entry is in and
		// reinitialise the files array
		for _, u := range src.compileUnits {
			if _, ok := u.children[entry.Offset]; ok {
				lr, err := src.dwrf.LineReader(u.unit)
				if err != nil {
					return err
				}
				files = lr.Files()
				break
			}
		}

		// name of entry
		fld = entry.AttrField(dwarf.AttrName)
		if fld == nil {
			return nil
		}
		name := fld.Val.(string)

		// declaration file
		fld = entry.AttrField(dwarf.AttrDeclFile)
		if fld == nil {
			return nil
		}
		filenum := fld.Val.(int64)

		// declaration line
		fld = entry.AttrField(dwarf.AttrDeclLine)
		if fld == nil {
			return nil
		}
		linenum := fld.Val.(int64)

		// prepare return value
		filename := files[filenum].Name
		if fn, ok := src.Files[filename]; ok {
			found = &SourceFunction{
				Name:     name,
				DeclLine: fn.Lines[linenum-1],
			}
		}

		return nil
	}

	r := src.dwrf.Reader()
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
		case dwarf.TagInlinedSubroutine:
			// the address range to match against for inlined subroutines is a
			// little more involved because these entries can have either a
			// low/high field or a ranges field
			//
			// assumption: if there is a low attribute there should be a high
			// field and there won't be a ranges field
			fld := entry.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				var low uint64
				var high uint64

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

				rngs, err := src.dwrf.Ranges(entry)
				if err != nil {
					return nil, err
				}

				match := false
				for _, r := range rngs {
					if addr >= r[0] && addr < r[1] {
						match = true
						break
					}
				}
				if !match {
					continue // for loop
				}
			}

			err = resolveAbstract(entry)
			if err != nil {
				return nil, err
			}

		case dwarf.TagSubprogram:
			// check address against low/high fields. compare to
			// InlinedSubroutines where address range can be given by either
			// low/high fields OR a Range field. for Subprograms, there is
			// never a Range field.

			var low uint64
			var high uint64

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

			err = resolveAbstract(entry)
			if err != nil {
				return nil, err
			}
		}
	}

	return found, nil
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
			return nil, err
		}

		if src.Files[entry.File.Name] == nil {
			return nil, fmt.Errorf("%s not in list of files", entry.File.Name)
		}

		return src.Files[entry.File.Name].Lines[entry.Line-1], nil
	}

	return nil, nil
}

func (src *Source) execute(addr uint32, ct float32) {
	line, ok := src.Lines[addr]
	if ok {
		src.cyclesCount += ct
		line.Stats.cyclesCount += ct
		line.Function.Stats.cyclesCount += ct
	}
}

func (src *Source) newFrame() {
	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, l := range src.SortedLines.Lines {
		l.Stats.newFrame(src.cyclesCount)
	}

	for _, f := range src.Functions {
		f.Stats.newFrame(src.cyclesCount)
	}

	src.cyclesCount = 0
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

	// split files into lines
	for i, s := range strings.Split(string(b), "\n") {
		fl.Lines = append(fl.Lines, &SourceLine{
			File:       &fl,
			LineNumber: i + 1,
			Content:    s,
			Function:   &SourceFunction{Name: UnknownFunction},
		})
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
