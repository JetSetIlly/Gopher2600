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

package archivefs

import "io"

// Open and return an io.ReadSeeker for the specified filename. Filename can be
// inside an archive supported by archivefs
//
// Returns the io.ReadSeeker, the size of the data behind the ReadSeeker and any
// errors.
func Open(filename string) (io.ReadSeeker, int, error) {
	var afs Path
	err := afs.Set(filename)
	if err != nil {
		return nil, 0, err
	}
	defer afs.Close()
	return afs.Open()
}
