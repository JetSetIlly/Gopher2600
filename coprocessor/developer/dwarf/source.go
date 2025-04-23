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
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/logger"
)

// Sentinal error to indicate that the DWARF data isn't supported by the
// package. It might be valid DWARF but we don't want to deal with it
var UnsupportedDWARF = errors.New("unsupported DWARF")

// Cartridge defines the interface to the cartridge required by the source package
type Cartridge interface {
	GetCoProcBus() coprocessor.CartCoProcBus
}

// compile units are made up of many children. for convenience/speed we keep
// track of the children as an index rather than a tree.
type compileUnit struct {
	unit     *dwarf.Entry
	children map[dwarf.Offset]*dwarf.Entry
	address  uint64
}

// Source is created from available DWARF data that has been found in relation
// to and ELF file that looks to be related to the specified ROM.
//
// It is possible for the arrays/map fields to be empty
type Source struct {
	cart Cartridge

	// simplified path to use
	path string

	// ELF sections that help DWARF locate local variables in memory
	debugLoc   *loclistDecoder
	debugFrame *frameSection

	// source is compiled with optimisation
	Optimised bool

	// every compile unit in the dwarf data
	compileUnits []*compileUnit

	// instructions in the source code
	Instructions map[uint64]*SourceInstruction

	// all the files in all the compile units
	Files     map[string]*SourceFile
	Filenames []string

	// as above but indexed by the file's short filename, which is sometimes
	// more useful than the full name
	//
	// short filenames also only include files that are in the same path as the
	// ROM file
	FilesByShortname map[string]*SourceFile
	ShortFilenames   []string

	// functions found in the compile units
	Functions     map[string]*SourceFunction
	FunctionNames []string

	// best guess at what the "main" function is in the program. very often
	// this function will be called "main" and will be easy to discern but
	// sometimes it is named something else and we must figure out as best we
	// can which function it is
	//
	// if no function can be found at all, MainFunction will be a stub entry
	MainFunction *SourceFunction

	// special purpose line used to collate instructions that are outside the
	// loaded ROM and are very likely instructions handled by the "driver". the
	// actual driver function is in the Functions map as normal, under the name
	// given in "const driverFunction"
	DriverSourceLine *SourceLine

	// sorted list of every function in all compile unit
	SortedFunctions SortedFunctions

	// all global variables in all compile units
	GlobalsByAddress map[uint64]*SourceVariable
	SortedGlobals    SortedVariables

	// all local variables in all compile units
	SortedLocals SortedVariablesLocal

	// the highest address of any variable (not just global variables, any
	// variable)
	HighAddress uint64

	// lines of source code found in the compile units. this is a sparse
	// coverage of the total address space
	LinesByAddress map[uint64]*SourceLine

	// sorted list of every source line in all compile units
	SortedLines SortedLines

	// every non-blank line of source code in all compile units
	AllLines AllSourceLines

	// sorted lines filtered by function name
	FunctionFilters []*FunctionFilter

	// profiling for the entire program
	Cycles profiling.Cycles

	// flag to indicate whether the execution profile has changed since it was cleared
	//
	// cheap and easy way to prevent sorting too often - rather than sort after
	// every call to execute(), we can use this flag to sort only when we need
	// to in the GUI.
	//
	// probably not scalable but sufficient for our needs of a single GUI
	// running and using the profiling data for only one reason
	ProfilingDirty bool
}

