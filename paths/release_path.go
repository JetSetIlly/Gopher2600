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

// +build release

package paths

import (
	"os"
	"path"
)

const gopherConfigDir = "gopher2600"

// the release version of getBasePath looks for and if necessary creates the
// gopherConfigDir (and child directories) in the User's configuration
// directory, which is dependent on the host OS (see os.UserConfigDir()
// documentation for details)
func getBasePath(subPth string) (string, error) {
	cnf, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	pth := path.Join(cnf, gopherConfigDir, subPth)

	if _, err := os.Stat(pth); err == nil {
		return pth, nil
	}

	if err := os.MkdirAll(pth, 0700); err != nil {
		return "", err
	}

	return pth, nil
}
