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
	"os"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/paths/fs"
)

// ResourcePath prepends the supplied path with a with OS/build specific base
// paths
//
// The function creates all folders necessary to reach the end of sub-path. It
// does not otherwise touch or create the file.
func ResourcePath(path ...string) (string, error) {
	b, err := baseResourcePath()
	if err != nil {
		return "", err
	}

	p := filepath.Join(b, filepath.Join(path...))

	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	if err := fs.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return "", err
	}

	return p, nil
}
