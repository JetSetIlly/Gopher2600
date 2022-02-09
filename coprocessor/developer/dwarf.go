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

type SourceFile struct {
	entry         *dwarf.Entry
	Filename      string
	ShortFilename string
	Lines         []*SourceLine
}

type SourceFunction struct {
	Name        string
	LowAddress  []uint32
	HighAddress []uint32

	// first source line in the function
	FirstLine *SourceLine

	// the number of cycles this function has consumed during the course of the previous frame
	FrameCycles float32

	// working value that will be assigned to FrameCycles on the next television.NewFrame()
	nextFrameCycles float32
}

type SourceLine struct {
	File     *SourceFile
	Function *SourceFunction

	LineNumber int
	Content    string

	// address of first instruction for this source line
	Address uint32

	// the generated assembly for this line. will be empty if line is a comment or otherwise unsused
	Disassembly []string

	// the number of times the line has been responsible for an illegal access.
	IllegalCount int

	// the number of cycles this line has consumed during the course of the previous frame
	FrameCycles float32

	// working value that will be assigned to FrameCycles on the next television.NewFrame()
	nextFrameCycles float32
}

func (ln *SourceLine) String() string {
	return fmt.Sprintf("%s:%d", ln.File.Filename, ln.LineNumber)
}

type Source struct {
	dwrf *dwarf.Data

	// disassembled binary
	Disassembly map[uint64]string

	// compile units are made up of many files. the files and filenames are in
	// the fields below
	compileUnits []*dwarf.Entry

	// all the files in all the compile units
	Files     map[string]*SourceFile
	Filenames []string

	// Functions found in the compile units
	Functions     map[string]*SourceFunction
	FunctionNames []string

	// list of funcions sorted by FrameCycles field
	SortedFunctions SortedFunctions

	// lines of source code found in the compile units
	Lines map[string]*SourceLine

	// list of source lines sorted by FrameCycles field
	SortedLines SortedLines

	// the number of cycles the ARM Program has consumed during the course of the previous frame
	FrameCycles float32

	// working value that will be assigned to FrameCycles on the next television.NewFrame()
	nextFrameCycles float32
}

const elfFile = "armcode.elf"
const elfFile_older = "custom2.elf"

func findELF(pathToROM string) *elf.File {
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
	od, err = elf.Open(filepath.Join(pathToROM, "arm", "main.elf"))
	if err == nil {
		return od
	}

	return nil
}

