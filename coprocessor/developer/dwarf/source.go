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
	"debug/elf"
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

// Source is created from available DWARF data that has been found in relation
// to an ELF file that looks to be related to the specified ROM.
//
// It is possible for the arrays/map fields to be empty
type Source struct {
	cart coprocessor.CartCoProcBus

	// simplified path to use
	path string

	// ELF sections that help DWARF locate local variables in memory
	debugLoc   *loclistDecoder
	debugFrame *frameSection

	// instructions in the source code
	instructions map[uint64]*SourceInstruction

	// source is compiled with optimisation
	Optimisation bool

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
func NewSource(cart coprocessor.CartCoProcBus, romFile string, elfFile string, yld YieldAddress) (*Source, error) {
	src := &Source{
		cart:             cart,
		path:             simplifyPath(filepath.Dir(romFile)),
		instructions:     make(map[uint64]*SourceInstruction),
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

	ef, ok := cart.(coprocessor.CartCoProcELF)
	if !ok {
		if elfFile == "" {
			ef = findELF(romFile)
			if ef == nil {
				return nil, fmt.Errorf("dwarf: cannot obtain elf information")
			}
		} else {
			f, err := elf.Open(elfFile)
			if err != nil {
				return nil, fmt.Errorf("dwarf: %w", err)
			}
			ef = &elfShim{ef: f}
		}
	}

	dwrf, err := ef.DWARF()
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// check the DWARF version before proceeding
	debug_info, _ := ef.Section(".debug_info")
	if debug_info == nil {
		return nil, fmt.Errorf("dwarf: ELF file does not have .debug_info section")
	}
	version := ef.ByteOrder().Uint16(debug_info[4:])
	if version != 4 {
		return nil, fmt.Errorf("%w: version %d of DWARF is not supported", UnsupportedDWARF, version)
	}

	// ignoring the boolean return value because the newFrameSection() will
	// warn about an empty data section
	data, _ := ef.Section(".debug_frame")
	src.debugFrame, err = newFrameSection(data, ef.ByteOrder(), cart.GetCoProc(), yld, nil)
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err)
	}

	// ignoring the boolean return value because the newLolistDecoder() will
	// warn about an empty data section
	data, _ = ef.Section(".debug_loc")
	src.debugLoc, err = newLoclistDecoder(data, ef.ByteOrder(), cart.GetCoProc())
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err)
	}

	// disassemble every word in all executable sections
	for _, n := range ef.ExecutableSections() {
		data, origin := ef.Section(n)
		_ = arm.StaticDisassemble(arm.StaticDisassembleConfig{
			Data:      data,
			Origin:    origin,
			ByteOrder: ef.ByteOrder(),
			Callback: func(e arm.DisasmEntry) {
				src.instructions[uint64(e.Addr)] = &SourceInstruction{
					Addr:   e.Addr,
					opcode: uint32(e.OpcodeHi)<<16 | uint32(e.Opcode),
					size:   e.Size(),
					Disasm: e,
				}
			},
		})
	}

	bld, err := newBuild(dwrf)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	err = bld.buildCompilationUnits()
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	err = bld.buildSourceFiles(src)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	err = bld.buildFunctions(src)
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// sanity check of functions list
	if len(src.Functions) != len(src.FunctionNames) {
		return nil, fmt.Errorf("dwarf: unmatched function definitions")
	}

	// complete function list with stubs for functions where we don't have any
	// DWARF data (but do have symbol data)
	resolveSymbols(src, ef.Symbols())

	// add instructions to each line of source
	err = addInstructionsToLines(src, bld, ef.Symbols())
	if err != nil {
		return nil, fmt.Errorf("dwarf: %w", err)
	}

	// assign each line of source to a function as best as we can
	assignFunctionsToLines(src)

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
	err = bld.buildVariables(src)
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

	// whether any compile unit was created with optimisation enabled
	for _, u := range bld.units {
		src.Optimisation = src.Optimisation || u.optimisation
	}

	// log summary
	logger.Logf(logger.Allow, "dwarf", "optimised compilation: %v", src.Optimisation)
	logger.Logf(logger.Allow, "dwarf", "identified %d functions in %d compile units", len(src.Functions), len(bld.units))
	logger.Logf(logger.Allow, "dwarf", "%d global variables", len(src.SortedGlobals.Variables))
	logger.Logf(logger.Allow, "dwarf", "%d local variable (loclists)", len(src.SortedLocals.Variables))
	logger.Logf(logger.Allow, "dwarf", "high address (%08x)", src.HighAddress)

	return src, nil
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

// find entry function to the program
func findEntryFunction(src *Source) {
	// TODO: this is a bit of ARM specific knowledge that should be removed
	addr, _ := src.cart.GetCoProc().Register(15)
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

// add function stubs for any functions without DWARF data
func resolveSymbols(src *Source, syms []elf.Symbol) error {
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
