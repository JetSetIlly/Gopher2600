// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package developer

import (
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// Profiling implements the CartCoProcDeveloper interface.
func (dev *Developer) Profiling() map[uint32]float32 {
	return dev.profiledAddresses
}

// process cycle counts for (non-zero) addresses in dev.profiledAddresses
func (dev *Developer) profileProcess(frameInfo television.FrameInfo) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	accumulate := func(k KernelVCS) {
		for pc, ct := range dev.profiledAddresses {
			if ct > 0 {
				dev.source.executionProfile(pc, ct, k)
				dev.profiledAddresses[pc] = 0
			}
		}
	}

	// checking to see if kernel has changed
	if frameInfo.Stable {
		c := dev.tv.GetCoords()

		if c.Scanline == frameInfo.VisibleTop-1 {
			accumulate(KernelVBLANK)
		} else if c.Scanline == frameInfo.VisibleBottom {
			accumulate(KernelScreen)
		} else if c.Scanline >= frameInfo.TotalScanlines {
			accumulate(KernelOverscan)
		}
	} else {
		accumulate(KernelUnstable)
	}
}

// KernelVCS indicates the 2600 kernel that is associated with a source function
// or source line.
type KernelVCS int

// List of KernelVCS values.
const (
	KernelAny      KernelVCS = 0x00
	KernelScreen   KernelVCS = 0x01
	KernelVBLANK   KernelVCS = 0x02
	KernelOverscan KernelVCS = 0x04

	// code that is run while the television is in an unstable state
	KernelUnstable KernelVCS = 0x08
)

func (k KernelVCS) String() string {
	switch k {
	case KernelScreen:
		return "Screen"
	case KernelVBLANK:
		return "VBLANK"
	case KernelOverscan:
		return "Overscan"
	case KernelUnstable:
		return "ROM Setup"
	}

	return "Any"
}

// List of KernelVCS values as strings
var AvailableInKernelOptions = []string{"Any", "VBLANK", "Screen", "Overscan", "ROM Setup"}
