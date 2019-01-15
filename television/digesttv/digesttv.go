package digesttv

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/television"
)

// DigestTV is a television implementation that
type DigestTV struct {
	television.HeadlessTV
	screenData []byte
	digest     [sha1.Size]byte
}

// NewDigestTV initialises a new instance of DigestTV
func NewDigestTV(tvType string) (*DigestTV, error) {
	tv := new(DigestTV)

	err := television.InitHeadlessTV(&tv.HeadlessTV, tvType)
	if err != nil {
		return nil, err
	}

	// register new frame callback from HeadlessTV to SDLTV
	// leaving SignalNewScanline() hook at its default
	tv.HookNewFrame = tv.newFrame
	tv.HookSetPixel = tv.setPixel

	// memory for screenData has to be sufficient for the entirity of the
	// screnn plus the size of a fingerprint. we'll use the additional space to
	// chain fingerprint hashes
	tv.screenData = make([]byte, len(tv.digest)+((tv.Spec.ClocksPerScanline+1)*(tv.Spec.ScanlinesTotal+1)*3))

	return tv, nil
}

func (tv *DigestTV) newFrame() error {
	// chain fingerprints by copying the value of the last fingerprint
	// to the head of the screen data
	copy(tv.screenData, tv.digest[:len(tv.digest)])
	tv.digest = sha1.Sum(tv.screenData)
	return nil
}

func (tv *DigestTV) setPixel(x, y int32, red, green, blue byte, vblank bool) error {
	// preserve the first few bytes for a chained fingerprint
	offset := len(tv.digest)

	offset += tv.Spec.ClocksPerScanline * int(y) * 3
	offset += int(x) * 3
	if offset < len(tv.screenData) {
		tv.screenData[offset] = red
		tv.screenData[offset+1] = green
		tv.screenData[offset+2] = blue
	} else {
		fmt.Println(tv.FrameNum.Value(), offset)
	}
	return nil
}

func (tv DigestTV) String() string {
	return fmt.Sprintf("%x", tv.digest)
}
