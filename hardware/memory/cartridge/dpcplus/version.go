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

package dpcplus

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

type mmap struct {
	arch architecture.Map

	// segment origins
	driverROMOrigin uint32
	customROMOrigin uint32
	dataROMOrigin   uint32
	freqROMOrigin   uint32
	driverRAMOrigin uint32
	dataRAMOrigin   uint32
	freqRAMOrigin   uint32

	// the memtop values in the version struct are the absolute maximum size
	// supported by the format. the actual memtop may be different depending on
	// the cartridge. the real memtop for a segment should not exceed these
	// maximum values
	driverROMMemtop uint32
	customROMMemtop uint32
	dataROMMemtop   uint32
	freqROMMemtop   uint32
	driverRAMMemtop uint32
	dataRAMMemtop   uint32
	freqRAMMemtop   uint32

	ccmAvailable bool
	ccmOrigin    uint32
	ccmMemtop    uint32
}

func newVersion(id string) (mmap, error) {
	var arch architecture.Map

	switch id {
	case "DPC+":
		arch = architecture.NewMap(architecture.Harmony)
		return mmap{
			arch:            arch,
			driverROMOrigin: arch.Regions["Flash"].Origin,
			driverROMMemtop: arch.Regions["Flash"].Origin | 0x00000bff,
			customROMOrigin: arch.Regions["Flash"].Origin | 0x00000c00,
			customROMMemtop: arch.Regions["Flash"].Origin | 0x00006bff,
			dataROMOrigin:   arch.Regions["Flash"].Origin | 0x00006c00,
			dataROMMemtop:   arch.Regions["Flash"].Origin | 0x00007bff,
			freqROMOrigin:   arch.Regions["Flash"].Origin | 0x00007c00,
			freqROMMemtop:   arch.Regions["Flash"].Origin | 0x00008000,
			driverRAMOrigin: arch.Regions["SRAM"].Origin,
			driverRAMMemtop: arch.Regions["SRAM"].Origin | 0x00000bff,
			dataRAMOrigin:   arch.Regions["SRAM"].Origin | 0x00000c00,
			dataRAMMemtop:   arch.Regions["SRAM"].Origin | 0x00001bff,
			freqRAMOrigin:   arch.Regions["SRAM"].Origin | 0x00001c00,
			freqRAMMemtop:   arch.Regions["SRAM"].Origin | 0x00002000,
		}, nil

	case "DPCp":
		arch = architecture.NewMap(architecture.PlusCart)
		return mmap{
			arch:            arch,
			driverROMOrigin: arch.Regions["SRAM"].Origin,
			driverROMMemtop: arch.Regions["SRAM"].Origin | 0x00000bff,
			customROMOrigin: arch.Regions["SRAM"].Origin | 0x00000c00,
			customROMMemtop: arch.Regions["SRAM"].Origin | 0x00006bff,
			dataROMOrigin:   arch.Regions["SRAM"].Origin | 0x00006c00,
			dataROMMemtop:   arch.Regions["SRAM"].Origin | 0x00007bff,
			freqROMOrigin:   arch.Regions["SRAM"].Origin | 0x00007c00,
			freqROMMemtop:   arch.Regions["SRAM"].Origin | 0x00008000,
			driverRAMOrigin: arch.Regions["SRAM"].Origin | 0x00010000,
			driverRAMMemtop: arch.Regions["SRAM"].Origin | 0x00010bff,
			dataRAMOrigin:   arch.Regions["SRAM"].Origin | 0x00010c00,
			dataRAMMemtop:   arch.Regions["SRAM"].Origin | 0x00011bff,
			freqRAMOrigin:   arch.Regions["SRAM"].Origin | 0x00011c00,
			freqRAMMemtop:   arch.Regions["SRAM"].Origin | 0x00012000,

			// DPCp has CCM memory
			ccmAvailable: true,
			ccmOrigin:    arch.Regions["CCM"].Origin,
			ccmMemtop:    arch.Regions["CCM"].Origin | 0x00010000,
		}, nil
	}

	return mmap{}, fmt.Errorf("unknown DPC+ version: %s", id)
}
