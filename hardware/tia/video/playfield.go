package video

// TickPlayfield moves playfield on one video cycle
func (vd *Video) TickPlayfield() {
	if vd.colorClock.MatchBeginning(17) {
		vd.pfTickPhase = 1
		vd.pfTickCt = 0
	} else if vd.colorClock.MatchBeginning(37) {
		vd.pfTickPhase = 2
		vd.pfTickCt = 0
	} else if vd.colorClock.MatchBeginning(0) {
		vd.pfTickPhase = 0
	} else if vd.pfTickPhase != 0 && vd.colorClock.Phase == 0 {
		vd.pfTickCt++
	}
}

// PixelPlayfield returns the color of the playfield at the current time.
// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
func (vd *Video) PixelPlayfield() (bool, uint8) {
	if vd.pfTickPhase != 0 {
		if vd.pfTickPhase == 1 || !vd.ctrlpfReflection {
			if vd.pf[vd.pfTickCt] {
				return true, vd.colupf
			}
		} else {
			// TODO: reflected playfield
		}
	}
	return false, 0
}
