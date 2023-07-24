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
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// SourceFileContent lists the lines in a source file
type SourceFileContent struct {
	Lines        []*SourceLine
	MaxLineWidth int
}

// String implements the fuzzy.Source interface
func (s SourceFileContent) String(i int) string {
	return s.Lines[i].PlainContent
}

// Len implements the fuzzy.Source interface
func (s SourceFileContent) Len() int {
	return len(s.Lines)
}

// SourceFile is a single source file indentified by the DWARF data.
type SourceFile struct {
	Filename      string
	ShortFilename string
	Content       SourceFileContent

	// the source file has at least one global variable if HasGlobals is true
	HasGlobals bool
}

// IsStub returns true if the SourceFile is just a stub.
func (f *SourceFile) IsStub() bool {
	return f.Filename == stubIndicator
}

// SourceInstruction is a single intruction from the ELF binary with a reference
// to a disassembly supplied by the cartridges coprocessor interface.
//
// Not to be confused with anything in the the coprocessor.disassembly package.
// SourceInstruction instances are intended to be used by static disasemblers.
type SourceInstruction struct {
	// the address in memory of the instruction
	Addr uint32

	// an opcode may be 16bit or 32bit. the size field indicates how much of the
	// opcode field is to be used
	opcode uint32
	size   int

	// the disassembly entry from the cartridge dissassembly. we don't deal with
	// the details of this type in the coprocessor.developer package
	Disasm mapper.CartCoProcDisasmEntry

	// the line of source code this diassembly entry is associated with
	Line *SourceLine
}

// Opcode returns a string formatted opcode appropriate for the bit length.
func (d *SourceInstruction) Opcode() string {
	switch d.size {
	case 2:
		return fmt.Sprintf("%04x", d.opcode)
	case 4:
		return fmt.Sprintf("%04x %04x", uint16(d.opcode>>16), uint16(d.opcode))
	}
	panic(fmt.Sprintf("unsupported opcode size (%d)", d.size))
}

func (d *SourceInstruction) String() string {
	switch d.size {
	case 2:
		return fmt.Sprintf("%#08x %04x %s", d.Addr, uint16(d.opcode), d.Disasm.String())
	case 4:
		return fmt.Sprintf("%#08x %04x %04x %s", d.Addr, uint16(d.opcode>>16), uint16(d.opcode), d.Disasm.String())
	}
	panic(fmt.Sprintf("unsupported opcode size (%d)", d.size))
}

type AllSourceLines struct {
	lines []*SourceLine
}

func (a AllSourceLines) Get(i int) *SourceLine {
	if i >= len(a.lines) {
		return nil
	}
	return a.lines[i]
}

func (a AllSourceLines) Len() int {
	return len(a.lines)
}

func (a AllSourceLines) String(i int) string {
	if i >= len(a.lines) {
		return ""
	}
	return a.lines[i].PlainContent
}

// SourceLine is a single line of source in a source file, identified by the
// DWARF data and loaded from the actual source file.
type SourceLine struct {
	// the actual file/line of the SourceLine. line numbers are counted from one
	File       *SourceFile
	LineNumber int

	// the function the line of source can be found within
	Function *SourceFunction

	// whether this line is to be found inlined elsewhere in the program
	Inlined bool

	// plain string of line
	PlainContent string

	// line divided into parts
	Fragments []SourceLineFragment

	// the list of instructions for this line. will be empty if line is not used
	// in the program (eg. a comment line)
	//
	// note that only instructions for non-inlined ranges will be collated
	Instruction []*SourceInstruction

	// whether this source line has been responsible for a likely bug (eg. illegal access of memory)
	Bug bool

	// statistics for the line
	Stats profiling.StatsGroup

	// which 2600 kernel has this line executed in
	Kernel profiling.Focus
}

func (ln *SourceLine) String() string {
	if ln.IsStub() {
		return ln.PlainContent
	}
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s:%d", ln.File.ShortFilename, ln.LineNumber))
	if !ln.Function.IsStub() {
		s.WriteString(fmt.Sprintf(" [%s]", ln.Function.Name))
	}
	return s.String()
}

// IsStub returns true if the SourceLine is just a stub.
func (ln *SourceLine) IsStub() bool {
	return ln.PlainContent == stubIndicator
}

// SourceRange is used to specify the effective start and end addresses of a
// function or a variable.
type SourceRange struct {
	Start  uint64
	End    uint64
	Inline bool
}

// String returns the start/end addresses of the range. If the range is inlined
// then the addresses are printed with square brackets.
func (r SourceRange) String() string {
	if r.Inline {
		return fmt.Sprintf("[%08x to %08x]", r.Start, r.End)
	}
	return fmt.Sprintf("(%08x to %08x)", r.Start, r.End)
}

// InRange returns true if address is in range of start and end addresses
func (r SourceRange) InRange(addr uint64) bool {
	return addr >= r.Start && addr <= r.End
}

