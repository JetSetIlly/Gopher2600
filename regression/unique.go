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

package regression

import (
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

// uniqueFilename is a shim function for unique.Filename().
//
// It create a unique filename from a CatridgeLoader instance. used when saving
// scripts into regressionScripts directory. calls paths.UniqueFilename() to
// maintain common formatting used in the project.
func uniqueFilename(filetype string, cartname string) (string, error) {
	f := unique.Filename(filetype, cartname)

	scriptsPath, err := resources.JoinPath(regressionPath, regressionScripts)
	if err != nil {
		return "", err
	}

	p, err := resources.JoinPath(scriptsPath, f)
	if err != nil {
		return "", err
	}

	return p, nil
}
