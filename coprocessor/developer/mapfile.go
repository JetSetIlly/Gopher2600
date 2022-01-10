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

// mapfile contains the parsed information from the map file. this supplements
// the information found in the Source structure and provides a little more
// detail that isn't easily retrieved with just the Source mechanism.
type mapfile struct {
	data    []entry
	program []entry
}

const mapFile = "armcode.map"
const mapFile_older = "custom2.map"

func findMapFile(romDir string) *os.File {
	// current working directory
	fl, err := os.Open(mapFile)
	if err == nil {
		return fl
	}

	// same direcotry as binary
	fl, err = os.Open(filepath.Join(romDir, mapFile))
	if err == nil {
		return fl
	}

	// main sub-directory
	fl, err = os.Open(filepath.Join(romDir, "main", mapFile))
	if err == nil {
		return fl
	}

	// main/bin sub-directory
	fl, err = os.Open(filepath.Join(romDir, "main", "bin", mapFile))
	if err == nil {
		return fl
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	fl, err = os.Open(filepath.Join(romDir, "custom", "bin", mapFile_older))
	if err == nil {
		return fl
	}

	// jetsetilly source tree
	fl, err = os.Open(filepath.Join(romDir, "arm", "main.map"))
	if err == nil {
		return fl
	}

	return nil
}

// newMapFile loads and parses a map file. Returns a new instance of Mapfile or
// any errors.
func newMapFile(pathToROM string) (*mapfile, error) {
	mf := &mapfile{
		data:    make([]entry, 0, 32),
		program: make([]entry, 0, 32),
	}

	// path to ROM without the filename
	romDir := filepath.Dir(pathToROM)

	// find objdump file and open it
	fl := findMapFile(romDir)
	if fl == nil {
		return nil, curated.Errorf("mapfile: gcc .map file not available (%s)", mapFile)
	}
	defer fl.Close()

	// read all data, split into lines
	data, err := io.ReadAll(fl)
	if err != nil {
		return nil, curated.Errorf("mapfile: processing error: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	// find the start of mapfile that we're interested in. everything we skip
	// is of no interest or misleading
	for i, l := range lines {
		if l == "Linker script and memory map" {
			lines = lines[i:]
			break // for loop
		}
	}

	var entryArray *[]entry
	var deferredfunctionName string

	// examine remaining lines of mapfile
	for _, l := range lines {
		// ignore empty lines
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

// findFunctionName returns the function name for the supplied address. returns
// the empty string if function name cannot be found.
func (mf *mapfile) findFunctionName(address uint32) string {
	functionName := ""

	for _, e := range mf.program {
		if address < e.address {
			return functionName
		}
		functionName = e.label
	}

	return functionName
}
