package video

// TickBall moves the counters along for the ball sprite
func (vd *Video) TickBall() {
	// position
	if vd.Ball.position.tick(nil) == true {
		vd.Ball.drawSig.start()
	} else {
		vd.Ball.drawSig.tick()
	}

	// reset
	if vd.Ball.resetDelay.tick() == true {
		vd.Ball.position.resetPosition(vd.colorClock)
		vd.Ball.drawSig.start()
	}

	// enable
	if vd.enablDelay.tick() == true {
		vd.enablPrev = vd.enabl
		vd.enabl = vd.enablDelay.payloadValue.(bool)
	}
}

// PixelBall returns the color of the ball at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (vd *Video) PixelBall() (bool, uint8) {
	// ball should be pixelled if:
	//  o ball is enabled and vertical delay is not enabled
	//  o OR ball was previously enabled and vertical delay is enabled
	//  o AND a reset signal (RESBL) has not recently been triggered
	if ((!vd.vdelbl && vd.enabl) || (vd.vdelbl && vd.enablPrev)) && !vd.Ball.resetDelay.isRunning() {
		switch vd.Ball.drawSig.count {
		case 0:
			return true, vd.colupf
		case 1:
			if vd.ctrlpfBallSize >= 0x1 {
				return true, vd.colupf
			}
		case 2, 3:
			if vd.ctrlpfBallSize >= 0x2 {
				return true, vd.colupf
			}
		case 4, 5, 6, 7:
			if vd.ctrlpfBallSize == 0x3 {
				return true, vd.colupf
			}
		}
	}
	return false, 0
}