// Size returns the size of the range
func (r SourceRange) Size() uint64 {
	return r.End - r.Start
}

// DriverFunctionName is the name given to a function that represents all the
// instructions that fall outside of the ROM and are in fact in the "driver".
const DriverFunctionName = "<driver>"

// SourceFunction is a single function identified by the DWARF data or by the
// ELF symbol table in the case of no DWARF information being available for the
// function.
type SourceFunction struct {
	// name of function
	Name string

	// range of addresses in which function resides
	Range []SourceRange

	// location list. used to identify the frame base of a function
	framebaseLoclist *loclist

	// first source line for each instance of the function. note that the first
	// line of a function may not have any code directly associated with it -
	// meaning that the Instruction and Stats fields of the DeclLine may be
	// empty
	DeclLine *SourceLine

	// stats for the function
	FlatStats       profiling.StatsGroup
	CumulativeStats profiling.StatsGroup

	// which 2600 kernel has this function executed in
	Kernel profiling.Focus

	// whether the call stack involving this function is likely inaccurate
	OptimisedCallStack bool
}

func (fn *SourceFunction) String() string {
	s := strings.Builder{}
	s.WriteString(fn.Name)
	s.WriteString("\n  ")
	for _, r := range fn.Range {
		s.WriteString(fmt.Sprintf(" %s", r))
	}
	return s.String()
}

// IsInlined returns true if the function has at least one inlined instance
func (fn *SourceFunction) IsInlined() bool {
	for _, r := range fn.Range {
		if r.Inline {
			return true
		}
	}
	return false
}

// framebase implements the loclistFramebase interface
func (fn *SourceFunction) framebase() (uint64, error) {
	if fn.IsStub() {
		return 0, fmt.Errorf("no framebase for function")
	}

	if fn.framebaseLoclist == nil {
		return 0, fmt.Errorf("no framebase loclist for %s", fn.Name)
	}

	r, err := fn.framebaseLoclist.resolve()
	if err != nil {
		return 0, err
	}

	// framebase loclist will almost certainly be flagged as having an address
	// but in case it does not we do the test and return value as appropriate

	if r.hasAddress {
		return r.address, nil
	}

	return uint64(r.value), nil
}

// IsStub returns true if the SourceFunction is just a stub
func (fn *SourceFunction) IsStub() bool {
	// it's possible to have a stub function that has a name. because of this
	// we check the DeclLine field in addition to the name field
	return fn.Name == stubIndicator || fn.DeclLine.IsStub()
}

// SourceType is a single type identified by the DWARF data. Composite types
// are differentiated by the existance of member fields
type SourceType struct {
	Name string

	// is a constant type
	Constant bool

	// the base type of pointer types. will be nil if type is not a pointer type
	PointerType *SourceType

	// size of values of this type (in bytes)
	Size int

	// empty if type is not a composite type. see SourceVariable.IsComposite()
	// function
	Members []*SourceVariable

	// number of elements in the type. if count is more than zero then this
	// type is an array. see SourceVariable.IsArry() function
	ElementCount int

	// the base type of all the elements in the type
	ElementType *SourceType
}

func (typ *SourceType) String() string {
	return typ.Name
}

// IsComposite returns true if SourceType is a composite type.
func (typ *SourceType) IsComposite() bool {
	return len(typ.Members) > 0
}

// IsArray returns true if SourceType is an array type.
func (typ *SourceType) IsArray() bool {
	return typ.ElementType != nil && typ.ElementCount > 0
}

// IsPointer returns true if SourceType is a pointer type.
func (typ *SourceType) IsPointer() bool {
	return typ.PointerType != nil
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

// stubIndicator is allocated to key fields in SourceFile and SourceLine to
// indicate that they are not "real" files or lines and only exist to avoid nil
// pointers.
//
// Values of type SourceFunction can also be stubs. Although do note that in
// all instances the name of the function is always known (or can be assumed,
// see DriverFunctionName). Whether a SourceFunction is a stub therefore, is
// decided by whether the DeclLine is a stub.
//
// The IsStub() functions for the SourceFile, SourceFunction and SourceLine
// types codify stub detection.
const stubIndicator = "<stub>"

// CreateStubLine returns an instance of SourceLine with the specified
// SourceFunction assigned to it.
//
// If stubFn is nil then a dummy function will be created.
//
// A stub SourceFile will be created for assignment to the SourceLine.File
// field.
func CreateStubLine(stubFn *SourceFunction) *SourceLine {
	if stubFn == nil {
		stubFn = &SourceFunction{
			Name: stubIndicator,
		}
	}

	// the DeclLine field must definitely be nil for a stubFn function
	stubFile := &SourceFile{
		Filename:      stubIndicator,
		ShortFilename: stubIndicator,
	}

	// each address in the stub function shares the same stub line
	stubLn := &SourceLine{
		File:         stubFile,
		Function:     stubFn,
		PlainContent: stubIndicator,
	}

	stubFn.DeclLine = stubLn
	return stubLn
}
