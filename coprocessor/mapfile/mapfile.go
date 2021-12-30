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

// package map file is a very basic parser for GCC style mapfiles.
// FindDataAccess() and FindProgramAccess() will return best-guess labels for
// the supplied address.
//
// Map file must be in the working directory and be called "armcode.map"
package mapfile

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

type entry struct {
	address uint32
	label   string
}

// Mapfile contains the parsed information from the map file.
type Mapfile struct {
	data    []entry
	program []entry
}

const mapFile = "armcode.map"
const mapFile_older = "custom2.map"

func findMapFile(pathToROM string) *os.File {
	// current working directory
	sf, err := os.Open(mapFile)
	if err == nil {
		return sf
	}

	dir := filepath.Dir(pathToROM)

	// same direcotry as binary
	sf, err = os.Open(filepath.Join(dir, mapFile))
	if err == nil {
		return sf
	}

	// main sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", mapFile))
	if err == nil {
		return sf
	}

	// main/bin sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", "bin", mapFile))
	if err == nil {
		return sf
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	sf, err = os.Open(filepath.Join(dir, "custom", "bin", mapFile_older))
	if err == nil {
		return sf
	}

	return nil
}

// NewMapFile loads and parses a map file. Returns a new instance of Mapfile or
// any errors.
func NewMapFile(pathToROM string) (*Mapfile, error) {
	mf := &Mapfile{
		data:    make([]entry, 0, 32),
		program: make([]entry, 0, 32),
	}

	sf := findMapFile(pathToROM)
	if sf == nil {
		return nil, curated.Errorf("mapfile: gcc .map file not available (%s)", mapFile)
	}
	defer sf.Close()

	data, err := io.ReadAll(sf)
	if err != nil {
		return nil, curated.Errorf("mapfile: processing error: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	for i, l := range lines {
		if l == "Linker script and memory map" {
			lines = lines[i:]
			break // for loop
		}
	}

	var entryArray *[]entry
	var deferredfunctionName string

	for _, l := range lines {
		flds := strings.Fields(l)
		if len(flds) == 0 {
			continue // for loop
		}

		if deferredfunctionName != "" {
			address, err := strconv.ParseInt(flds[0], 0, 64)
			if err != nil {
				return nil, curated.Errorf("mapfile: processing error: %v", err)
			}

			(*entryArray) = append(*entryArray, entry{
				address: uint32(address),
				label:   deferredfunctionName,
			})

			deferredfunctionName = ""
			continue // for loop
		}

		if strings.HasPrefix(flds[0], "0x") {
			if entryArray != nil {
				if !(flds[1][0] == '.' || flds[1][0] == '_') {
					address, err := strconv.ParseInt(flds[0], 0, 64)
					if err != nil {
						return nil, curated.Errorf("mapfile: processing error: %v", err)
					}

					(*entryArray) = append(*entryArray, entry{
						address: uint32(address),
						label:   flds[1],
					})
				}
			}
		} else {
			switch flds[0] {
			case ".rodata":
				entryArray = &mf.data
			case ".data":
				entryArray = &mf.data
			case "COMMON":
				entryArray = &mf.data
			default:
				if len(flds) == 1 && strings.HasPrefix(flds[0], ".text.") {
					deferredfunctionName = flds[0][6:]
					entryArray = &mf.program
				} else {
					entryArray = nil
				}
			}
		}
	}

	return mf, nil
}

// FindDataAccess returns the data label for the supplied address. Addresses
// will be matched exactly.
func (mf *Mapfile) FindDataAccess(address uint32) string {
	for _, e := range mf.data {
		if address == e.address {
			return e.label
		}
	}
	return ""
}

// FindProgramAccess returns the program (function) label for the supplied
// address. Addresses may be in a range.
func (mf *Mapfile) FindProgramAccess(address uint32) string {
	label := ""

	for _, e := range mf.program {
		if address < e.address {
			return label
		}
		label = e.label
	}

	return label
}
