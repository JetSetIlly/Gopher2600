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

func (sp sprite) String() string {
	return fmt.Sprintf("%v\n%v\n%v\n", sp.position, sp.drawSig, sp.resetDelay)
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

func (ps position) String() string {
	if ps.polycounter.Count == ps.polycounter.ResetPoint {
		return fmt.Sprintf("position: %s <- drawing in %d", ps.polycounter, polycounter.MaxPhase-ps.polycounter.Phase+1)
	} else if ps.polycounter.Count == ps.polycounter.ResetPoint {
		return fmt.Sprintf("position: %s <- drawing start", ps.polycounter)
	}
	return fmt.Sprintf("position: %s", ps.polycounter)
}

func (ps *position) synchronise(cc *colorclock.ColorClock) {
	ps.polycounter.Reset()
	ps.coarsePixel = cc.Pixel()
}

func (ps *position) tick() bool {
	if ps.polycounter.Tick(false) == true {
		return true
	}
	return false
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
	count        int
	delayedReset bool
}

func newDrawSig() *drawSig {
	ds := new(drawSig)
	if ds == nil {
		return nil
	}
	ds.count = 8
	return ds
}

func (ds drawSig) isRunning() bool {
	return ds.count < 8
}

func (ds drawSig) String() string {
	if ds.isRunning() {
		return fmt.Sprintf(" drawsig: inactive")
	}
	return fmt.Sprintf(" drawsig: %d cycle(s) remaining", 8-ds.count+1)
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
