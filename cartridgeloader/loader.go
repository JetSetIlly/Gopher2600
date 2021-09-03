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
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
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
	Hash string

	// does the Data field consist of sound (PCM) data
	IsSoundData bool

	// cartridge data. empty until Load() is called unless the loader was
	// created by NewLoaderFromEmbed()
	Data []byte

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

	// callback function from the cartridge to the VCS. used for example. when
	// cartridge has been successfully inserted. not all cartridge formats
	// support/require this
	//
	// if the cartridge mapper needs to communicate more information then the
	// action string should be used
	VCSHook VCSHook
}

// VCSHook function signature. Used for direct communication between a
// cartridge mapper and the core emulation. Not often used but necessary for
// (currently):
//
//		. Supercharger (tape start/end, fastload)
//		. PlusROM (new installation)
//
// The emulation must understand how to interpret the event.
type VCSHook func(cart mapper.CartMapper, event mapper.Event, args ...interface{}) error

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
		return Loader{}, curated.Errorf("catridgeloader: no filename")
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	cl := Loader{
		Filename:         filename,
		Mapping:          mapping,
		RequestedMapping: mapping,
		VCSHook: func(cart mapper.CartMapper, event mapper.Event, args ...interface{}) error {
			return nil
		},
	}

	if mapping == "AUTO" {
		ext := strings.ToUpper(filepath.Ext(filename))
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
		case ".E3P":
			fallthrough // synonym for 3E+
		case ".E3+":
			fallthrough // synonym for 3E+
		case ".3E+":
			fallthrough
		case ".SB":
			fallthrough
		case ".DPC":
			cl.Mapping = ext[1:]
		case ".DP+":
			cl.Mapping = "DPC+"
		case "CDF":
			cl.Mapping = "CDF"
		case ".WAV":
			fallthrough
		case ".MP3":
			cl.Mapping = "AR"
			cl.IsSoundData = true
		case ".MVC":
			cl.Mapping = "MVC"
		}
	}

	// create stream pointer only for streaming sources. these file formats are
	// likely to be very large by comparison to regular cartridge files.
	if cl.Mapping == "MVC" || (cl.Mapping == "AR" && cl.IsSoundData) {
		cl.stream = new(*os.File)
	}

	cl.Spec = specification.SearchSpec(filename)

	return cl, nil
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
		return Loader{}, curated.Errorf("catridgeloader: emebedded data is empty")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Loader{}, curated.Errorf("catridgeloader: no name for embedded data")
	}

	mapping = strings.TrimSpace(strings.ToUpper(mapping))
	if mapping == "" {
		mapping = "AUTO"
	}

	return Loader{
		Filename:         name,
		Mapping:          mapping,
		RequestedMapping: mapping,
		Data:             data,
		embedded:         true,
		Hash:             fmt.Sprintf("%x", sha1.Sum(data)),
		VCSHook: func(cart mapper.CartMapper, event mapper.Event, args ...interface{}) error {
			return nil
		},
	}, nil
}

// Close should be called before disposing of a Loader instance.
func (cl Loader) Close() error {
	if cl.stream == nil || *cl.stream == nil {
		return nil
	}

	err := (**cl.stream).Close()
	*cl.stream = nil
	if err != nil {
		return curated.Errorf("cartridgeloader: %v", err)
	}
	logger.Logf("cartridgeloader", "stream closed (%s)", cl.Filename)

	return nil
}

// ShortName returns a shortened version of the CartridgeLoader filename field.
// In the case of embedded data, the filename field will be returned unaltered.
func (cl Loader) ShortName() string {
	if cl.embedded {
		return cl.Filename
	}

	// return the empty string if filename is undefined
	if len(strings.TrimSpace(cl.Filename)) == 0 {
		return ""
	}

	sn := filepath.Base(cl.Filename)
	sn = strings.TrimSuffix(sn, filepath.Ext(cl.Filename))
	return sn
}

// IsStreaming returns two booleans. The first will be true if Loader was
// created for what appears to be a streaming source, and the second will be
// true if the stream has been open.
func (cl Loader) IsStreaming() (bool, bool) {
	return cl.stream != nil, cl.stream != nil && *cl.stream != nil
}

// IsEmbedded returns true if Loader was created from embedded data. If data
// has a length of zero then this function will return false.
func (cl Loader) IsEmbedded() bool {
	return cl.embedded && len(cl.Data) > 0
}

// Load the cartridge data and return as a byte array. Loader filenames with a
// valid schema will use that method to load the data. Currently supported
// schemes are HTTP and local files.
func (cl *Loader) Load() error {
	// data is already "loaded" when using embedded data
	if cl.embedded {
		return nil
	}

	if cl.stream != nil {
		err := cl.Close()
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}

		cl.StreamedData, err = os.Open(cl.Filename)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
		logger.Logf("cartridgeloader", "stream open (%s)", cl.Filename)

		*cl.stream = cl.StreamedData

		return nil
	}

	if len(cl.Data) > 0 {
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

		cl.Data, err = io.ReadAll(resp.Body)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}

	case "file":
		fallthrough

	case "":
		fallthrough

	default:
		f, err := os.Open(cl.Filename)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
		defer f.Close()

		cl.Data, err = io.ReadAll(f)
		if err != nil {
			return curated.Errorf("cartridgeloader: %v", err)
		}
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
