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

	"github.com/jetsetilly/gopher2600/paths"
)

var biosFile = [...]string{
	"Supercharger BIOS.bin",
	"Supercharger.BIOS.bin",
	"Supercharger_BIOS.bin",
}

func loadBIOS() ([]uint8, error) {
	for _, b := range biosFile {
		biosFilePath, err := paths.ResourcePath("", b)
		if err != nil {
		}

		var f *os.File

		f, err = os.Open(biosFilePath)
		if err != nil {
			continue
		}
		defer f.Close()

		// get file info. not using Stat() on the file handle because the
		// windows version (when running under wine) does not handle that
		cfi, err := os.Stat(biosFilePath)
		if err != nil {
			continue
		}
		size := cfi.Size()

		data := make([]byte, size)
		_, err = f.Read(data)
		if err != nil {
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("can't load BIOS")

}
