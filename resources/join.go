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
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600/resources/fs"
)

// JoinPath prepends the supplied path with a with OS/build specific base
// paths, if required.
//
// The function creates all folders necessary to reach the end of sub-path. It
// does not otherwise touch or create the file.
func JoinPath(path ...string) (string, error) {
	// join supplied path
	p := filepath.Join(path...)

	var b string

	// resources are either in the portable path or the path returned by resourcePath(). the
	// resourcePath() function depends on how the program has been compiled - as a release binary or
	// as a development binary
	if checkPortable() {
		b = portablePath
	} else {
		var err error
		b, err = resourcePath()
		if err != nil {
			return "", err
		}
	}

	// do not prepend base path if it is already present
	if !strings.HasPrefix(p, b) {
		p = filepath.Join(b, filepath.Join(path...))
	}

	// check if path already exists
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	// create path if necessary
	if err := fs.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return "", err
	}

	return p, nil
}
