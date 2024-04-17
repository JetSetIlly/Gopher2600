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
	"path/filepath"
	"slices"
	"strings"
)

// use information in the Loader instance to decide how the cartridge should be
// referred to by code outside of the package
func decideOnName(ld Loader) string {
	if ld.embedded {
		return ld.Filename
	}

	// return the empty string if filename is undefined
	if len(strings.TrimSpace(ld.Filename)) == 0 {
		return ""
	}

	return NameFromFilename(ld.Filename)
}

// NameFromFilename converts a filename to a shortened version suitable for
// display. Useful in some contexts where creating a cartridge loader instance
// is inconvenient.
func NameFromFilename(filename string) string {
	name := filepath.Base(filename)
	ext := filepath.Ext(filename)
	if slices.Contains(FileExtensions, ext) {
		name = strings.TrimSuffix(name, filepath.Ext(filename))
	}
	return name
}
