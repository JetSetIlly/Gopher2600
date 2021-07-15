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

//go:build !release
// +build !release

package paths

import (
	"os"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/paths/fs"
)

const gopherConfigDir = ".gopher2600"

// the non-release version of getBasePath looks for and if necessary creates
// the gopherConfigDir (and child directories) in the current working
// directory.
func getBasePath(subPth string) (string, error) {
	pth := filepath.Join(gopherConfigDir, subPth)

	if _, err := os.Stat(pth); err == nil {
		return pth, nil
	}

	if err := fs.MkdirAll(pth, 0700); err != nil {
		return "", err
	}

	return pth, nil
}