// NewSource is the preferred method of initialisation for the Source type.
//
// If no ELF file or valid DWARF data can be found in relation to the ROM file
// the function will return nil with an error.
//
// Once the ELF and DWARF file has been identified then Source will always be
// non-nil but with the understanding that the fields may be empty.
func NewSource(romFile string, cart Cartridge, elfFile string, yld YieldAddress) (*Source, error) {
	src := &Source{
		cart:             cart,
		path:             simplifyPath(filepath.Dir(romFile)),
		Instructions:     make(map[uint64]*SourceInstruction),
		Files:            make(map[string]*SourceFile),
		Filenames:        make([]string, 0, 10),
		FilesByShortname: make(map[string]*SourceFile),
		ShortFilenames:   make([]string, 0, 10),
		Functions:        make(map[string]*SourceFunction),
		FunctionNames:    make([]string, 0, 10),
		GlobalsByAddress: make(map[uint64]*SourceVariable),
		SortedGlobals: SortedVariables{
			Variables: make([]*SourceVariable, 0, 100),
		},
		SortedFunctions: SortedFunctions{
			Functions: make([]*SourceFunction, 0, 100),
		},
		LinesByAddress: make(map[uint64]*SourceLine),
		SortedLines: SortedLines{
			Lines: make([]*SourceLine, 0, 100),
		},
		ProfilingDirty: true,
	}

	var ef *elf.File
	var fromCartridge bool
	var err error

	// open ELF file
	if elfFile != "" {
		ef, err = elf.Open(elfFile)
		if err != nil {
			return nil, fmt.Errorf("dwarf: %w", err)
		}

	} else {
		ef, fromCartridge = findELF(romFile)
		if ef == nil {
			return nil, fmt.Errorf("dwarf: compiled ELF file not found")
		}
	}
	defer ef.Close()

	// check existance of DWARF data and the DWARF version before proceeding
	debug_info := ef.Section(".debug_info")
	if debug_info == nil {
		return nil, fmt.Errorf("dwarf: ELF file does not have .debug_info section")
	}
	b, err := debug_info.Data()
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}
	version := ef.ByteOrder.Uint16(b[4:])
	if version != 4 {
		return nil, fmt.Errorf("%w: version %d of DWARF is not supported", UnsupportedDWARF, version)
	}

	// whether ELF file is isRelocatable or not
	isRelocatable := ef.Type&elf.ET_REL == elf.ET_REL

	// sanity checks on ELF data only if we've loaded the file ourselves and
	// it's not from the cartridge.
	if !fromCartridge {
		if ef.FileHeader.Machine != elf.EM_ARM {
			return nil, fmt.Errorf("dwarf: elf file is not ARM")
		}
		if ef.FileHeader.Version != elf.EV_CURRENT {
			return nil, fmt.Errorf("dwarf: elf file is of unknown version")
		}

		// big endian byte order is probably fine but we've not tested it
		if ef.FileHeader.ByteOrder != binary.LittleEndian {
			return nil, fmt.Errorf("dwarf: elf file is not little-endian")
		}

		// we do not permit relocatable ELF files unless it's been supplied by
		// the cartridge. it's not clear what a relocatable ELF file would mean
		// in this context so we just disallow it
		if isRelocatable {
			return nil, fmt.Errorf("dwarf: elf file is relocatable. not permitted for non-ELF cartridges")
		}
	}

	// keeping things simple. only 32bit ELF files supported. 64bit files are
	// probably fine but we've not tested them
	if ef.Class != elf.ELFCLASS32 {
		return nil, fmt.Errorf("dwarf: only 32bit ELF files are supported")
	}

	// no need to continue if ELF file does not have any DWARF data
	dwrf, err := ef.DWARF()
	if err != nil {
		return nil, fmt.Errorf("dwarf: no DWARF data in ELF file")
	}

	// addressAdjustment is the value that is added to the addresses in the
	// DWARF data to adjust them to the correct value for the emulation
	//
	// in the case of the relocatable binaries, such as those provided by the
	// "ELF" cartridge mapper, the value is taken from the ".text" section. this
	// relies on the cartridge mapper supporting the CartCoProcRelocatable
	// interface. the exception with this is if another ELF/DWARF file has been
	// explicitely specified
	//
	// in the case of non-relocatable binaries the value comes from the
	// cartridge mapper if it supports the CartCoProcOrigin interface
	var addressAdjustment uint64

	// cartridge coprocessor
	bus := cart.GetCoProcBus()
	if bus == nil {
		return nil, fmt.Errorf("dwarf: cartridge has no coprocessor to work with")
	}

	// acquire origin addresses and debugging sections according to whether the
	// cartridge is relocatable or not
	if isRelocatable && fromCartridge {
		c, ok := bus.(coprocessor.CartCoProcRelocatable)
		if !ok {
			return nil, fmt.Errorf("dwarf: ELF file is reloctable but the cartridge mapper does not support that")
		}
		if _, o, ok := c.ELFSection(".text"); ok {
			addressAdjustment = uint64(o)
		} else {
			return nil, fmt.Errorf("dwarf: no .text section in ELF file")
		}

		// always create debugFrame and debugLoc sections even when the
		// cartridge doesn't have the corresponding sections. in the case of
		// the loclist section this is definitely needed because even without
		// .debug_loc data we use the loclistDecoder to help decode single
		// address descriptions (which will definitely be present)

		// ignoring the boolean return value because the newFrameSection() will
		// warn about an empty data section
		data, _, _ := c.ELFSection(".debug_frame")
		src.debugFrame, err = newFrameSection(data, ef.ByteOrder, src.cart.GetCoProcBus().GetCoProc(), yld, nil)
		if err != nil {
			logger.Log(logger.Allow, "dwarf", err)
		}

		// ignoring the boolean return value because the newLolistDecoder() will
		// warn about an empty data section
		data, _, _ = c.ELFSection(".debug_loc")
		src.debugLoc, err = newLoclistDecoder(data, ef.ByteOrder, src.cart.GetCoProcBus().GetCoProc())
		if err != nil {
			logger.Log(logger.Allow, "dwarf", err)
		}
	} else {
		var adjust bool

		// if ELF file was manually specified prefer
		if elfFile != "" {
			addressAdjustment = ef.Entry
			adjust = true
		} else {
			if c, ok := bus.(coprocessor.CartCoProcOrigin); ok {
				addressAdjustment = uint64(c.ExecutableOrigin())
				adjust = true
			}
		}

		// create frame section from the raw ELF section
		rel := frameSectionRelocate{
			origin: uint32(addressAdjustment),
		}
		src.debugFrame, err = newFrameSectionFromFile(ef, src.cart.GetCoProcBus().GetCoProc(), yld, &rel)
		if err != nil {
			logger.Log(logger.Allow, "dwarf", err)
		}

		// create loclist section from the raw ELF section
		src.debugLoc, err = newLoclistDecoderFromFile(ef, src.cart.GetCoProcBus().GetCoProc())
		if err != nil {
			logger.Log(logger.Allow, "dwarf", err)
		}

		if adjust {
			// the addressAdjustment needs further adjustment based on the
			// executable section with the lowest address. the assumption here
			// is that the list of sections are in address order lowest to
			// highest
			for _, sec := range ef.Sections {
				if sec.Flags&elf.SHF_EXECINSTR == elf.SHF_EXECINSTR {
					addressAdjustment -= sec.Addr
					break // for loop
				}
			}
		}
	}

	// log address adjustment value. how the value was arrived at is slightly
	// different depending on whether the ELF file relocatable or not
	if addressAdjustment == 0 {
		logger.Log(logger.Allow, "dwarf", "address adjustment not required")
	} else {
		logger.Logf(logger.Allow, "dwarf", "using address adjustment: %#x", int(addressAdjustment))
	}

	// disassemble every word in the ELF file using the cartridge coprocessor interface
	//
	// we could traverse of the progs array of the file here but some ELF files
	// that we want to support do not have any program headers. we get the same
	// effect by traversing the Sections array and ignoring any section that
	// does not have the EXECINSTR flag
	for _, sec := range ef.Sections {
		if sec.Flags&elf.SHF_EXECINSTR != elf.SHF_EXECINSTR {
			continue // for loop
		}

		// section data
		var data []byte
		data, err = sec.Data()
		if err != nil {
			return nil, fmt.Errorf("dwarf: %w", err)
		}

		// origin is section address adjusted by both the executable origin and
		// the adjustment amount previously recorded
		origin := sec.Addr + addressAdjustment

		// disassemble section
		_ = arm.StaticDisassemble(arm.StaticDisassembleConfig{
			Data:      data,
			Origin:    uint32(origin),
			ByteOrder: ef.ByteOrder,
			Callback: func(e arm.DisasmEntry) {
				src.Instructions[uint64(e.Addr)] = &SourceInstruction{
					Addr:   e.Addr,
					opcode: uint32(e.OpcodeHi)<<16 | uint32(e.Opcode),
					size:   e.Size(),
					Disasm: e,
				}
			},
		})
	}

	bld, err := newBuild(dwrf, src.debugLoc, src.debugFrame)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// compile units are made up of many files. the files and filenames are in
	// the fields below
	r := dwrf.Reader()

	// loop through file and collate compile units
	for {
		e, err := r.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // for loop
			}
			return nil, fmt.Errorf("dwarf: %w", err)
		}
		if e == nil {
			break // for loop
		}
		if e.Offset == 0 {
			continue // for loop
		}

		switch e.Tag {
		case dwarf.TagCompileUnit:
			unit := &compileUnit{
				unit:     e,
				children: make(map[dwarf.Offset]*dwarf.Entry),
				address:  addressAdjustment,
			}

			fld := e.AttrField(dwarf.AttrLowpc)
			if fld != nil {
				unit.address = addressAdjustment + fld.Val.(uint64)
			}

			// assuming DWARF never has duplicate compile unit entries
			src.compileUnits = append(src.compileUnits, unit)

			r, err := dwrf.LineReader(e)
			if err != nil {
				return nil, fmt.Errorf("dwarf: %w", err)
			}

			// loop through files in the compilation unit. entry 0 is always nil
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

			fld = e.AttrField(dwarf.AttrProducer)
			if fld != nil {
				producer := fld.Val.(string)

				if strings.HasPrefix(producer, "GNU") {
					// check optimisation directive
					if strings.Contains(producer, " -O") {
						src.Optimised = true
					}
				}
			}

		default:
			if len(src.compileUnits) == 0 {
				return nil, fmt.Errorf("dwarf: bad data: no compile unit tag")
			}
			src.compileUnits[len(src.compileUnits)-1].children[e.Offset] = e
		}
	}

	// log optimisation message as appropriate
	if src.Optimised {
		logger.Logf(logger.Allow, "dwarf", "source compiled with optimisation")
	}

	// build functions from DWARF data
	err = bld.buildFunctions(src, addressAdjustment)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// complete function list with stubs for functions where we don't have any
	// DWARF data (but do have symbol data)
	resolveSymbols(bus, src, ef)

	// sanity check of functions list
	if len(src.Functions) != len(src.FunctionNames) {
		return nil, fmt.Errorf("dwarf: unmatched function definitions")
	}

	// read source lines
	err = allocateSourceLines(src, dwrf, addressAdjustment)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// assign functions to every source line
	assignFunctionToSourceLines(src)

	// assemble sorted functions list
	for _, fn := range src.Functions {
		src.SortedFunctions.Functions = append(src.SortedFunctions.Functions, fn)
	}

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
	for _, ln := range src.LinesByAddress {
		if _, ok := observed[ln]; !ok {
			observed[ln] = true
			src.SortedLines.Lines = append(src.SortedLines.Lines, ln)
		}
	}

	// build types
	err = bld.buildTypes(src)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// build variables
	if relocatable, ok := bus.(coprocessor.CartCoProcRelocatable); ok {
		err = bld.buildVariables(src, ef, relocatable, addressAdjustment)
	} else {
		err = bld.buildVariables(src, ef, nil, addressAdjustment)
	}
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// sort list of filenames and functions
	sort.Strings(src.Filenames)
	sort.Strings(src.ShortFilenames)

	// sort lines by function and number. sort is stable so we can do this in
	// two passes
	src.SortedLines.Sort(SortLinesFunction, false, false, false, profiling.FocusAll)
	src.SortedLines.Sort(SortLinesNumber, false, false, false, profiling.FocusAll)

	// sorted functions
	src.SortedFunctions.Sort(SortFunctionsName, false, false, false, profiling.FocusAll)
	sort.Strings(src.FunctionNames)

	// sorted variables
	for _, g := range bld.globals {
		src.GlobalsByAddress[g.resolve(nil).address] = g
		src.SortedGlobals.Variables = append(src.SortedGlobals.Variables, g)
	}

	for _, l := range bld.locals {
		src.SortedLocals.Variables = append(src.SortedLocals.Variables, l)
	}
	sort.Sort(src.SortedGlobals)
	sort.Sort(src.SortedLocals)

	// add children to global and local variables
	addVariableChildren(src)

	// update global variables
	src.UpdateGlobalVariables()

	// determine highest address occupied by the program
	findHighAddress(src)

	// find entry function to the program
	findEntryFunction(src)

	// log summary
	logger.Logf(logger.Allow, "dwarf", "identified %d functions in %d compile units", len(src.Functions), len(src.compileUnits))
	logger.Logf(logger.Allow, "dwarf", "%d global variables", len(src.SortedGlobals.Variables))
	logger.Logf(logger.Allow, "dwarf", "%d local variable (loclists)", len(src.SortedLocals.Variables))
	logger.Logf(logger.Allow, "dwarf", "high address (%08x)", src.HighAddress)

	return src, nil
}

