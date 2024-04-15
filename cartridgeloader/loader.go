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
	"os"
	"path/filepath"
	"slices"
	"strings"
)

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

	// hashes of data. will be empty if data is being streamed
	HashSHA1 string
	HashMD5  string

	// does the Data field consist of sound (PCM) data
	IsSoundData bool

	// cartridge data
	data       []byte
	dataReader io.ReadSeeker

	// data was supplied through NewLoaderFromData()
	embedded bool
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
func NewLoaderFromFilename(filename string, mapping string) (Loader, error) {
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
		if err != nil {
			return Loader{}, fmt.Errorf("loader: %w", err)
		}
		if ok {
			ld.Mapping = "MVC"
		}
	}

	err = ld.open()
	if err != nil {
		return Loader{}, fmt.Errorf("loader: %w", err)
	}

	// decide on the name for this cartridge
	ld.Name = decideOnName(ld)

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
func NewLoaderFromData(name string, data []byte, mapping string) (Loader, error) {
	if len(data) == 0 {
		return Loader{}, fmt.Errorf("loader: emebedded data is empty")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Loader{}, fmt.Errorf("loader: no name for embedded data")
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	ld := Loader{
		Filename:   name,
		Mapping:    mapping,
		data:       data,
		dataReader: bytes.NewReader(data),
		HashSHA1:   fmt.Sprintf("%x", sha1.Sum(data)),
		HashMD5:    fmt.Sprintf("%x", md5.Sum(data)),
		embedded:   true,
	}

	// decide on the name for this cartridge
	ld.Name = decideOnName(ld)

	return ld, nil
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
	ld.data = nil

	if closer, ok := ld.dataReader.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}
	}

	return nil
}

// Implements the io.Reader interface.
func (ld Loader) Read(p []byte) (int, error) {
	if ld.dataReader != nil {
		return ld.dataReader.Read(p)
	}
	return 0, io.EOF
}

// Implements the io.Seeker interface.
func (ld Loader) Seek(offset int64, whence int) (int64, error) {
	if ld.dataReader != nil {
		return ld.dataReader.Seek(offset, whence)
	}
	return 0, io.EOF
}

// Size returns the size of the cartridge data in bytes
func (ld Loader) Size() int {
	return len(ld.data)
}

// Contains returns true if subslice appears anywhere in the data
func (ld Loader) Contains(subslice []byte) bool {
	return bytes.Contains(ld.data, subslice)
}

// ContainsLimit returns true if subslice appears in the data at an offset between
// zero and limit
func (ld Loader) ContainsLimit(limit int, subslice []byte) bool {
	limit = min(limit, ld.Size())
	return bytes.Contains(ld.data[:limit], subslice)
}

// Count returns the number of non-overlapping instances of subslice in the data
func (ld Loader) Count(subslice []byte) int {
	return bytes.Count(ld.data, subslice)
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

		ld.data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}

	case "file":
		fallthrough

	default:
		f, err := os.Open(ld.Filename)
		if err != nil {
			return fmt.Errorf("loader: %w", err)
		}

		fs, err := f.Stat()
		if err != nil {
			f.Close()
			return fmt.Errorf("loader: %w", err)
		}

		if fs.Size() < 1048576 {
			defer f.Close()
			ld.data, err = io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("loader: %w", err)
			}
			ld.dataReader = bytes.NewReader(ld.data)
		} else {
			ld.dataReader = f
		}
	}

	// generate hashes
	ld.HashSHA1 = fmt.Sprintf("%x", sha1.Sum(ld.data))
	ld.HashMD5 = fmt.Sprintf("%x", md5.Sum(ld.data))

	return nil
}
