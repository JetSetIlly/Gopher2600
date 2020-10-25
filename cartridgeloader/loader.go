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
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// Loader is used to specify the cartridge to use when Attach()ing to
// the VCS. it also permits the called to specify the mapping of the cartridge
// (if necessary. fingerprinting is pretty good).
type Loader struct {
	// filename of cartridge to load.
	Filename string

	// empty string or "AUTO" indicates automatic fingerprinting
	Mapping string

	// expected hash of the loaded cartridge. empty string indicates that the
	// hash is unknown and need not be validated. after a load operation the
	// value will be the hash of the loaded data
	//
	// in the case of sound data (IsSoundData is true) then the hash is of the
	// original binary file not he decoded PCM data
	Hash string

	// copy of the loaded data. subsequence calls to Load() will return a copy
	// of this data
	Data []byte

	// does the Data field consist of sound (PCM) data
	IsSoundData bool

	// callback function when cartridge has been successfully inserted/loaded.
	// not all cartridge formats support this
	//
	// !!TODO: all cartridge formats to support OnLoaded() callback (for completeness)
	OnLoaded func(cart mapper.CartMapper) error
}

// NewLoader is the preferred method of initialisation for the Loader type.
//
// The mapping argument will be used to set the Mapping field, unless the
// argument is either "AUTO" or the empty string. In which case the file
// extension is used to set the field.
//
// File extensions should be the same as the ID of the intended mapper, as
// defined in the cartridge package. The exception is the DPC+ format which
// requires the file extension "DP+"
//
// File extensions ".BIN" and "A26" will set the Mapping field to "AUTO".
//
// Alphabetic characters in file extensions can be in upper or lower case or a
// mixture of both.
func NewLoader(filename string, mapping string) Loader {
	cl := Loader{
		Filename: filename,
		Mapping:  "AUTO",
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping != "AUTO" && mapping != "" {
		cl.Mapping = mapping
	} else {
		ext := strings.ToUpper(path.Ext(filename))
		switch ext {
		case ".BIN":
			fallthrough
		case ".ROM":
			fallthrough
		case ".A26":
			cl.Mapping = "AUTO"
		case ".2k":
			fallthrough
		case ".4k":
			fallthrough
		case ".F8":
			fallthrough
		case ".F6":
			fallthrough
		case ".F4":
			fallthrough
		case ".2k+":
			fallthrough
		case ".4k+":
			fallthrough
		case ".F8+":
			fallthrough
		case ".F6+":
			fallthrough
		case ".F4+":
			fallthrough
		case ".FA":
			fallthrough
		case ".FE":
			fallthrough
		case ".E0":
			fallthrough
		case ".E7":
			fallthrough
		case ".3F":
			fallthrough
		case ".AR":
			fallthrough
		case ".DF":
			fallthrough
		case ".3E":
			fallthrough
		case ".3E+":
			fallthrough
		case ".DPC":
			cl.Mapping = ext[1:]
		case ".DP+":
			cl.Mapping = "DPC+"
		case ".WAV":
			fallthrough
		case ".MP3":
			cl.Mapping = "AR"
			cl.IsSoundData = true
		}
	}

	return cl
}

// FileExtensions is the list of file extensions that are recognised by the
// cartridgeloader package.
var FileExtensions = [...]string{".BIN", ".ROM", ".A26", ".2k", ".4k", ".F8", ".F6", ".F4", ".2k+", ".4k+", ".F8+", ".F6+", ".F4+", ".FA", ".FE", ".E0", ".E7", ".3F", ".AR", ".DF", "3E", "3E+", ".DPC", ".DP+", ".WAV", ".MP3"}

// ShortName returns a shortened version of the CartridgeLoader filename.
func (cl Loader) ShortName() string {
	shortCartName := path.Base(cl.Filename)
	shortCartName = strings.TrimSuffix(shortCartName, path.Ext(cl.Filename))
	return shortCartName
}

// HasLoaded returns true if Load() has been successfully called.
func (cl Loader) HasLoaded() bool {
	return len(cl.Data) > 0
}

// Load the cartridge data and return as a byte array. Loader filenames with a
// valid schema will use that method to load the data. Currently supported
// schemes are HTTP and local files.
func (cl *Loader) Load() error {
	if len(cl.Data) > 0 {
		// !!TODO: already-loaded error?
		return nil
	}

	scheme := "file"

	url, err := url.Parse(cl.Filename)
	if err == nil {
		scheme = url.Scheme
	}

	switch scheme {
	case "http":
		fallthrough
	case "https":
		resp, err := http.Get(cl.Filename)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
		defer resp.Body.Close()

		cl.Data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}

	case "file":
		fallthrough

	case "":
		f, err := os.Open(cl.Filename)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
		defer f.Close()

		// get file info. not using Stat() on the file handle because the
		// windows version (when running under wine) does not handle that
		cfi, err := os.Stat(cl.Filename)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
		size := cfi.Size()

		cl.Data = make([]byte, size)
		_, err = f.Read(cl.Data)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}

	default:
		return curated.Errorf("cartridgeloader: %v", fmt.Sprintf("unsupported URL scheme (%s)", scheme))
	}

	// generate hash
	hash := fmt.Sprintf("%x", sha1.Sum(cl.Data))

	// check for hash consistency
	if cl.Hash != "" && cl.Hash != hash {
		return curated.Errorf("cartridgeloader: %v", "unexpected hash value")
	}

	// not generated hash
	cl.Hash = hash

	return nil
}