func allocateSourceLines(src *Source, dwrf *dwarf.Data, addressAdjustment uint64) error {
	var addressConflicts int

	for _, e := range src.compileUnits {
		// read every line in the compile unit
		r, err := dwrf.LineReader(e.unit)
		if err != nil {
			return err
		}

		var le dwarf.LineEntry
		for {
			err := r.Next(&le)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // line entry for loop. will continue with compile unit loop
				}
				return err
			}

			// check that source file has been loaded
			if src.Files[le.File.Name] == nil {
				logger.Logf(logger.Allow, "dwarf", "file not available for linereader: %s", le.File.Name)
				break // line entry for loop. will continue with compile unit loop
			}
			if le.Line-1 > src.Files[le.File.Name].Content.Len() {
				logger.Logf(logger.Allow, "dwarf", "current source is unrelated to ELF/DWARF data (number of lines)")
				break // line entry for loop. will continue with compile unit loop
			}

			ln := src.Files[le.File.Name].Content.Lines[le.Line-1]

			// start and end address of line entry
			var startAddr, endAddr uint64
			startAddr = le.Address + addressAdjustment

			// end address determined by peeking at the next entry
			p := r.Tell()
			var peek dwarf.LineEntry
			err = r.Next(&peek)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // line entry for loop. will continue with compile unit loop
				}
				return err
			}
			endAddr = peek.Address + addressAdjustment
			r.Seek(p)

			// sanity check start/end address
			if startAddr > endAddr {
				if le.EndSequence {
					continue
				} else {
					return fmt.Errorf("dwarf: allocate source line: start address (%08x) is after end address (%08x)", startAddr, endAddr)
				}
			}

			// add breakpoint and instruction information to the source line
			if ln != nil && endAddr-startAddr > 0 {
				// add instruction to source line and add source line to linesByAddress
				for addr := startAddr; addr < endAddr; addr++ {
					// look for address in list of source instructions
					if ins, ok := src.Instructions[addr]; ok {

						// add instruction to the list for the source line
						ln.Instruction = append(ln.Instruction, ins)

						// link source line to instruction
						ins.Line = ln

						// add source line to list of lines by address if the
						// address has not been allocated a line already
						if x := src.LinesByAddress[addr]; x == nil {
							src.LinesByAddress[addr] = ln
						} else {
							addressConflicts++
						}

						// advance address value by opcode size. reduce value by
						// one because the loop increment advances by one
						// already (which will always apply even if there is no
						// instruction for the address)
						addr += uint64(ins.size) - 1
					}
				}
			}
		}
	}

	for _, e := range src.compileUnits {
		// read every line in the compile unit
		r, err := dwrf.LineReader(e.unit)
		if err != nil {
			return err
		}

		var le dwarf.LineEntry
		for {
			err := r.Next(&le)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // line entry for loop. will continue with compile unit loop
				}
				return err
			}

			// no need to check whether source file has been loaded because
			// we've already checked that on the previous LineReader run

			// add breakpoint address to the correct line
			if le.IsStmt {
				addr := le.Address + addressAdjustment
				ln := src.LinesByAddress[addr]
				if ln != nil {
					ln.BreakAddresses = append(ln.BreakAddresses, uint32(addr))
				}
			}
		}
	}

	if addressConflicts > 0 {
		logger.Logf(logger.Allow, "dwarf", "address conflicts when allocating to source lines: %d", addressConflicts)
	}

	return nil
}

