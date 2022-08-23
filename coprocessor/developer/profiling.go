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
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// Profiling implements the CartCoProcDeveloper interface.
func (dev *Developer) Profiling() map[uint32]float32 {
	if dev.disabled {
		return nil
	}

	return dev.profiledAddresses
}

// process cycle counts for (non-zero) addresses in dev.profiledAddresses
func (dev *Developer) profileProcess(frameInfo television.FrameInfo) {
	if dev.disabled || dev.source == nil {
		return
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	newCoords := dev.tv.GetCoords()
	newKernel := KernelUnstable

	// because the execution profile is only processed once per scanline, it's
	// important that the profile is flushed at boundary points between
	// kernels. without this check there is a danger that execution will be
	// associated with the wrong kernel
	boundary := false

	// checking to see if kernel has changed
	if frameInfo.Stable {
		if newCoords.Scanline <= frameInfo.VisibleTop {
			newKernel = KernelVBLANK
			boundary = newCoords.Scanline == frameInfo.VisibleTop
		} else if newCoords.Scanline >= frameInfo.VisibleBottom {
			newKernel = KernelOverscan
			boundary = newCoords.Scanline == frameInfo.VisibleBottom
		} else {
			newKernel = KernelScreen
		}
	}

	// process statistics if the kernel has changed or scanline is at a kernel boundary (ie. the kernel is about to change)
	if newKernel != dev.profilingKernel || boundary {
		dev.profilingKernel = newKernel

		for pc, ct := range dev.profiledAddresses {
			if ct > 0 {
				dev.source.executionProfile(pc, ct, dev.profilingKernel)
				dev.profiledAddresses[pc] = 0
			}
		}

		// ignoring execution during the setup "kernel"
		if dev.profilingKernel != KernelUnstable {
			diff := coords.Diff(newCoords, dev.profilingCoords, frameInfo.TotalScanlines)
			clocks := coords.Sum(diff, frameInfo.TotalScanlines)

			dev.frameStatsLock.Lock()
			dev.frameStats.accumulate(clocks, dev.profilingKernel)
			dev.frameStatsLock.Unlock()
		}
	}

	// note coords for next scanline
	dev.profilingCoords = newCoords
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
