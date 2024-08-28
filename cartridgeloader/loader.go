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
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jetsetilly/gopher2600/archivefs"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/properties"
)

// the maximum amount of data to preload
const maxPreloadLength = 1048576

// use this function when assigning to the Loader.preload field
func preloadLimit(data []byte) []byte {
	return data[:min(len(data), maxPreloadLength)]
}

// Loader abstracts all the ways data can be loaded into the emulation.
type Loader struct {
	io.ReadSeeker

	// the name to use for the cartridge. in the case of embedded data the name
	// will be the name provided to the NewLoaderFromData() function
	Name string

	// filename of cartridge being loaded. this is the absolute path that was
	// used to load the cartridge
	//
	// in the case of embedded data the filename will be the name provided to
	// the NewLoaderFromData() function
	//
	// use of the Filename can be useful, for example, for checking if the TV
	// specification is indicated
	Filename string

	// empty string or "AUTO" indicates automatic fingerprinting
	Mapping string

	// startup bank of cartridge
	Bank string

	// property entry from property package
	Property properties.Entry

	// requested specification for television
	ReqSpec string

	// hashes of data
	HashSHA1 string
	HashMD5  string

	// does the Data field consist of sound (PCM) data
	IsSoundData bool

	// data and size of
	data io.ReadSeeker
	size int

	// preload is the data at the beginning of the cartridge data that has been
	// preloaded immediately on creation of the cartridge loader
	//
	// in reality, most cartridges are small enough to fit entirely inside the
	// preload field. currently it is only moviecart data and supercharger sound
	// files that are ever larger than that
	//
	// the preload data is used to create the hashes
	preload []byte

	// data was supplied through NewLoaderFromData()
	embedded bool
}

// Properties is a minimal interface to the properties package
type Properties interface {
	Lookup(md5Hash string) properties.Entry
}

// sentinal error for when it is attempted to create a loader with no filename
var NoFilename = errors.New("no filename")

// NewLoaderFromFilename is the preferred method of initialisation for the
// Loader type when loading data from a filename.
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
//
// Filenames can contain whitespace, including leading and trailing whitespace,
// but cannot consist only of whitespace.
func NewLoaderFromFilename(filename string, mapping string, bank string, props Properties) (Loader, error) {
	// check filename but don't change it. we don't want to allow the empty
	// string or a string only consisting of whitespace, but we *do* want to
	// allow filenames with leading/trailing spaces
	if strings.TrimSpace(filename) == "" {
		return Loader{}, fmt.Errorf("loader: %w", NoFilename)
	}

	// absolute path of filename
	var err error
	filename, err = filepath.Abs(filename)
	if err != nil {
		return Loader{}, fmt.Errorf("loader: %w", err)
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	ld := Loader{
		Filename: filename,
		Mapping:  mapping,
		Bank:     bank,
	}

	// decide what mapping to use if the requested mapping is AUTO
	if mapping == "AUTO" {
		extension := strings.ToUpper(filepath.Ext(filename))
		if slices.Contains(autoFileExtensions, extension) {
			ld.Mapping = "AUTO"
		} else if slices.Contains(explicitFileExtensions, extension) {
			ld.Mapping = extension[1:]
		} else if slices.Contains(audioFileExtensions, extension) {
			ld.Mapping = "AR"
			ld.IsSoundData = true
		}
	}

	// if mapping value is still AUTO, make a special check for moviecart data.
	// we want to do this now so we can initialise the stream
	if ld.Mapping == "AUTO" {
		ok, err := miniFingerprintMovieCart(filename)
		if err == nil && ok {
			ld.Mapping = "MVC"
		}
	}

	err = ld.open()
	if err != nil {
		return Loader{}, err
	}

	// decide on the name for this cartridge
	ld.Name = decideOnName(ld)

	// get properties entry
	if props != nil {
		ld.Property = props.Lookup(ld.HashMD5)
	}

	// decide on TV specification
	if ld.Property.IsValid() {
		ld.ReqSpec = specification.SearchReqSpec(ld.Property.Name)
	}
	if ld.ReqSpec == "" {
		ld.ReqSpec = specification.SearchReqSpec(ld.Filename)
	}

	return ld, nil
}

// NewLoaderFromData is the preferred method of initialisation for the Loader
// type when loading data from a byte array. It's a great way of loading
// embedded data (using go:embed) into the emulator.
//
// The mapping argument should indicate the format of the data or "AUTO" to
// indicate that the emulator can perform a fingerprint.
//
// The name argument should not include a file extension because it won't be
// used.
func NewLoaderFromData(name string, data []byte, mapping string, bank string, props Properties) (Loader, error) {
	if len(data) == 0 {
		return Loader{}, fmt.Errorf("loader: data is empty")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Loader{}, fmt.Errorf("loader: no name for data")
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	ld := Loader{
		Filename: name,
		Mapping:  mapping,
		Bank:     bank,
		preload:  preloadLimit(data),
		data:     bytes.NewReader(data),
		HashSHA1: fmt.Sprintf("%x", sha1.Sum(data)),
		HashMD5:  fmt.Sprintf("%x", md5.Sum(data)),
		size:     len(data),
		embedded: true,
	}

	// decide on the name for this cartridge
	ld.Name = decideOnName(ld)

	// get properties entry
	if props != nil {
		ld.Property = props.Lookup(ld.HashMD5)
	}

	// decide on TV specification
	if ld.Property.IsValid() {
		ld.ReqSpec = specification.SearchReqSpec(ld.Property.Name)
	}
	if ld.ReqSpec == "" {
		ld.ReqSpec = specification.SearchReqSpec(ld.Filename)
	}

	return ld, nil
}

func (ld *Loader) Reload() error {
	err := ld.Close()
	if err != nil {
		return err
	}
	return ld.open()
}

// Reset prepares the loader for fresh reading. Useful to call after data has
// been Read() or if you need to make absolutely sure subsequent calls to Read()
// start from the beginning of the data stream
func (ld *Loader) Reset() error {
	_, err := ld.Seek(0, io.SeekStart)
	return err
}

// Implements the io.Closer interface.
//
// Should be called before disposing of a Loader instance.
func (ld *Loader) Close() error {
	if closer, ok := ld.data.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}
	}
	ld.data = nil
	ld.size = 0
	ld.preload = nil

	return nil
}