// add children to global and local variables
func addVariableChildren(src *Source) {
	for _, g := range src.SortedGlobals.Variables {
		g.addVariableChildren(src.debugLoc)
	}

	for _, l := range src.SortedLocals.Variables {
		l.addVariableChildren(src.debugLoc)
	}
}

func assignFunctionToSourceLines(src *Source) {
	// for every source line in every file find the function with the smallest
	// range in which any of the instructions for the line falls in any of the
	// functions possible ranges
	for _, sf := range src.Files {
		for _, ln := range sf.Content.Lines {
			// move on if there are no instructions for the line
			if len(ln.Instruction) > 0 {
				var candidateFunction *SourceFunction
				var rangeSize uint64

				// maximise the range size so that the first comparison will
				// always succeed (the comparison is "less than rangeSize")
				rangeSize = ^uint64(0)

				for _, ins := range ln.Instruction {
					addr := uint64(ins.Addr)
					for _, fn := range src.Functions {
						for _, r := range fn.Range {
							if addr >= r.Start && addr <= r.End {
								if r.End-r.Start < rangeSize {
									rangeSize = r.End - r.Start
									candidateFunction = fn
									break // range loop
								}
							}
						}
					}
				}

				// commit to candidate function
				ln.Function = candidateFunction
			}
		}
	}

	// any source lines without a function assigned to it is allocated the
	// function of the preceeding line
	for _, sf := range src.Files {
		var currentFunction *SourceFunction
		for _, ln := range sf.Content.Lines {
			if ln.Function == nil {
				ln.Function = currentFunction
			} else {
				currentFunction = ln.Function
			}
		}
	}
}

