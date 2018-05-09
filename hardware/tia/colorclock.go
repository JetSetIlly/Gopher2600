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

// StringTerse returns the color clock information in terse format
func (cc colorClock) StringTerse() string {
	return fmt.Sprintf("CC=%s", cc.Polycounter.StringTerse())
}

// String returns the color clock information in verbose format
func (cc colorClock) String() string {
	// print polycount and VCS "pixel" equivalent
	return fmt.Sprintf("CCLOCK: %v [%dpx]\n", cc.Polycounter, int(cc.Count*4)+cc.Phase-68)
}

// match checks whether colorClock is at the *end* of the given count
func (cc colorClock) match(count int) bool {
	return cc.Count == count && cc.Phase == 3
}
