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
	"os"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
)

const biosFile = "Supercharger BIOS.bin"

func loadBIOS() ([]uint8, error) {
	var f *os.File

	biosFilePath, err := paths.ResourcePath("", biosFile)
	if err != nil {
		return nil, errors.New(errors.SuperchargerError, err)
	}

	f, err = os.Open(biosFilePath)
	if err != nil {
		return nil, errors.New(errors.SuperchargerError, "can't load BIOS")
	}
	defer f.Close()

	// get file info
	cfi, err := f.Stat()
	if err != nil {
		return nil, errors.New(errors.SuperchargerError, "can't load BIOS")
	}
	size := cfi.Size()

	data := make([]byte, size)
	_, err = f.Read(data)
	if err != nil {
		return nil, errors.New(errors.SuperchargerError, "can't load BIOS")
	}

	return data, nil
}
