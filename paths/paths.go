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

package paths

import (
	"path"
)

// ResourcePath returns the resource string (representing the resource to be
// loaded) prepended with OS/build specific paths.
//
// The function takes care of creation of all folders necessary to reach the
// end of sub-path. It does not otherwise touch or create the file.
//
// Either subPth or file can be empty, depending on context.
func ResourcePath(subPth string, file string) (string, error) {
	var pth []string

	basePath, err := getBasePath(subPth)
	if err != nil {
		return "", err
	}

	pth = make([]string, 0)
	pth = append(pth, basePath)
	pth = append(pth, file)

	return path.Join(pth...), nil
}
