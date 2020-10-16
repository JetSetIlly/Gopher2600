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

package supercharger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
)

// list of allowed filenames for the supercharger BIOS.
var biosFile = [...]string{
	"Supercharger BIOS.bin",
	"Supercharger.BIOS.bin",
	"Supercharger_BIOS.bin",
}

// tag string used in called to Log().
const biosLogTag = "supercharger: bios"

// loadBIOS attempts to load BIOS from (in order of priority):
//	- current working directory
//	- the same directory as the tape/bin file
//	- the emulator's resource path
func loadBIOS(path string) ([]uint8, error) {
	// current working directory
	for _, b := range biosFile {
		d, err := _loadBIOS(b)
		if err != nil {
			continue
		}

		logger.Log(biosLogTag, fmt.Sprintf("using %s (from current working directory)", b))
		return d, nil
	}

	// the same directory as the tape/bin file
	for _, b := range biosFile {
		p := filepath.Join(path, b)
		d, err := _loadBIOS(p)
		if err != nil {
			continue
		}

		logger.Log(biosLogTag, fmt.Sprintf("using %s", p))
		return d, nil
	}

	// the emulator's resource path
	for _, b := range biosFile {
		p, err := paths.ResourcePath("", b)
		if err != nil {
			return nil, err
		}

		d, err := _loadBIOS(p)
		if err != nil {
			continue
		}

		logger.Log(biosLogTag, fmt.Sprintf("using %s", p))
		return d, nil
	}

	return nil, fmt.Errorf("can't load BIOS")
}

func _loadBIOS(biosFilePath string) ([]uint8, error) {
	f, err := os.Open(biosFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// get file info. not using Stat() on the file handle because the
	// windows version (when running under wine) does not handle that
	cfi, err := os.Stat(biosFilePath)
	if err != nil {
		return nil, err
	}
	size := cfi.Size()

	data := make([]byte, size)
	_, err = f.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
