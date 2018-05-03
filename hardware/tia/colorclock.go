package tia

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

type colorClock struct {
	polycounter.Polycounter
}

func newColorClock() *colorClock {
	cc := new(colorClock)
	cc.SetResetPattern("010100")
	if cc == nil {
		return nil
	}
	return cc
}

func (cc colorClock) String() string {
	// print polycount and VCS "pixel" equivalent
	return fmt.Sprintf("CCLOCK: %s [%dpx]", cc.ToString(), int(cc.Count*4)+cc.Phase-68)
}

// match checks whether colorClock is at the *end* of the given count
func (cc colorClock) match(count int) bool {
	return cc.Count == count && cc.Phase == 3
}