// assign source lines to a function
func assignFunctionToSourceLines_old(src *Source) {
	// for each line in a file compare the address of the first instruction for
	// the line to each range in every function. the function with the smallest
	// range is the function the line belongs to
	for _, sf := range src.Files {
		for _, ln := range sf.Content.Lines {
			if len(ln.Instruction) > 0 {
				var candidateFunction *SourceFunction
				var rangeSize uint64
				rangeSize = ^uint64(0)

				addr := uint64(ln.Instruction[0].Addr)
				for _, fn := range src.Functions {
					for _, r := range fn.Range {
						if addr >= r.Start && addr <= r.End {
							if r.End-r.Start < rangeSize {
								rangeSize = r.End - r.Start
								candidateFunction = fn
								break // range loop
							}
						}
					}
				}

				// we may sometimes reach the end of a loop without having found a corresponding function
				if candidateFunction != nil {
					ln.Function = candidateFunction
				}
			}
		}
	}

	for _, sf := range src.Files {
		var currentFunction *SourceFunction
		for _, ln := range sf.Content.Lines {
			if ln.Function == nil {
				ln.Function = currentFunction
			} else {
				currentFunction = ln.Function
			}
		}
	}
}

// find entry function to the program
func findEntryFunction(src *Source) {
	// TODO: this is a bit of ARM specific knowledge that should be removed
	addr, _ := src.cart.GetCoProcBus().GetCoProc().Register(15)
	if ln, ok := src.LinesByAddress[uint64(addr)]; ok {
		src.MainFunction = ln.Function
		return
	}

	// use function called "main" if it's present. we could add to this list
	// other likely names but this would depend on convention, which doesn't
	// exist yet (eg. elf_main)
	if fn, ok := src.Functions["main"]; ok {
		src.MainFunction = fn
		return
	}

	// assume the function of the first line in the source is the entry
	// function
	for _, ln := range src.SortedLines.Lines {
		if len(ln.Instruction) > 0 {
			src.MainFunction = ln.Function
			break
		}
	}
	if src.MainFunction != nil {
		return
	}

	// if no function can be found for some reason then a stub entry is created
	src.MainFunction = CreateStubLine(nil).Function
}