// Implements the io.Reader interface.
func (ld Loader) Read(p []byte) (int, error) {
	if ld.data != nil {
		return ld.data.Read(p)
	}
	return 0, io.EOF
}

// Implements the io.Seeker interface.
func (ld Loader) Seek(offset int64, whence int) (int64, error) {
	if ld.data != nil {
		return ld.data.Seek(offset, whence)
	}
	return 0, io.EOF
}

// Size returns the size of the cartridge data in bytes
func (ld Loader) Size() int {
	return ld.size
}

// Contains returns true if subslice appears anywhere in the preload data.
func (ld Loader) Contains(subslice []byte) bool {
	return bytes.Contains(ld.preload, subslice)
}

// ContainsLimit returns true if subslice appears anywhere in the preload data and
// within the byte limit value supplied as a fuction parameter.
func (ld Loader) ContainsLimit(limit int, subslice []byte) bool {
	limit = min(limit, ld.Size())
	return bytes.Contains(ld.preload[:limit], subslice)
}

// Count returns the number of non-overlapping instances of subslice in the
// preload data.
func (ld Loader) Count(subslice []byte) int {
	return bytes.Count(ld.preload, subslice)
}

// open the cartridge data. filenames with a valid schema will use that method
// to load the data. currently supported schemes are HTTP and local files.
func (ld *Loader) open() error {
	_ = ld.Close()

	scheme := "file"
	url, err := url.Parse(ld.Filename)
	if err == nil {
		scheme = url.Scheme
	}

	switch scheme {
	case "http":
		fallthrough
	case "https":
		resp, err := http.Get(ld.Filename)
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}

		ld.data = bytes.NewReader(data)
		ld.size = len(data)
		ld.preload = preloadLimit(data)

	case "file":
		fallthrough

	default:
		r, sz, err := archivefs.Open(ld.Filename)
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}

		ld.preload, err = io.ReadAll(io.LimitReader(r, maxPreloadLength))
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("loader: %w", err)
		}

		ld.data = r
		ld.size = sz
	}

	// generate hashes
	ld.HashSHA1 = fmt.Sprintf("%x", sha1.Sum(ld.preload))
	ld.HashMD5 = fmt.Sprintf("%x", md5.Sum(ld.preload))

	return nil
}
