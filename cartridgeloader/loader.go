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