// determine highest address occupied by the program
func findHighAddress(src *Source) {
	src.HighAddress = 0

	for _, g := range src.SortedGlobals.Variables {
		a := g.resolve(nil).address + uint64(g.Type.Size)
		if a > src.HighAddress {
			src.HighAddress = a
		}
	}

	for _, f := range src.Functions {
		for _, r := range f.Range {
			if r.End > src.HighAddress {
				src.HighAddress = r.End
			}
		}
	}
}

// regular expressions used by resolveSymbols() to test whether the stubs
// should be added or whether they are compiler artefacts
var (
	mangledCppNames        *regexp.Regexp
	compilerGeneratedNames *regexp.Regexp
)

// initialisation of regular expressions used by resolveSymbols()
func init() {
	mangledCppNames = regexp.MustCompile(`^(?:_Z[\w\d_]+|\?[^\s@]+\@.*)$`)
	compilerGeneratedNames = regexp.MustCompile(`^_GLOBAL__(sub_I|D|I)_[\w\d_]+$`)
}

// CartridgeFunctionSymbol is implemented by cartridges that can supply missing
// symbol information about functions
type CartridgeFunctionSymbol interface {
	GetFunctionRange(name string) (uint64, uint64, bool)
}

// add function stubs for functions without DWARF data. we do this *after*
// we've looked for functions in the DWARF data (via the line reader) because
// it appears that not every function will necessarily have a symbol and it's
// easier to handle the adding of stubs *after* the the line reader. it does
// mean though that we need to check that a function has not already been added
func resolveSymbols(cart coprocessor.CartCoProcBus, src *Source, ef *elf.File) error {
	// we'll use the cartridge to resolve symbols if at all possible
	cartSymbols, _ := cart.(CartridgeFunctionSymbol)

	// all the symbols in the ELF file
	syms, err := ef.Symbols()
	if err != nil {
		return err
	}

	type fn struct {
		name string
		rng  SourceRange
	}

	var symbolTableFunctions []fn

	// the functions from the symbol table
	for _, s := range syms {
		if mangledCppNames.MatchString(s.Name) {
			continue // for loop
		}

		if compilerGeneratedNames.MatchString(s.Name) {
			continue // for loop
		}

		typ := s.Info & 0x0f
		if typ == 0x02 {
			// align address
			// TODO: this is a bit of ARM specific knowledge that should be removed
			a := s.Value & 0xfffffffe
			symbolTableFunctions = append(symbolTableFunctions, fn{
				name: s.Name,
				rng: SourceRange{
					Start: a,
					End:   a + s.Size - 1,
				},
			})
		} else if cartSymbols != nil {
			a, b, ok := cartSymbols.GetFunctionRange(s.Name)
			if ok {
				symbolTableFunctions = append(symbolTableFunctions, fn{
					name: s.Name,
					rng: SourceRange{
						Start: a,
						End:   b,
					},
				})
			}
		}
	}

	for _, fn := range symbolTableFunctions {
		if _, ok := src.Functions[fn.name]; !ok {
			// chop off suffix from symbol table name if there is one. not sure
			// about this but it neatens things up for the cases I've seen so
			// far
			fn.name = strings.Split(fn.name, ".")[0]

			stubFn := &SourceFunction{
				Name: fn.name,
			}
			stubFn.Range = append(stubFn.Range, fn.rng)
			stubFn.DeclLine = CreateStubLine(stubFn)

			// add stub function to list of functions but not if the function
			// covers an area that has already been seen
			addFunction := true

			// process all addresses in range, skipping any addresses that we
			// already know about from the DWARF data
			for a := fn.rng.Start; a <= fn.rng.End; a++ {
				if _, ok := src.LinesByAddress[a]; !ok {
					src.LinesByAddress[a] = CreateStubLine(stubFn)
				} else {
					addFunction = false
					break
				}
			}

			if addFunction {
				if _, ok := src.Functions[stubFn.Name]; !ok {
					src.Functions[stubFn.Name] = stubFn
					src.FunctionNames = append(src.FunctionNames, stubFn.Name)
				}
			}
		}
	}

	// add driver function
	driverFn := &SourceFunction{
		Name: DriverFunctionName,
	}
	src.Functions[DriverFunctionName] = driverFn
	src.FunctionNames = append(src.FunctionNames, DriverFunctionName)
	src.DriverSourceLine = CreateStubLine(driverFn)

	return nil
}

