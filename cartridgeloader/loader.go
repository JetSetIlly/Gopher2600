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

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/fs"
)

// Loader is used to specify the cartridge to use when Attach()ing to
// the VCS. it also permits the called to specify the mapping of the cartridge
// (if necessary. fingerprinting is pretty good).
type Loader struct {
	// filename of cartridge to load. In the case of embedded data, this field
	// will contain the name of the data provided to the the
	// NewLoaderFromEmbed() function.
	Filename string

	// empty string or "AUTO" indicates automatic fingerprinting
	Mapping string

	// the Mapping value that was used to initialise the loader
	RequestedMapping string

	// any detected TV spec in the filename. will be the empty string if
	// nothing is found. note that the empty string is treated like "AUTO" by
	// television.SetSpec().
	Spec string

	// expected hash of the loaded cartridge. empty string indicates that the
	// hash is unknown and need not be validated. after a load operation the
	// value will be the hash of the loaded data
	//
	// in the case of sound data (IsSoundData is true) then the hash is of the
	// original binary file not he decoded PCM data
	//
	// the value of Hash will be checked on a call to Loader.Load(). if the
	// string is empty then that check passes.
	Hash string

	// HashMD5 is an alternative to hash for use with the properties package
	HashMD5 string

	// does the Data field consist of sound (PCM) data
	IsSoundData bool

	// cartridge data. empty until Load() is called unless the loader was
	// created by NewLoaderFromEmbed()
	//
	// the pointer-to-a-slice construct allows the cartridge to be
	// loaded/changed by a Loader instance that has been passed by value.
	Data *[]byte

	// whether the data was assigned during NewLoaderFromEmbed()
	embedded bool

	// for some file types streaming is necessary. nil until Load() is called
	// and the cartridge format requires streaming.
	StreamedData *os.File

	// pointer to pointer of StreamedData. this is a tricky construct but it
	// allows us to pass an instance of Loader by value but still be able to
	// close an opened stream at an "earlier" point in the code.
	//
	// if stream is nil then the data will not be streamed. if *stream is nil
	// then the stream is not open. although use the IsStreamed() function for
	// this information.
	stream **os.File
}

// sentinal error for when it is attempted to create a loader with no filename
var NoFilename = errors.New("no filename")

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
//
// Filenames can contain whitespace, including leading and trailing whitespace,
// but cannot consists only of whitespace.
func NewLoader(filename string, mapping string) (Loader, error) {
	// check filename but don't change it. we don't want to allow the empty
	// string or a string only consisting of whitespace, but we do want to
	// allow filenames with leading/trailing spaces
	if strings.TrimSpace(filename) == "" {
		return Loader{}, fmt.Errorf("catridgeloader: %w", NoFilename)
	}

	// absolute path of filename
	var err error
	filename, err = fs.Abs(filename)
	if err != nil {
		return Loader{}, fmt.Errorf("catridgeloader: %w", err)
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	ld := Loader{
		Filename:         filename,
		Mapping:          mapping,
		RequestedMapping: mapping,
	}

	// create an empty slice for the Data field to refer to
	data := make([]byte, 0)
	ld.Data = &data

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
		ok, err := fingerprintMovieCart(filename)
		if err != nil {
			return Loader{}, fmt.Errorf("catridgeloader: %w", err)
		}
		if ok {
			ld.Mapping = "MVC"
		}
	}

	// create stream pointer only for streaming sources. these file formats are
	// likely to be very large by comparison to regular cartridge files.
	if ld.Mapping == "MVC" || (ld.Mapping == "AR" && ld.IsSoundData) {
		ld.stream = new(*os.File)
	}

	ld.Spec = specification.SearchSpec(filename)

	return ld, nil
}

// special handling for MVC files without the MVC file extension
func fingerprintMovieCart(filename string) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, fmt.Errorf("cartridgeloader: %w", err)
	}
	b := make([]byte, 4)
	f.Read(b)
	f.Close()
	if bytes.Compare(b, []byte{'M', 'V', 'C', 0x00}) == 0 {
		return true, nil
	}
	return false, nil
}

