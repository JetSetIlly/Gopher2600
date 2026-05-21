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

package supercharger

import (
	"fmt"
	"io"
)

type fastLoadBlock struct {
	data []byte

	// remainder of block is the "header"

	// PC address to jump to once loading has finished
	startAddressLo uint8
	startAddressHi uint8

	// RAM config to be set after tape load
	configByte uint8

	// number of pages to load
	numPages uint8

	// checksum of fields in header (excluding pageTable and pageChecksums)
	headerChecksum uint8

	// we'll use this to check if the correct multiload is being read
	multiload uint8

	// not using progress speed in any meaningul way
	progressSpeed uint16

	// data is loaded according to page table
	pageTable [fastLoadPageCount]byte

	// pageChecksums of the pages in the data
	pageChecksums [fastLoadPageCount]byte
}

// from 'Stolberg': "checksum (the sum over all 8 game header bytes must be $55)"
const fastloadChecksumBase = 0x55

func (b *fastLoadBlock) setChecksums() {
	b.headerChecksum = fastloadChecksumBase
	b.headerChecksum -= b.startAddressLo + b.startAddressHi +
		b.configByte + b.numPages + b.multiload +
		uint8(b.progressSpeed) + uint8(b.progressSpeed>>8)

	for c, p := range b.pageChecksums {
		p = fastloadChecksumBase
		for _, d := range b.data[c*fastLoadPageLen : (c+1)*fastLoadHeaderLen] {
			p -= d
		}
		p -= b.pageTable[c]
		b.pageChecksums[c] = p
	}

	if !b.verifyChecksum() {
		panic("error in supercharger/fastload checksums. this is a programming error in the setChecksum() function")
	}
}

func (b *fastLoadBlock) verifyChecksum() bool {
	var verified bool

	headerChecksum := b.headerChecksum + b.startAddressLo + b.startAddressHi +
		b.configByte + b.numPages + b.multiload +
		uint8(b.progressSpeed) + uint8(b.progressSpeed>>8)

	verified = headerChecksum == fastloadChecksumBase

	for c, p := range b.pageChecksums {
		for _, d := range b.data[c*fastLoadPageLen : (c+1)*fastLoadHeaderLen] {
			p += d
		}
		p += b.pageTable[c]
		verified = verified && p == fastloadChecksumBase
	}

	return verified
}

func (b *fastLoadBlock) romdump(w io.Writer) error {
	n, err := w.Write(b.data)
	if err != nil {
		return err
	}
	if n != len(b.data) {
		return fmt.Errorf("data block is incomplete")
	}

	h := make([]byte, fastLoadHeaderLen)

	h[0] = b.startAddressLo
	h[1] = b.startAddressHi
	h[2] = b.configByte
	h[3] = b.numPages
	h[4] = b.headerChecksum
	h[5] = b.multiload
	h[6] = byte(b.progressSpeed)
	h[7] = byte(b.progressSpeed >> 8)
	copy(h[fastLoadPageTableOffset:fastLoadPageTableOffset+len(b.pageTable)], b.pageTable[:])
	copy(h[fastLoadPageChecksumTableOffset:fastLoadPageChecksumTableOffset+len(b.pageChecksums)], b.pageChecksums[:])

	n, err = w.Write(h)
	if err != nil {
		return err
	}
	if n != len(h) {
		return fmt.Errorf("block header is incomplete")
	}

	return nil
}