func newSource(pathToROM string) (*Source, error) {
	src := &Source{
		Disassembly:   make(map[uint64]string),
		Files:         make(map[string]*SourceFile),
		Filenames:     make([]string, 0, 10),
		Functions:     make(map[string]*SourceFunction),
		FunctionNames: make([]string, 0, 10),
		SortedFunctions: SortedFunctions{
			Functions: make([]*SourceFunction, 0, 100),
		},
		Lines: make(map[string]*SourceLine),
		SortedLines: SortedLines{
			Lines: make([]*SourceLine, 0, 100),
		},
	}

	// find objdump file and open it
	elf := findELF(pathToROM)
	if elf == nil {
		return nil, curated.Errorf("dwarf: compiled ELF file not found")
	}
	defer elf.Close()

	var err error

	// get DWARF information from ELF file
	src.dwrf, err = elf.DWARF()
	if err != nil {
		return nil, curated.Errorf("dwarf: %v", err)
	}

	// maps of concrete and abstract functions. we'll link them together after
	// the main loop
	concrete := make(map[dwarf.Offset]*dwarf.Entry)
	abstract := make(map[dwarf.Offset]*dwarf.Entry)

	// read source information
	r := src.dwrf.Reader()
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
			src.compileUnits = append(src.compileUnits, entry)

			r, err := src.dwrf.LineReader(entry)
			if err != nil {
				return nil, err
			}

			// loop through files in the compilation unit. entry 0 is always nil
			for _, f := range r.Files()[1:] {
				sf, err := readSourceFile(f.Name, pathToROM)
				if err != nil {
					logger.Logf("dwarf", "%v", err)
				} else {
					sf.entry = entry
					src.Files[sf.Filename] = sf
					src.Filenames = append(src.Filenames, sf.Filename)
				}
			}

		case dwarf.TagSubprogram:
			var name string
			var abstractOrigin bool
			var lowAddress uint32
			var highAddress uint32
			var inline bool

			for _, fld := range entry.Field {
				switch fld.Attr {
				case dwarf.AttrName:
					name = fld.Val.(string)
				case dwarf.AttrLowpc:
					lowAddress = uint32(fld.Val.(uint64))
				case dwarf.AttrHighpc:
					switch fld.Class {
					case dwarf.ClassConstant:
						// dwarf-4
						highAddress = lowAddress + uint32(fld.Val.(int64))
					case dwarf.ClassAddress:
						// dwarf-2
						highAddress = uint32(fld.Val.(uint64))
					default:
						logger.Logf("dwarf", "unsupported class (%s) for dwarf.AttrHighpc", fld.Class)
					}
				case dwarf.AttrInline:
					inline = true

				case dwarf.AttrAbstractOrigin:
					abstractOrigin = true
				}
			}

			if inline {
				// this will add all inline functions to the abstract map
				abstract[entry.Offset] = entry
			} else if abstractOrigin {
				// concrete inlined functions sometimes have the Subprogram
				// tag. othertimes they have the InlinedSubRoutine tag
				concrete[entry.Offset] = entry
			} else {
				if name != "" && lowAddress != 0x00 && highAddress != 0x00 {
					if fn, ok := src.Functions[name]; !ok {
						var fn SourceFunction
						fn.Name = name
						fn.LowAddress = append(fn.LowAddress, lowAddress)
						fn.HighAddress = append(fn.HighAddress, highAddress)
						src.Functions[name] = &fn
						src.FunctionNames = append(src.FunctionNames, fn.Name)
					} else {
						fn.LowAddress = append(fn.LowAddress, lowAddress)
						fn.HighAddress = append(fn.HighAddress, highAddress)
					}
				}
			}

		case dwarf.TagInlinedSubroutine:
			for _, fld := range entry.Field {
				switch fld.Attr {
				case dwarf.AttrAbstractOrigin:
					// concrete inlined functions sometimes have the Subprogram
					// tag. othertimes they have a "normal" SubProgram tag
					concrete[entry.Offset] = entry
					break // for loop
				}
			}

		default:
		}
	}

	// link concrete functions to abstract functions
	for _, c := range concrete {
		var name string
		var lowAddress uint32
		var highAddress uint32

		for _, fld := range c.Field {
			switch fld.Attr {
			case dwarf.AttrLowpc:
				lowAddress = uint32(fld.Val.(uint64))
			case dwarf.AttrHighpc:
				switch fld.Class {
				case dwarf.ClassConstant:
					// dwarf-4
					highAddress = lowAddress + uint32(fld.Val.(int64))
				case dwarf.ClassAddress:
					// dwarf-2
					highAddress = uint32(fld.Val.(uint64))
				default:
					logger.Logf("dwarf", "unsupported class (%s) for dwarf.AttrHighpc", fld.Class)
				}
			case dwarf.AttrAbstractOrigin:
				if a, ok := abstract[fld.Val.(dwarf.Offset)]; ok {
					for _, fld := range a.Field {
						switch fld.Attr {
						case dwarf.AttrName:
							name = fld.Val.(string)
						}
					}
				} else {
					logger.Logf("dwarf", "abstract function not found for concrete instance\n%#v", c)
				}
			}
		}

		if name != "" && lowAddress != 0x00 && highAddress != 0x00 {
			if fn, ok := src.Functions[name]; !ok {
				var fn SourceFunction
				fn.Name = name
				fn.LowAddress = append(fn.LowAddress, lowAddress)
				fn.HighAddress = append(fn.HighAddress, highAddress)
				src.Functions[name] = &fn
				src.FunctionNames = append(src.FunctionNames, fn.Name)
			} else {
				fn.LowAddress = append(fn.LowAddress, lowAddress)
				fn.HighAddress = append(fn.HighAddress, highAddress)
			}
		}
	}

	// sanity check function map
	for _, fn := range src.Functions {
		if len(fn.LowAddress) != len(fn.HighAddress) {
			return nil, curated.Errorf("dwarf: invalid address range for %s", fn.Name)
		}
		if len(src.Functions) != len(src.FunctionNames) {
			return nil, curated.Errorf("dwarf: unmatched function definitions")
		}

		// find first line of function
		fn.FirstLine, _ = src.findSourceLine(fn.LowAddress[0])

		// add to sorted functions list
		src.SortedFunctions.Functions = append(src.SortedFunctions.Functions, fn)
	}

	// disassemble the program(s)
	for _, p := range elf.Progs {
		o := p.Open()

		pc := p.ProgHeader.Vaddr

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
				return nil, err
			}

			opcode := elf.ByteOrder.Uint16(b)
			disasm := arm7tdmi.Disassemble(opcode)
			src.Disassembly[pc] = fmt.Sprintf("%04x %s", opcode, disasm)

			pc += 2
		}
	}

	// find reference for every line. includes assignment of Asm entries.
	for _, e := range src.compileUnits {
		r, err := src.dwrf.LineReader(e)
		if err != nil {
			continue
		}

		// allocation of asm entries to a SourceLine is a step behind the
		// dwarf.LineEntry. this is because of how dwarf data is stored
		//
		// the instructions associated with a source line include those at
		// addresses between the address given in an dwarf.LineEntry and the
		// address in the next entry
		var prevln *SourceLine
		var prevAddress uint64

		for {
			var entry dwarf.LineEntry

			err := r.Next(&entry)
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Logf("dwarf", "%v", err)
			}

			if src.Files[entry.File.Name] != nil {
				// if prevLn is non-nil then lookup the assembly and add to the
				// SourceLine.Asm array
				if prevln != nil {
					for addr := prevAddress; addr < entry.Address; addr += 2 {
						// append disassembly to source line if it exists in
						// the src.Disassembly map
						if disasm, ok := src.Disassembly[addr]; ok {
							prevln.Disassembly = append(prevln.Disassembly, disasm)

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
							if disasm[5:8] == "BAL" {
								break
							}
						}
					}
				}

				// if the entry is the end of a sequence then assign nil to prevLn
				if entry.EndSequence {
					prevln = nil
				} else {
					// get source line
					prevln = src.Files[entry.File.Name].Lines[entry.Line-1]

					// look up function name
					prevln.Function = src.findFunction(uint32(entry.Address))

					if prevln.Function.Name != UnknownFunction {
						if _, ok := src.Lines[prevln.String()]; !ok {
							src.SortedLines.Lines = append(src.SortedLines.Lines, prevln)
							src.Lines[prevln.String()] = prevln
						}
					}

					// note line for next entry
					prevAddress = entry.Address

				}
			}
		}
	}

	// sort list of filenames and functions
	sort.Strings(src.Filenames)
	sort.Strings(src.FunctionNames)

	// sort source lines
	sort.Sort(src.SortedLines)

	// sort function
	sort.Sort(src.SortedFunctions)

	// log summary
	logger.Logf("dwarf", "identified %d functions in %d compile units", len(src.Functions), len(src.compileUnits))

	return src, nil
}

