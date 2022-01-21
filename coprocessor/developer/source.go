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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/logger"
)

// SrcFile represents a single file of original source code. A file is made up
// of many SrcLine entries.
type SrcFile struct {
	// name and path of loaded file
	Filename string

	// the lines of the file
	Lines []*SrcLine
}

// SrcLine represents a single line of source in a SrcFile.
type SrcLine struct {
	// the file the line is found in
	File *SrcFile

	Function   string // function name line is contained in (if found)
	LineNumber int    // counting from one
	Content    string // the actual line

	Inlined bool

	// the generated assembly for this line. will be empty if line is a comment
	// or otherwise unsused
	Asm []*SrcLineAsm

	// the number of cycles this line has instruction consumed on the
	// coprocessor during the course of the previous frame
	FrameCycles     float32
	nextFrameCycles float32

	// the total number of cycles over the lifetime of the program
	LifetimeCycles float32

	// whether this src line has been responsible for an illegal access
	IllegalAccess bool

	// the number of times the line has been responsible for an illegal access
	IllegalCount int
}

func (src *SrcLine) String() string {
	return fmt.Sprintf("%s: %d", src.File.Filename, src.LineNumber)
}

// SrcLineAsm associates an asm with a block of source (which might be a single line).
type SrcLineAsm struct {
	// address of instruction
	Addr uint32

	// the actual coprocessor instruction
	Instruction string

	// the line of source code this instruction was generated for
	line *SrcLine
}

// Source files for the currently loaded ROM. It is built through a combination
// of a binary objdump and the original source files.
type Source struct {
	Files      map[string]*SrcFile
	FilesNames []string

	// all the asm instructions in the program
	asm map[uint32]*SrcLineAsm

	// A list of all the source lines in the program. only those lines that
	// have SrcLineAsm entries are included.
	//
	// sorted by cycle count from highest to lowest
	ExecutedLines ExecutedLines

	// the number of cycles this line has instruction consumed on the
	// coprocessor during the course of the previous frame
	FrameCycles     float32
	nextFrameCycles float32

	// the total number of cycles over the lifetime of the program
	TotalCycles float32
}

// ExecutedLines orders every line of executable source code in the identified
// source files. Useful for determining the most expensive lines of source code
// in terms of execution time.
type ExecutedLines struct {
	Lines []*SrcLine

	// if true then Lines will be sorted by TotalCycles, otherwise sorted by FrameCycles
	byLifetimeCycles bool
}

// SortedBy returns a string describing the sort method
func (e ExecutedLines) SortedBy() string {
	if e.byLifetimeCycles {
		return "lifetime"
	}
	return "previous frame"
}

// Len implements sort.Interface.
func (e ExecutedLines) Len() int {
	return len(e.Lines)
}

// Less implements sort.Interface.
func (e ExecutedLines) Less(i int, j int) bool {
	// higher cycle counts come first
	if e.byLifetimeCycles {
		return e.Lines[i].LifetimeCycles > e.Lines[j].LifetimeCycles
	}
	return e.Lines[i].FrameCycles > e.Lines[j].FrameCycles
}