func readSourceFile(filename string, path string, all *AllSourceLines) (*SourceFile, error) {
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
		ln := &SourceLine{
			File:       &fl,
			LineNumber: i + 1, // counting from one
			Function: &SourceFunction{
				Name: stubIndicator,
			},
			PlainContent: s,
		}
		fl.Content.Lines = append(fl.Content.Lines, ln)
		fp.parseLine(ln)

		// update max line width
		if len(s) > fl.Content.MaxLineWidth {
			fl.Content.MaxLineWidth = len(s)
		}

		if len(strings.TrimSpace(s)) > 0 {
			all.lines = append(all.lines, ln)
		}
	}

	// evaluate symbolic links for the source filename. path has already been
	// processed so the comparison later should work in all instance
	filename = simplifyPath(filename)
	fl.ShortFilename = longestPath(filename, path)

	return &fl, nil
}

func findELF(romFile string) (*elf.File, bool) {
	// try the ROM file itself. it might be an ELF file
	ef, err := elf.Open(romFile)
	if err == nil {
		return ef, true
	}

	// the file is not an ELF file so the remainder of the function will work
	// with the path component of the ROM file only
	pathToROM := filepath.Dir(romFile)

	filenames := []string{
		"armcode.elf",
		"custom2.elf",
		"main.elf",
		"ACE_debugging.elf",
	}

	subpaths := []string{
		"",
		"main",
		filepath.Join("main", "bin"),
		filepath.Join("custom", "bin"),
		"arm",
	}

	for _, p := range subpaths {
		for _, f := range filenames {
			ef, err = elf.Open(filepath.Join(pathToROM, p, f))
			if err == nil {
				return ef, false
			}
		}
	}

	return nil, false
}