func (src *Source) findFunction(pc uint32) *SourceFunction {
	for _, fn := range src.Functions {
		for i := range fn.LowAddress {
			if pc >= fn.LowAddress[i] && pc < fn.HighAddress[i] {
				return fn
			}
		}
	}

	return &SourceFunction{Name: UnknownFunction}
}

func (src *Source) findSourceLine(pc uint32) (*SourceLine, error) {
	for _, e := range src.compileUnits {
		r, err := src.dwrf.LineReader(e)
		if err != nil {
			return nil, err
		}

		var entry dwarf.LineEntry
		err = r.SeekPC(uint64(pc), &entry)
		if err != nil {
			if err == dwarf.ErrUnknownPC {
				return nil, nil
			}
			return nil, err
		}

		return src.Files[entry.File.Name].Lines[entry.Line-1], nil
	}

	return nil, nil
}

func (src *Source) execute(pc uint32, ct float32) error {
	for _, e := range src.compileUnits {
		r, err := src.dwrf.LineReader(e)
		if err != nil {
			return err
		}

		var entry dwarf.LineEntry
		err = r.SeekPC(uint64(pc), &entry)
		if err != nil {
			if err == dwarf.ErrUnknownPC {
				return nil
			}
			return err
		}

		// if EndSequence is true then the fields were interested in aren't
		// defined. in other words, we've found the entry but it's not
		// meaningful, so return nil
		if entry.EndSequence {
			return nil
		}

		// increase nextFrameCycles values by count for the program, the line, and the function
		src.nextFrameCycles += ct
		src.Files[entry.File.Name].Lines[entry.Line-1].nextFrameCycles += ct
		src.Files[entry.File.Name].Lines[entry.Line-1].Function.nextFrameCycles += ct

		return nil
	}

	return nil
}

func readSourceFile(filename string, pathToROM string) (*SourceFile, error) {
	// remove superfluous path direction
	filename = filepath.Clean(filename)

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

	if strings.HasPrefix(filename, pathToROM) {
		fl.ShortFilename = filename[len(pathToROM)+1:]
	} else {
		fl.ShortFilename = filename
	}

	return &fl, nil
}

// SortedLines orders all the source lines in order of computationally expense
type SortedLines struct {
	Lines []*SourceLine
}

// Len implements sort.Interface.
func (e SortedLines) Len() int {
	return len(e.Lines)
}

// Less implements sort.Interface.
func (e SortedLines) Less(i int, j int) bool {
	return e.Lines[i].FrameCycles > e.Lines[j].FrameCycles
}

// Swap implements sort.Interface.
func (e SortedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

// SortedFunctions orders all the source lines in order of computationally expense
type SortedFunctions struct {
	Functions []*SourceFunction
}

// Len implements sort.Interface.
func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

// Less implements sort.Interface.
func (e SortedFunctions) Less(i int, j int) bool {
	return e.Functions[i].FrameCycles > e.Functions[j].FrameCycles
}

// Swap implements sort.Interface.
func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}
