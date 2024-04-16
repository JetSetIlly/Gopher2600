// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the gnu general public license as published by
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

package cartridgeloader

import (
	"bytes"
	"os"
)

// mini-fingerprints exist only to help the cartridge loader make a correct
// decision about how to handle the cartridge data. we don't need to know much
// about the data for most cartridge types
//
// full cartridge fingerprinting is in the cartridge package

// special handling for MVC files without the MVC file extension
func miniFingerprintMovieCart(filename string) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer f.Close()
	b := make([]byte, 4)
	f.Read(b)
	if bytes.Compare(b, []byte{'M', 'V', 'C', 0x00}) == 0 {
		return true, nil
	}
	return false, nil
}