// NewLoaderFromEmbed initialises a loader with an array of bytes. Suitable for
// loading embedded data (using go:embed for example) into the emulator.
//
// The mapping argument should indicate the format of the data or "AUTO" to
// indicate that the emulator can perform a fingerprint.
//
// The name argument should not include a file extension because it won't be
// used.
func NewLoaderFromEmbed(name string, data []byte, mapping string) (Loader, error) {
	if len(data) == 0 {
		return Loader{}, fmt.Errorf("catridgeloader: emebedded data is empty")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Loader{}, fmt.Errorf("catridgeloader: no name for embedded data")
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	return Loader{
		Filename:         name,
		Mapping:          mapping,
		RequestedMapping: mapping,
		Data:             &data,
		embedded:         true,
		Hash:             fmt.Sprintf("%x", sha1.Sum(data)),
		HashMD5:          fmt.Sprintf("%x", md5.Sum(data)),
	}, nil
}

// Close should be called before disposing of a Loader instance.
func (ld Loader) Close() error {
	if ld.stream == nil || *ld.stream == nil {
		return nil
	}

	err := (**ld.stream).Close()
	*ld.stream = nil
	if err != nil {
		return fmt.Errorf("cartridgeloader: %w", err)
	}
	logger.Logf("cartridgeloader", "stream closed (%s)", ld.Filename)

	return nil
}

// ShortName returns a shortened version of the CartridgeLoader filename field.
// In the case of embedded data, the filename field will be returned unaltered.
func (ld Loader) ShortName() string {
	if ld.embedded {
		return ld.Filename
	}

	// return the empty string if filename is undefined
	if len(strings.TrimSpace(ld.Filename)) == 0 {
		return ""
	}

	sn := filepath.Base(ld.Filename)
	sn = strings.TrimSuffix(sn, filepath.Ext(ld.Filename))
	return sn
}

// IsStreaming returns two booleans. The first will be true if Loader was
// created for what appears to be a streaming source, and the second will be
// true if the stream has been open.
func (ld Loader) IsStreaming() (bool, bool) {
	return ld.stream != nil, ld.stream != nil && *ld.stream != nil
}

// IsEmbedded returns true if Loader was created from embedded data. If data
// has a length of zero then this function will return false.
func (ld Loader) IsEmbedded() bool {
	return ld.embedded && len(*ld.Data) > 0
}

// Load the cartridge data and return as a byte array. Loader filenames with a
// valid schema will use that method to load the data. Currently supported
// schemes are HTTP and local files.
func (ld *Loader) Load() error {
	// data is already "loaded" when using embedded data
	if ld.embedded {
		return nil
	}

	if ld.stream != nil {
		err := ld.Close()
		if err != nil {
			return fmt.Errorf("cartridgeloader: %w", err)
		}

		ld.StreamedData, err = os.Open(ld.Filename)
		if err != nil {
			return fmt.Errorf("cartridgeloader: %w", err)
		}
		logger.Logf("cartridgeloader", "stream open (%s)", ld.Filename)

		*ld.stream = ld.StreamedData

		return nil
	}

	if ld.Data != nil && len(*ld.Data) > 0 {
		return nil
	}

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
			return fmt.Errorf("cartridgeloader: %w", err)
		}
		defer resp.Body.Close()

		*ld.Data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cartridgeloader: %w", err)
		}

	case "file":
		fallthrough

	case "":
		fallthrough

	default:
		f, err := os.Open(ld.Filename)
		if err != nil {
			return fmt.Errorf("cartridgeloader: %w", err)
		}
		defer f.Close()

		*ld.Data, err = io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("cartridgeloader: %w", err)
		}
	}

	// generate hash and check for consistency
	hash := fmt.Sprintf("%x", sha1.Sum(*ld.Data))
	if ld.Hash != "" && ld.Hash != hash {
		return fmt.Errorf("cartridgeloader: unexpected hash value")
	}
	ld.Hash = hash

	// generate md5 hash and check for consistency
	hashmd5 := fmt.Sprintf("%x", md5.Sum(*ld.Data))
	if ld.HashMD5 != "" && ld.HashMD5 != hashmd5 {
		return fmt.Errorf("cartridgeloader: unexpected hash value")
	}
	ld.HashMD5 = hashmd5

	return nil
}
