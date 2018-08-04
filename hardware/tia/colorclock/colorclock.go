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
	cc.SetResetPattern("010100") // count==56,
	return cc
}

// MachineInfoTerse returns the color clock information in terse format
func (cc ColorClock) MachineInfoTerse() string {
	return fmt.Sprintf("CC=%s", cc.Polycounter.MachineInfoTerse())
}

// MachineInfo returns the color clock information in verbose format
func (cc ColorClock) MachineInfo() string {
	// print polycount and VCS "pixel" equivalent
	return fmt.Sprintf("CCLOCK: %v [%dpx]", cc.Polycounter, cc.Pixel())
}

// map String to MachineInfo
func (cc ColorClock) String() string {
	return cc.MachineInfo()
}

// Pixel returns the color clock when expressed a pixel
func (cc ColorClock) Pixel() int {
	return (cc.Count * 4) + cc.Phase - 68
}
