// this file is part of gopher2600.
//
// gopher2600 is free software: you can redistribute it and/or modify
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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package cartridgeloader

import (
	"gopher2600/errors"
	"net/http"
	"os"
	"path"
	"strings"
)

// Loader is used to specify the cartridge to use when Attach()ing to
// the VCS. it also permits the called to specify the format of the cartridge
// (if necessary. fingerprinting is pretty good)
type Loader struct {
	Filename string

	// empty string or "AUTO" indicates automatic fingerprinting
	Format string

	// expected hash of the loaded cartridge. empty string indicates that the
	// hash is unknown and need not be validated
	Hash string

	data []byte
}

// ShortName returns a shortened version of the CartridgeLoader filename
func (cl Loader) ShortName() string {
	shortCartName := path.Base(cl.Filename)
	shortCartName = strings.TrimSuffix(shortCartName, path.Ext(cl.Filename))
	return shortCartName
}

// HasLoaded returns true if Load() has been successfully called
func (cl Loader) HasLoaded() bool {
	return len(cl.data) > 0
}

// Load the cartridge
func (cl Loader) Load() ([]byte, error) {
	if len(cl.data) > 0 {
		return cl.data, nil
	}

	var err error

	if strings.HasPrefix(cl.Filename, "http://") {
		var resp *http.Response

		resp, err = http.Get(cl.Filename)
		if err != nil {
			return nil, errors.New(errors.CartridgeLoader, cl.Filename)
		}
		defer resp.Body.Close()

		size := resp.ContentLength

		cl.data = make([]byte, size)
		_, err = resp.Body.Read(cl.data)
		if err != nil {
			return nil, errors.New(errors.CartridgeLoader, cl.Filename)
		}
	} else {
		var f *os.File
		f, err = os.Open(cl.Filename)
		if err != nil {
			return nil, errors.New(errors.CartridgeLoader, cl.Filename)
		}
		defer f.Close()

		// get file info
		cfi, err := f.Stat()
		if err != nil {
			return nil, errors.New(errors.CartridgeLoader, cl.Filename)
		}
		size := cfi.Size()

		cl.data = make([]byte, size)
		_, err = f.Read(cl.data)
		if err != nil {
			return nil, errors.New(errors.CartridgeLoader, cl.Filename)
		}
	}

	return cl.data, nil
}
