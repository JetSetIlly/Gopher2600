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

package resources

import (
	"os"
	"strings"
)

const (
	portableFile = "portable.txt"
	portablePath = "Gopher2600_UserData"
)

// check working directory for a file named portable.txt
func checkPortable() bool {
	files, err := os.ReadDir(".")
	if err != nil {
		return false
	}

	// the name of the portable.txt file is case insensitive
	for _, file := range files {
		if strings.ToLower(file.Name()) == portableFile {
			return true
		}
	}

	return false
}
