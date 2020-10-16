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
	"fmt"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/paths"
)

// create a unique filename from a CatridgeLoader instance. used when saving
// scripts into regressionScripts directory.
func uniqueFilename(prepend string, cartload cartridgeloader.Loader) (string, error) {
	n := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
	newScript := fmt.Sprintf("%s_%s_%s", prepend, cartload.ShortName(), timestamp)

	scrPth, err := paths.ResourcePath(regressionScripts, newScript)
	if err != nil {
		return "", err
	}

	return scrPth, nil
}
