package renderers

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/errors"
	"gopher2600/television"
)

// DigestTV is a television implementation that
type DigestTV struct {
	television.Television
	screenData []byte
	digest     [sha1.Size]byte
}

// NewDigestTV initialises a new instance of DigestTV
func NewDigestTV(tvType string, tv television.Television) (*DigestTV, error) {
	var err error

	// set up digest tv
	dtv := new(DigestTV)

	// create or attach television implementation
	if tv == nil {
		dtv.Television, err = television.NewBasicTelevision(tvType)
		if err != nil {
			return nil, err
		}
	} else {
		// check that the quoted tvType matches the specification of the
		// supplied BasicTelevision instance. we don't really need this but
		// becuase we're implying that tvType is required, even when an
		// instance of BasicTelevision has been supplied, the caller may be
		// expecting an error
		if tvType != tv.GetSpec().ID {
			return nil, errors.NewFormattedError(errors.DigestTV, "trying to piggyback a tv of a different spec")
		}
		dtv.Television = tv
	}

	// register ourselves as a television.Renderer
	dtv.AddRenderer(dtv)

	// memory for screenData has to be sufficient for the entirety of the
	// screen plus the size of a fingerprint. we'll use the additional space to
	// chain fingerprint hashes
	dtv.screenData = make([]byte, len(dtv.digest)+((dtv.GetSpec().ClocksPerScanline+1)*(dtv.GetSpec().ScanlinesTotal+1)*3))

	return dtv, nil
}

// NewFrame implements television.Renderer interface
func (dtv *DigestTV) NewFrame() error {
	// chain fingerprints by copying the value of the last fingerprint
	// to the head of the screen data
	copy(dtv.screenData, dtv.digest[:len(dtv.digest)])
	dtv.digest = sha1.Sum(dtv.screenData)
	return nil
}

// NewScanline implements television.Renderer interface
func (dtv *DigestTV) NewScanline() error {
	return nil
}

// SetPixel implements television.Renderer interface
func (dtv *DigestTV) SetPixel(x, y int32, red, green, blue byte, vblank bool) error {
	// preserve the first few bytes for a chained fingerprint
	offset := len(dtv.digest)

	offset += dtv.GetSpec().ClocksPerScanline * int(y) * 3
	offset += int(x) * 3

	// allow indexing to naturally fail if offset is too big

	dtv.screenData[offset] = red
	dtv.screenData[offset+1] = green
	dtv.screenData[offset+2] = blue

	return nil
}

// SetAltPixel implements television.Renderer interface
func (dtv *DigestTV) SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return nil
}

func (dtv DigestTV) String() string {
	return fmt.Sprintf("%x", dtv.digest)
}

// ResetDigest resets the current digest value to 0
func (dtv *DigestTV) ResetDigest() {
	for i := range dtv.digest {
		dtv.digest[i] = 0
	}
}
