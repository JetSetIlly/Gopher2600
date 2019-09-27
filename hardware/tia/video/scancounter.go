package video

// scancounter is the mechanism for outputting player sprite pixels. it is the
// equivalent of the enclockifier type used by the ball and missile sprite.
// scancounter is used only by the player sprite
//
// once a player sprite has reached a START signal during its polycounter
// cycle, the scanCounter is started and is ticked forward every cycle (subject
// to MOTCK, HMOVE and NUSIZ rules)
type scanCounter struct {
	nusiz   *uint8
	latches int

	// pixel counts from 7 to -1 for a total of 8 active pixels. we're counting
	// backwards because it is more convenient for the Pixel() function
	pixel int

	// for the wider player sizes, real ticks are only made every two or every
	// four clocks. pixelCt counts how many ticks the scanCounter has been on
	// the current pixel value
	pixelCt int

	// which copy of the sprite is being drawn. value of zero means the primary
	// copy is being drawn (if enable is true)
	cpy int
}

func (sc *scanCounter) start() {
	if *sc.nusiz == 0x05 || *sc.nusiz == 0x07 {
		sc.latches = 2
	} else {
		sc.latches = 1
	}
}

func (sc scanCounter) active() bool {
	return sc.pixel != -1
}

func (sc scanCounter) isLatching() bool {
	return sc.latches > 0
}

// isMissileMiddle is used by missile sprite as part of the reset-to-player
// implementation
func (sc scanCounter) isMissileMiddle() bool {
	switch *sc.nusiz {
	case 0x05:
		return sc.pixel == 3 && sc.pixelCt == 0
	case 0x07:
		return sc.pixel == 5 && sc.pixelCt == 3
	}
	return sc.pixel == 2
}

func (sc *scanCounter) tick(nextPixel bool) {
	if sc.latches > 0 {
		sc.latches--
		if sc.latches == 0 {
			sc.pixel = 7
		}
	} else if sc.pixel >= 0 {
		if nextPixel {
			sc.pixelCt = 0
			sc.pixel--
		} else {
			sc.pixelCt++
		}
	}
}
