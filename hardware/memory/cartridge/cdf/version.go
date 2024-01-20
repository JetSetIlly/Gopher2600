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
	"bytes"
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// versions contains the information that can differ between CDF versions.
type version struct {
	mmap architecture.Map

	// mappingID depends on the version
	submapping string

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

	// segment origins
	driverROMOrigin    uint32
	customROMOrigin    uint32
	driverRAMOrigin    uint32
	dataRAMOrigin      uint32
	variablesRAMOrigin uint32

	// the memtop values in the version struct are the absolute maximum size
	// supported by the format. the actual memtop may be different depending on
	// the cartridge. the real memtop for a segment should not exceed these
	// maximum values
	driverROMMemtop    uint32
	customROMMemtop    uint32
	driverRAMMemtop    uint32
	dataRAMMemtop      uint32
	variablesRAMMemtop uint32

	// entry point into ARM program
	entrySP uint32
	entryLR uint32
	entryPC uint32

	// fast fetch modes. these are always disabled except for some versions of CDFJ+
	fastLDX bool
	fastLDY bool

	// offset by which to adjust the datastream fetcher. this is always zero
	// except for some versions of CDFJ+
	datastreamOffset uint8
}

func newVersion(memModel string, v string, data []uint8) (version, error) {
	var mmap architecture.Map

	switch memModel {
	case "AUTO":
		mmap = architecture.NewMap(architecture.Harmony)

	case "LPC2000":
		// older preference value. deprecated.
		fallthrough
	case "ARM7TDMI":
		// old value used to indicate ARM7TDMI architecture. easiest to support
		// it here in this manner
		mmap = architecture.NewMap(architecture.Harmony)

	case "STM32F407VGT6":
		// older preference value. deprecated.
		fallthrough
	case "ARMv7_M":
		// old value used to indicate ARM7TDMI architecture. easiest to support
		// it here in this manner
		mmap = architecture.NewMap(architecture.PlusCart)
	}

	ver := version{
		mmap: mmap,

		driverROMOrigin: mmap.FlashOrigin,
		driverROMMemtop: mmap.FlashOrigin | 0x000007ff, // 2k
		customROMOrigin: mmap.FlashOrigin | 0x00000800,
		customROMMemtop: mmap.FlashMemtop,
		driverRAMOrigin: mmap.SRAMOrigin,
		driverRAMMemtop: mmap.SRAMOrigin | 0x000007ff, // 2k
		dataRAMOrigin:   mmap.SRAMOrigin | 0x00000800,

		// data RAM memtop is different for CDFJ+
		dataRAMMemtop: mmap.SRAMOrigin | 0x000017ff,

		// there is no variables segment in CDFJ+ so these value are reset in
		// that instance
		variablesRAMOrigin: mmap.SRAMOrigin | 0x00001800,
		variablesRAMMemtop: mmap.SRAMOrigin | 0x00001fff,
	}

	// entry point into ARM program (different for CDFJ+)
	ver.entrySP = mmap.SRAMOrigin | 0x00001fdc
	ver.entryLR = ver.customROMOrigin
	ver.entryPC = ver.entryLR + 8

	// different version of the CDF mapper have different addresses
	switch v {
	case "CDFJ+":
		// data RAM memtop is different for CDFJ+
		ver.dataRAMMemtop = mmap.SRAMOrigin | 0x00007fff

		// variables segment concept not used in CDFJ+
		ver.variablesRAMOrigin = 0x0
		ver.variablesRAMMemtop = 0x0

		ver.submapping = "CDFJ+"
		ver.fetcherBase = 0x0098
		ver.incrementBase = 0x0124
		ver.musicBase = 0x01b0
		ver.fastJMPmask = 0xfe
		ver.amplitudeRegister = 35
		ver.fetcherShift = 16
		ver.incrementShift = 8
		ver.musicFetcherShift = 12
		ver.fetcherMask = 0xff000000

		// CDFJ+ sets the MAMCR to full in the driver
		ver.mmap.PreferredMAMCR = architecture.MAMfull

		// entry point different for CDFJ+
		idx := 0x17f8
		ver.entryLR = uint32(data[idx])
		ver.entryLR |= uint32(data[idx+1]) << 8
		ver.entryLR |= uint32(data[idx+2]) << 16
		ver.entryLR |= uint32(data[idx+3]) << 24
		ver.entryLR &= 0xfffffffe
		ver.entryPC = ver.entryLR

		// stack top differnt for CDFJ+
		idx = 0x17f4
		ver.entrySP = uint32(data[idx])
		ver.entrySP |= uint32(data[idx+1]) << 8
		ver.entrySP |= uint32(data[idx+2]) << 16
		ver.entrySP |= uint32(data[idx+3]) << 24

		// CDFJ+ additional differences

		// detect fastfetch mode by searching for bytes in the CDFJ driver
		ver.fastLDX = bytes.Contains(data[:2048], []byte{ldxImmediate, 0x00, 0x52, 0x13})
		ver.fastLDY = bytes.Contains(data[:2048], []byte{ldyImmediate, 0x00, 0x52, 0x13})

		// bytes.Contains(data[:2048], []byte{ldaImmediate, 0x00, 0x52, 0x13}) {

		offset := bytes.Index(data[:2048], []byte{0x20, 0x42, 0xe2})
		if offset > 1 {
			ver.datastreamOffset = data[offset-1]
		} else {
			ver.datastreamOffset = 0
		}

	case "CDFJ":
		ver.submapping = "CDFJ"
		ver.fetcherBase = 0x0098
		ver.incrementBase = 0x0124
		ver.musicBase = 0x01b0
		ver.fastJMPmask = 0xfe
		ver.amplitudeRegister = 35
		ver.fetcherShift = 20
		ver.incrementShift = 12
		ver.musicFetcherShift = 20
		ver.fetcherMask = 0xf0000000

	case "CDF0":
		ver.submapping = "CDF0"
		ver.fetcherBase = 0x06e0
		ver.incrementBase = 0x0768
		ver.musicBase = 0x07f0
		ver.fastJMPmask = 0xff
		ver.amplitudeRegister = 34
		ver.fetcherShift = 20
		ver.incrementShift = 12
		ver.musicFetcherShift = 20
		ver.fetcherMask = 0xf0000000

	case "CDF1":
		ver.submapping = "CDF1"
		ver.fetcherBase = 0x00a0
		ver.incrementBase = 0x0128
		ver.musicBase = 0x01b0
		ver.fastJMPmask = 0xff
		ver.amplitudeRegister = 34
		ver.fetcherShift = 20
		ver.incrementShift = 12
		ver.musicFetcherShift = 20
		ver.fetcherMask = 0xf0000000

	default:
		return version{}, fmt.Errorf("unknown version: %s", v)
	}

	return ver, nil
}
