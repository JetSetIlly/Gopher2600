package colorclock

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

// ColorClock is how the VCS keeps track of horizontal positioning
type ColorClock struct {
	polycounter.Polycounter
}

// New is the preferred method of initialisation for the ColorClock
func New() *ColorClock {
	cc := new(ColorClock)
	if cc == nil {
		return nil
	}
	cc.SetResetPattern("010100")
	return cc
}

// StringTerse returns the color clock information in terse format
func (cc ColorClock) StringTerse() string {
	return fmt.Sprintf("CC=%s", cc.Polycounter.StringTerse())
}

// String returns the color clock information in verbose format
func (cc ColorClock) String() string {
	// print polycount and VCS "pixel" equivalent
	return fmt.Sprintf("CCLOCK: %v [%dpx]\n", cc.Polycounter, cc.Pixel())
}

// Pixel returns the color clock when expressed a pixel
func (cc ColorClock) Pixel() int {
	return (cc.Count * 4) + cc.Phase - 68
}
