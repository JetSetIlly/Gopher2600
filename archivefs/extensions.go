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

import (
	"fmt"
	"path/filepath"
	"strings"
)

// list of file extensions for the supported archive types
var ArchiveExtensions = [...]string{".ZIP"}

// RemoveArchiveExt removes the file extension of any supported/recognised
// archive type from within the string. Only the first instance of the extension
// is removed
func RemoveArchiveExt(s string) string {
	t := strings.ToUpper(s)
	for _, ext := range ArchiveExtensions {
		i := strings.Index(t, ext)
		if i >= 0 {
			return fmt.Sprintf("%s%s", s[:i], s[i+len(ext):])
		}
	}

	return s
}

// TrimArchiveExt removes the file extension of any supported/recognised archive
// type from the end of the string
func TrimArchiveExt(s string) string {
	sext := strings.ToUpper(filepath.Ext(s))
	for _, ext := range ArchiveExtensions {
		if sext == ext {
			return strings.TrimSuffix(s, filepath.Ext(s))
		}
	}
	return s
}