// Swap implements sort.Interface.
func (e ExecutedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

const objFile = "armcode.obj"
const objFile_older = "custom2.obj"

func findObjDump(romDir string) *os.File {
	// current working directory
	od, err := os.Open(objFile)
	if err == nil {
		return od
	}

	// same directory as binary
	od, err = os.Open(filepath.Join(romDir, objFile))
	if err == nil {
		return od
	}

	// main sub-directory
	od, err = os.Open(filepath.Join(romDir, "main", objFile))
	if err == nil {
		return od
	}

	// main/bin sub-directory
	od, err = os.Open(filepath.Join(romDir, "main", "bin", objFile))
	if err == nil {
		return od
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	od, err = os.Open(filepath.Join(romDir, "custom", "bin", objFile_older))
	if err == nil {
		return od
	}

	// jetsetilly source tree
	od, err = os.Open(filepath.Join(romDir, "arm", "main.obj"))
	if err == nil {
		return od
	}

	return nil
}

func readSourceFile(filename string, pathToROM string) (*SrcFile, error) {
	// remove superfluous path direction
	filename = filepath.Clean(filename)

	fl := SrcFile{
		Filename: filename,
	}

	var err error
	var b []byte

	// try to open file. first as a path relative to the ROM and if that fails,
	// as an absolute path
	b, err = ioutil.ReadFile(filepath.Join(filepath.Dir(pathToROM), filename))
	if err != nil {
		b, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	}

	// split files into lines
	for i, s := range strings.Split(string(b), "\n") {
		fl.Lines = append(fl.Lines, &SrcLine{
			File:       &fl,
			LineNumber: i + 1,
			Content:    s,
		})
	}

	return &fl, nil
}

// newSource loads and parses an obj file. Returns a new instance of ObjDump
// or any errors.
func newSource(pathToROM string) (*Source, error) {
	src := &Source{
		Files:      make(map[string]*SrcFile),
		FilesNames: make([]string, 0),
		asm:        make(map[uint32]*SrcLineAsm),
		ExecutedLines: ExecutedLines{
			Lines: make([]*SrcLine, 0),
		},
	}

	// path to ROM without the filename
	romDir := filepath.Dir(pathToROM)

	// find objdump file and open it
	od := findObjDump(romDir)
	if od == nil {
		return nil, curated.Errorf("objfile: gcc .obj file not available (%s)", objFile)
	}
	defer od.Close()

	// read all data, split into lines
	data, err := io.ReadAll(od)
	if err != nil {
		return nil, curated.Errorf("objfile: processing error: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	// regexes for lines in objdump file

	// lines that refer to a source file
	fileMatch, err := regexp.Compile("^[[:print:]]+:[[:digit:]]+$")
	if err != nil {
		panic(fmt.Sprintf("objdump: %s", err.Error()))
	}

	// lines that contain the compiled ASM instructions
	asmMatch, err := regexp.Compile("[[:xdigit:]]{3}:.*$")
	if err != nil {
		panic(fmt.Sprintf("objdump: %s", err.Error()))
	}

	// map of ReadFile errors already seen, so we don't print the same error
	// over and over
	fileNotFound := make(map[string]bool)

	var currentLine *SrcLine

	// examine every line of the objdump
	for _, ol := range lines {
		if fileMatch.Match([]byte(ol)) {
			fm := strings.Split(ol, ":")
			if len(fm) != 2 {
				logger.Log("objdump", "malformed filename/linenumber entry")
				continue // for loop
			}

			// convert UNIX seperator (which may be used in the .obj file) to
			// the native host path separator
			fm[0] = strings.Join(strings.Split(fm[0], "/"), string(filepath.Separator))
			fm[0] = fmt.Sprintf("%s%s", filepath.VolumeName(romDir), fm[0])

			// chop off path prefix
			prefix := fmt.Sprintf("%s%c", romDir, filepath.Separator)
			if strings.HasPrefix(fm[0], prefix) {
				fm[0] = fm[0][len(prefix):]
			}

			// objdump refers to a file that has not been seen before - read the source file
			if _, ok := src.Files[fm[0]]; !ok {
				var err error
				src.Files[fm[0]], err = readSourceFile(fm[0], pathToROM)

				if err != nil {
					delete(src.Files, fm[0])
					if _, ok := fileNotFound[err.Error()]; !ok {
						fileNotFound[err.Error()] = true
						logger.Log("objdump", err.Error())
					}

					continue // for loop
				}

				// add filename to list of keys
				src.FilesNames = append(src.FilesNames, fm[0])
			}

			// parse line number directive and note current line
			ln, err := strconv.ParseUint(fm[1], 10, 32)
			if err != nil {
				logger.Log("objdump", err.Error())
				continue
			}

			// we index lines from zero but lines are counted from one in the objdump
			ln -= 1

			currentLine = src.Files[fm[0]].Lines[ln]

		} else if asmMatch.Match([]byte(ol)) {
			// addrEnd always seems to  be at index 8 but we'll search for it
			// anyway because a fixed value doesn't seem safe
			addrEnd := strings.Index(ol, ":")

			addr, err := strconv.ParseUint(strings.TrimSpace(ol[:addrEnd]), 16, 32)
			if err == nil {
				if currentLine != nil {
					asmEntry := SrcLineAsm{
						Addr:        uint32(addr),
						Instruction: strings.TrimSpace(ol[addrEnd+1:]),
						line:        currentLine,
					}
					src.asm[uint32(addr)] = &asmEntry
					currentLine.Asm = append(currentLine.Asm, &asmEntry)
				}
			}
		}
	}

	// label every source line theat belongs to a function. we'll use a map
	// file for this because it's easier
	mapfile, err := newMapFile(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	} else {
		for _, f := range src.Files {
			for _, l := range f.Lines {
				if len(l.Asm) > 0 {
					// get function name for line using the address of the
					// first assembly instruction
					e := mapfile.findEntry(l.Asm[0].Addr)
					l.Function = e.functionName

					// rudimentary detection of inline functions. to do this
					// we're assuming that the filename as detected by the
					// objfile parsing is "similar" to the .o file detected by
					// the mapfile parsing
					if len(e.objfile) > 0 && len(l.File.Filename) > 0 {
						// because the build system may put the output file in
						// a different location to the source file we'll just
						// look at the base filename (without the path)
						//
						// again, this is very rudimentary - we need a better
						// way of finding inline functions
						a := filepath.Base(e.objfile)
						b := filepath.Base(l.File.Filename)

						l.Inlined = a[:len(a)-2] != b[:len(b)-2]
					}
				}
			}
		}
	}

	// populate SrcLinesAll by recursing through every file and every line
	for _, f := range src.Files {
		for _, l := range f.Lines {
			if len(l.Asm) > 0 {
				src.ExecutedLines.Lines = append(src.ExecutedLines.Lines, l)
			}
		}
	}

	// sort SrcLinesAll
	sort.Sort(src.ExecutedLines)

	// sort list of filename keys
	sort.Strings(src.FilesNames)

	return src, nil
}

// findProgramAccess returns the program (function) label for the supplied
// address. Addresses may be in a range.
func (src *Source) findProgramAccess(address uint32) *SrcLine {
	asm := src.asm[address]
	if asm == nil {
		return nil
	}
	return asm.line
}

// dump everything to io.Writer.
func (src *Source) dump(w io.Writer) {
	for fn, f := range src.Files {
		w.Write([]byte(fmt.Sprintf("%s\n", fn)))
		w.Write([]byte(fmt.Sprintf("-------\n")))

		for _, ln := range f.Lines {
			w.Write([]byte(fmt.Sprintf("%d:\t%s\n", ln.LineNumber, ln.Content)))
			for _, asm := range ln.Asm {
				w.Write([]byte(fmt.Sprintf("%s\n", asm.Instruction)))
			}
		}
	}
}

// execute address and increase source line count.
func (src *Source) execute(address uint32, ct float32) {
	if a, ok := src.asm[address]; ok {
		a.line.nextFrameCycles += ct
		src.nextFrameCycles += ct
		a.line.LifetimeCycles += ct
		src.TotalCycles += ct
	}
}

// Resort the execution source lines.
func (src *Source) Resort(byLifetimeCycles bool) {
	if src.ExecutedLines.byLifetimeCycles == byLifetimeCycles {
		return
	}

	src.ExecutedLines.byLifetimeCycles = byLifetimeCycles
	sort.Sort(src.ExecutedLines)
}
