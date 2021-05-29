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

package cdf

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"
)

// versions contains the information that can differ between CDF versions.
type version struct {
	// mappingID and description differ depending on the version
	submapping  string
	description string

	// the base index for the CDF registers. These values are indexes into the
	// data RAM.
	fetcherBase   uint32
	incrementBase uint32
	musicBase     uint32

	// which data fetcher is the amplitude fetcher differs by CFD version
	amplitudeRegister int

	// the significant bits of the most-significant byte of a fastjmp operand
	// must be masked appropriately. in the case of CDFJ a fast jump can be
	// triggered with either "4c 00 00" or "4c 01 00"
	fastJMPmask uint8

	// how we access the bits of the registers differ for different versions of
	// the CDF mapper
	fetcherShift      uint32
	incrementShift    uint32
	musicFetcherShift uint32

	// the DSPTR register is written one byte at a time from the 6507. How many
	// bytes are in the DSPTR depends on the size of the CDF ROM.
	fetcherMask uint32

	// addresses (driver is always in the same location)
	driverOriginROM uint32
	driverMemtopROM uint32
	driverOriginRAM uint32
	driverMemtopRAM uint32

	// addresses (different for CDFJ+)
	customOriginROM    uint32
	customMemtopROM    uint32
	dataOriginRAM      uint32
	dataMemtopRAM      uint32
	variablesOriginRAM uint32
	variablesMemtopRAM uint32

	// entry point into ARM program
	entrySR uint32
	entryLR uint32
	entryPC uint32
}

func newVersion(v string, data []uint8) (version, error) {
	r := version{
		// addresses (driver is always in the same location)
		driverOriginROM: arm7tdmi.FlashOrigin,
		driverMemtopROM: arm7tdmi.FlashOrigin | 0x000007ff, // 2k
		driverOriginRAM: arm7tdmi.SRAMOrigin,
		driverMemtopRAM: arm7tdmi.SRAMOrigin | 0x000007ff, // 2k

		// addresses (different for CDFJ+)
		customOriginROM:    arm7tdmi.FlashOrigin | 0x00000800,
		customMemtopROM:    arm7tdmi.Flash32kMemtop,
		dataOriginRAM:      arm7tdmi.SRAMOrigin | 0x00000800,
		dataMemtopRAM:      arm7tdmi.SRAMOrigin | 0x000017ff,
		variablesOriginRAM: arm7tdmi.SRAMOrigin | 0x00001800,
		variablesMemtopRAM: arm7tdmi.SRAMOrigin | 0x00001fff,
	}

	// entry point into ARM program
	r.entrySR = arm7tdmi.SRAMOrigin | 0x00001fdc
	r.entryLR = r.customOriginROM
	r.entryPC = r.entryLR + 8

	// different version of the CDF mapper have different addresses
	switch v {
	case "CDF0":
		r.submapping = "CDF0"
		r.description = "Harmony (CDF0)"
		r.fetcherBase = 0x06e0
		r.incrementBase = 0x0768
		r.musicBase = 0x07f0
		r.fastJMPmask = 0xff
		r.amplitudeRegister = 34
		r.fetcherShift = 20
		r.incrementShift = 12
		r.musicFetcherShift = 20
		r.fetcherMask = 0xf0000000

	case "CDFJ+":
		r.submapping = "CDFJ+"
		r.description = "Harmony (CDFJ+)"
		r.fetcherBase = 0x0098
		r.incrementBase = 0x0124
		r.musicBase = 0x01b0
		r.fastJMPmask = 0xfe
		r.amplitudeRegister = 35
		r.fetcherShift = 16
		r.incrementShift = 8
		r.musicFetcherShift = 12
		r.fetcherMask = 0xff000000

		idx := 0x17f8
		r.entryLR = uint32(data[idx])
		r.entryLR |= uint32(data[idx+1]) << 8
		r.entryLR |= uint32(data[idx+2]) << 16
		r.entryLR |= uint32(data[idx+3]) << 24
		r.entryLR &= 0xfffffffe
		r.entryPC = r.entryLR

		// custom oring unchange. memtop is changed
		r.customMemtopROM = arm7tdmi.Flash64kMemtop

		// data origin unchanged. memtop is changed
		r.dataMemtopRAM = arm7tdmi.SRAMOrigin | 0x00007fff

		// variables concept not used in CDFJ
		r.variablesOriginRAM = 0x0
		r.variablesMemtopRAM = 0x0

		idx = 0x17f4
		r.entrySR = uint32(data[idx])
		r.entrySR |= uint32(data[idx+1]) << 8
		r.entrySR |= uint32(data[idx+2]) << 16
		r.entrySR |= uint32(data[idx+3]) << 24

	case "CDFJ":
		r.submapping = "CDFJ"
		r.description = "Harmony (CDFJ)"
		r.fetcherBase = 0x0098
		r.incrementBase = 0x0124
		r.musicBase = 0x01b0
		r.fastJMPmask = 0xfe
		r.amplitudeRegister = 35
		r.fetcherShift = 20
		r.incrementShift = 12
		r.musicFetcherShift = 20
		r.fetcherMask = 0xf0000000

	case "CDF1":
		r.submapping = "CDF1"
		r.description = "Harmony (CDF1)"
		r.fetcherBase = 0x00a0
		r.incrementBase = 0x0128
		r.musicBase = 0x01b0
		r.fastJMPmask = 0xff
		r.amplitudeRegister = 34
		r.fetcherShift = 20
		r.incrementShift = 12
		r.musicFetcherShift = 20
		r.fetcherMask = 0xf0000000

	default:
		return version{}, curated.Errorf("unknown version: %s", v)
	}

	return r, nil
}