// FindSourceLine returns line entry for the address. Returns nil if the
// address has no source line.
func (src *Source) FindSourceLine(addr uint32) *SourceLine {
	return src.LinesByAddress[uint64(addr)]
}

// UpdateGlobalVariables using the current state of the emulated coprocessor.
// Local variables are updated when coprocessor yields (see OnYield() function)
func (src *Source) UpdateGlobalVariables() {
	for _, varb := range src.SortedGlobals.Variables {
		varb.Update()
	}
}

// GetLocalVariables retuns the list of local variables for the supplied
// address. Local variables will not be updated.
func (src *Source) GetLocalVariables(ln *SourceLine, addr uint32) []*SourceVariableLocal {
	var locals []*SourceVariableLocal
	var chosenLocal *SourceVariableLocal

	// choose function that covers the smallest (most specific) range in which startAddr
	// appears
	chosenSize := ^uint64(0)

	// function to add chosen local variable to the yield
	commitChosen := func() {
		locals = append(locals, chosenLocal)
		chosenLocal = nil
		chosenSize = ^uint64(0)
	}

	// there's an assumption here that SortedLocals is sorted by variable name
	for _, local := range src.SortedLocals.Variables {
		// append chosen local variable. comparing variable name (rather than
		// declartion line) even though there may be multiple variables in
		// differnt places with the same name. this is because we don't want to
		// see multiple entries of a variable name
		if chosenLocal != nil && chosenLocal.Name != local.Name {
			commitChosen()
		}

		// ignore variables that are not declared to be in the same function as the break
		// line. this can happen for inlined functions when function ranges overlap
		if local.DeclLine.Function == ln.Function {
			if local.Range.InRange(uint64(addr)) {
				if local.Range.Size() < chosenSize || (local.IsValid() && !chosenLocal.IsValid()) {
					chosenLocal = local
					chosenSize = local.Range.Size()
				}
			}
		}
	}

	// append chosen local variable
	if chosenLocal != nil {
		commitChosen()
	}

	return locals
}

// FramebaseCurrent returns the current framebase value
func (src *Source) FramebaseCurrent(derive io.Writer) (uint64, error) {
	return src.debugFrame.resolveFramebase(derive)
}

func simplifyPath(path string) string {
	nosymlinks, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return nosymlinks
}

func longestPath(a, b string) string {
	c := strings.Split(a, string(os.PathSeparator))
	d := strings.Split(b, string(os.PathSeparator))

	m := len(d)
	if len(c) < m {
		return a
	}

	var i int
	for i < m && c[i] == d[i] {
		i++
	}

	return filepath.Join(c[i:]...)
}

// SourceLineByAddr returns the source line for an instruction address. If there
// is no corresponding source line then a stub is returned.
func (src *Source) SourceLineByAddr(addr uint32) *SourceLine {
	ln := src.LinesByAddress[uint64(addr)]
	if ln == nil {
		ln = CreateStubLine(nil)
	}
	return ln
}
