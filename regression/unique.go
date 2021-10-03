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
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/paths"
)

// create a unique filename from a CatridgeLoader instance. used when saving
// scripts into regressionScripts directory. calls paths.UniqueFilename() to
// maintain common formatting used in the project.
func uniqueFilename(prepend string, cartload cartridgeloader.Loader) (string, error) {
	f := paths.UniqueFilename(prepend, cartload.ShortName())

	scriptsPath, err := paths.ResourcePath(regressionPath, regressionScripts)
	if err != nil {
		return "", err
	}

	scrPth, err := paths.ResourcePath(scriptsPath, f)
	if err != nil {
		return "", err
	}

	return scrPth, nil
}
