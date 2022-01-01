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

// package objdump file is a very basic parser for obj files as produced by
// "objdump -S" on the base elf file that is used to create a cartridge binary
//
// FindDataAccess() and FindProgramAccess() will return best-guess labels for
// the supplied address.
package objdump

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

// asmEntry associates an asm asmEntry with a block of source (which might be a
// single line)
type asmEntry struct {
	asm string
	src *line
}

// asmEntry associates an asm asmEntry with a block of source (which might be a
// single line)
type srcEntry struct {
	block string
	addr  []uint32
}

type line struct {
	file    *file
	number  int // counting from one
	content string
	asm     []*asmEntry
}

type file struct {
	filename string
	lines    []line
}

// ObjDump contains the parsed information from the obj file.
type ObjDump struct {
	files     map[string]*file
	files_key []string
	asm       map[uint32]asmEntry
}

const objFile = "armcode.obj"
const objFile_older = "custom2.obj"

func findObjDump(pathToROM string) *os.File {
	// current working directory
	sf, err := os.Open(objFile)
	if err == nil {
		return sf
	}

	dir := filepath.Dir(pathToROM)

	// same directory as binary
	sf, err = os.Open(filepath.Join(dir, objFile))
	if err == nil {
		return sf
	}

	// main sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", objFile))
	if err == nil {
		return sf
	}

	// main/bin sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", "bin", objFile))
	if err == nil {
		return sf
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	sf, err = os.Open(filepath.Join(dir, "custom", "bin", objFile_older))
	if err == nil {
		return sf
	}

	return nil
}

func readSourceFile(filename string, pathToROM string) (*file, error) {
	// remove superfluous path direction
	filename = filepath.Clean(filename)

	fl := file{
		filename: filename,
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
		fl.lines = append(fl.lines, line{
			file:    &fl,
			number:  i + 1,
			content: s,
		})
	}

	return &fl, nil
}

// NewObjDump loads and parses an obj file. Returns a new instance of ObjDump
// or any errors.
func NewObjDump(pathToROM string) (*ObjDump, error) {
	obj := &ObjDump{
		files:     make(map[string]*file),
		files_key: make([]string, 0),
		asm:       make(map[uint32]asmEntry),
	}

	// find objdump file and open it
	sf := findObjDump(pathToROM)
	if sf == nil {
		return nil, curated.Errorf("objfile: gcc .obj file not available (%s)", objFile)
	}
	defer sf.Close()

	// read all data, split into lines
	data, err := io.ReadAll(sf)
	if err != nil {
		return nil, curated.Errorf("objfile: processing error: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	asmMatch, err := regexp.Compile("^[[:xdigit:]]{8}:.*$")
	if err != nil {
		panic(fmt.Sprintf("objdump: %s", err.Error()))
	}

	fileMatch, err := regexp.Compile("^[[:print:]]+:[[:digit:]]+$")
	if err != nil {
		panic(fmt.Sprintf("objdump: %s", err.Error()))
	}

	// map of ReadFile errors already seen, so we don't print the same error
	// over and over
	fileNotFound := make(map[string]bool)

	var currentLine *line

	for _, ol := range lines {
		if fileMatch.Match([]byte(ol)) {
			fm := strings.Split(ol, ":")
			if len(fm) != 2 {
				logger.Log("objdump", "malformed filename/linenumber entry")
				continue // for loop
			}

			// file has not been seen before
			if _, ok := obj.files[fm[0]]; !ok {
				var err error
				obj.files[fm[0]], err = readSourceFile(fm[0], pathToROM)

				if err != nil {
					delete(obj.files, fm[0])
					if _, ok := fileNotFound[err.Error()]; !ok {
						fileNotFound[err.Error()] = true
						logger.Log("objdump", err.Error())
					}

					continue // for loop
				}

				// add filename to list of keys
				obj.files_key = append(obj.files_key, fm[0])
			}

			// parse line number directive and note current line
			ln, err := strconv.ParseUint(fm[1], 10, 32)
			if err != nil {
				logger.Log("objdump", err.Error())
				continue
			}

			// we index lines from zero but lines are counted from one in the objdump
			ln -= 1

			currentLine = &obj.files[fm[0]].lines[ln]

		} else if asmMatch.Match([]byte(ol)) {
			addr, err := strconv.ParseUint(ol[:8], 16, 32)
			if err == nil {
				if currentLine != nil {
					obj.asm[uint32(addr)] = asmEntry{
						asm: strings.TrimSpace(ol[9:]),
						src: currentLine,
					}

					asmEntry := obj.asm[uint32(addr)]
					currentLine.asm = append(currentLine.asm, &asmEntry)
				}
			}
		}
	}

	// sort list of filename keys
	sort.Strings(obj.files_key)

	return obj, nil
}

// FindProgramAccess returns the program (function) label for the supplied
// address. Addresses may be in a range.
func (obj *ObjDump) FindProgramAccess(address uint32) string {
	asm := obj.asm[address]
	return fmt.Sprintf("%s:%d\n%s", asm.src.file.filename, asm.src.number, asm.src.content)
}

// dump everything to io.Writer.
func (obj *ObjDump) dump(w io.Writer) {
	for fn, f := range obj.files {
		w.Write([]byte(fmt.Sprintf("%s\n", fn)))
		w.Write([]byte(fmt.Sprintf("-------\n")))

		for _, ln := range f.lines {
			w.Write([]byte(fmt.Sprintf("%d:\t%s\n", ln.number, ln.content)))
			for _, asm := range ln.asm {
				w.Write([]byte(fmt.Sprintf("%s\n", asm.asm)))
			}
		}
	}
}
