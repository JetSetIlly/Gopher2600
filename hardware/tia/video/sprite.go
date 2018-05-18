package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
	"gopher2600/hardware/tia/polycounter"
)

// the sprite type is used for those video elements that move about - players,
// missiles and the ball. the VCS doesn't really have anything called a sprite
// but we all know what it means
//
// two functions for the sprite type which would be undoubtedly be useful would
// be Tick() and Pixel(). however, each sprite in the VCS reacts slightly
// differently to draw signals and reset delays. we abbrogate responsibility
// for sprite-level ticking therefore, to functions not attached to the sprite
// class. For example, see TickBall() and PixelBall(); TickPlayer() and
// PixelPlayer(); and TickMissile() and PixelMissile()

type sprite struct {
	position   *position
	drawSig    *drawSig
	resetDelay *delayCounter

	// because we use the sprite type in more than one context we need some way
	// of providing String() output with a helpful label
	label string

	enableFlag *bool
}

// newSprite takes an optional argument, enableFlag. this is a pointer to the
// boolean flag that controls the presence of the sprite on screen. if the
// sprite does not have an enableFlag (player sprites) then pass a nil
// pointer
func newSprite(label string, enableFlag *bool) *sprite {
	sp := new(sprite)
	if sp == nil {
		return nil
	}

	sp.label = label
	sp.enableFlag = enableFlag

	sp.position = newPosition()
	if sp.position == nil {
		return nil
	}

	sp.drawSig = newDrawSig()
	if sp.drawSig == nil {
		return nil
	}

	sp.resetDelay = newDelayCounter("reset")
	if sp.resetDelay == nil {
		return nil
	}

	return sp
}

// MachineInfoTerse returns the sprite information in terse format
func (sp sprite) MachineInfoTerse() string {
	enableStr := ""
	if sp.enableFlag != nil && *sp.enableFlag == true {
		enableStr = "(+)"
	} else if sp.enableFlag != nil && *sp.enableFlag == false {
		enableStr = "(-)"
	}
	return fmt.Sprintf("%s%s: %s %s %s", sp.label, enableStr, sp.position.MachineInfoTerse(), sp.drawSig.MachineInfoTerse(), sp.resetDelay.MachineInfoTerse())
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	enableStr := ""
	if sp.enableFlag != nil && *sp.enableFlag == true {
		enableStr = "enabled"
	} else if sp.enableFlag != nil && *sp.enableFlag == false {
		enableStr = "disabled"
	}
	return fmt.Sprintf("%s: %s, %v\n %v\n %v", sp.label, enableStr, sp.position, sp.drawSig, sp.resetDelay)
}

// map String to MachineInfo
func (sp sprite) String() string {
	return sp.MachineInfo()
}

// the position type is only used by the sprite type

type position struct {
	polycounter polycounter.Polycounter

	// coarsePixel is the pixel value of the color clock when position.reset()
	// was last called
	coarsePixel int
}

func newPosition() *position {
	ps := new(position)
	if ps == nil {
		return nil
	}
	ps.polycounter.SetResetPattern("101101")
	return ps
}

// MachineInfoTerse returns the position information in terse format
func (ps position) MachineInfoTerse() string {
	return fmt.Sprintf("pos=%d", ps.coarsePixel)
}

// MachineInfo returns the position information in verbose format
func (ps position) MachineInfo() string {
	s := fmt.Sprintf("reset at pixel %d", ps.coarsePixel)
	return fmt.Sprintf("%s\nposition: %s", s, ps.polycounter)
}

// map String to Machine Info
func (ps position) String() string {
	return ps.MachineInfo()
}

func (ps *position) resetPosition(cc *colorclock.ColorClock) {
	ps.polycounter.Reset()
	ps.coarsePixel = cc.Pixel()
}

func (ps *position) tick(triggerList []int) bool {
	if ps.polycounter.Tick(false) == true {
		return true
	}

	if triggerList != nil {
		for _, v := range triggerList {
			if v == ps.polycounter.Count && ps.polycounter.Phase == 0 {
				return true
			}
		}
	}

	return false
}

// the drawSig type is only used by the sprite type

type drawSig struct {
	// the direction of count and maxCount is important - don't monkey with it
	// the value is used in Pixel*() functions to determine which pixel to check
	maxCount int
	count    int

	delayedReset bool
}

func newDrawSig() *drawSig {
	ds := new(drawSig)
	if ds == nil {
		return nil
	}
	ds.maxCount = 8
	ds.count = ds.maxCount + 1
	return ds
}

// MachineInfoTerse returns the draw signal information in terse format
func (ds drawSig) MachineInfoTerse() string {
	if ds.isActive() {
		return fmt.Sprintf("dsig=%d", ds.maxCount-ds.count+1)
	}
	return "dsig=-"
}

// MachineInfo returns the draw signal information in verbose format
func (ds drawSig) MachineInfo() string {
	if ds.isActive() {
		return fmt.Sprintf("drawsig: pixel %d", ds.maxCount-ds.count+1)
	}
	return fmt.Sprintf("drawsig: inactive")
}

// map String to MachineInfo
func (ds drawSig) String() string {
	return ds.MachineInfo()
}

func (ds drawSig) isActive() bool {
	return ds.count <= ds.maxCount
}

func (ds *drawSig) tick() {
	if ds.isActive() && !ds.delayedReset {
		ds.count++
	}
}

// confirmDelay confirms that the reset has been delayed
func (ds *drawSig) confirmDelay() {
	ds.delayedReset = true
}

// start begins the draw signal
func (ds *drawSig) start() {
	ds.count = 0
	ds.delayedReset = false
}
