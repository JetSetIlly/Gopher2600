package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
	"gopher2600/hardware/tia/polycounter"
)

// the sprite type is used for those video elements that move about - players,
// missiles and the ball. the VCS doesn't really have anything called a sprite
// but we all know what it means

type sprite struct {
	position   *position
	drawSig    *drawSig
	resetDelay *delayCounter

	// because we use the sprite type in more than one context we need some way
	// of providing String() output with a helpful label
	label string
}

func newSprite(label string) *sprite {
	sp := new(sprite)
	if sp == nil {
		return nil
	}

	sp.label = label

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
	return fmt.Sprintf("%s: %s %s %s", sp.label, sp.position.MachineInfoTerse(), sp.drawSig.MachineInfoTerse(), sp.resetDelay.MachineInfoTerse())
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	return fmt.Sprintf("%s: %v\n %v\n %v", sp.label, sp.position, sp.drawSig, sp.resetDelay)
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
	if ps.polycounter.Count == ps.polycounter.ResetPoint {
		return fmt.Sprintf("%s\nposition: %s <- drawing in %d", s, ps.polycounter, polycounter.MaxPhase-ps.polycounter.Phase+1)
	} else if ps.polycounter.Count == ps.polycounter.ResetPoint {
		return fmt.Sprintf("%s\nposition: %s <- drawing start", s, ps.polycounter)
	}
	return fmt.Sprintf("%s\nposition: %s", s, ps.polycounter)
}

// map String to Machine Info
func (ps position) String() string {
	return ps.MachineInfo()
}

func (ps *position) synchronise(cc *colorclock.ColorClock) {
	ps.polycounter.Reset()
	ps.coarsePixel = cc.Pixel()
}

func (ps *position) tick() bool {
	return ps.polycounter.Tick(false)
}

func (ps *position) tickAndTriggerList(triggerList []int) bool {
	if ps.polycounter.Tick(false) == true {
		return true
	}

	for _, v := range triggerList {
		if v == ps.polycounter.Count && ps.polycounter.Phase == 0 {
			return true
		}
	}

	return false
}

func (ps position) match(count int) bool {
	return ps.polycounter.Match(count)
}

// the drawSig type is only used by the sprite type

type drawSig struct {
	maxCount     int
	count        int
	delayedReset bool
}

func newDrawSig() *drawSig {
	ds := new(drawSig)
	if ds == nil {
		return nil
	}
	ds.maxCount = 8
	ds.count = ds.maxCount
	return ds
}

func (ds drawSig) isRunning() bool {
	return ds.count <= ds.maxCount
}

// MachineInfoTerse returns the draw signal information in terse format
func (ds drawSig) MachineInfoTerse() string {
	if ds.isRunning() {
		return fmt.Sprintf("dsig=%d", ds.maxCount-ds.count)
	}
	return "dsig=-"
}

// MachineInfo returns the draw signal information in verbose format
func (ds drawSig) MachineInfo() string {
	if ds.isRunning() {
		return fmt.Sprintf("drawsig: %d cycle(s) remaining", ds.maxCount-ds.count)
	}
	return fmt.Sprintf("drawsig: inactive")
}

// map String to MachineInfo
func (ds drawSig) String() string {
	return ds.MachineInfo()
}

func (ds *drawSig) tick() {
	if ds.isRunning() && !ds.delayedReset {
		ds.count++
	}
}

// confirm that the reset has been delayed
func (ds *drawSig) confirm() {
	ds.delayedReset = true
}

func (ds *drawSig) reset() {
	ds.count = 0
	ds.delayedReset = false
}
