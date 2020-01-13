package sdldebug

const pixelDepth = 4
const pixelWidth = 2.0

// the pixels type stores all pixel information for the screen.
//
// the overlay type looks after it's own pixels.
type pixels struct {
	l       int
	regular []byte
	alt     []byte
	clr     []byte
}

// create a new instance of the pixels type. called everytime the screen
// dimensions change.
func newPixels(w, h int) *pixels {
	l := w * h * pixelDepth

	pxl := &pixels{
		l:       l,
		regular: make([]byte, l),
		alt:     make([]byte, l),
		clr:     make([]byte, l),
	}

	// set alpha bit for regular and alt pixels to opaque. we'll be changing
	// this value during clear() and setPixel() operations but it's important
	// we set it to opaque for when we first use the pixels, or we'll get to
	// see nasty artefacts on the screen.
	for i := pixelDepth - 1; i < l; i += pixelDepth {
		pxl.regular[i] = 255
		pxl.alt[i] = 255
	}

	return pxl
}

func (pxl pixels) length() int {
	return pxl.l
}

func (pxl pixels) clear() {
	copy(pxl.regular, pxl.clr)
	copy(pxl.alt, pxl.clr)
}
