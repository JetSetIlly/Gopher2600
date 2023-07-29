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

package dwarf

import (
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf/leb128"
)

type frameSectionFDE struct {
	cie          *frameSectionCIE
	startAddress uint32 // start address
	endAddress   uint32 // end address
	instructions []byte
}

func (f *frameSectionFDE) String() string {
	return fmt.Sprintf("range: %08x to %08x", f.startAddress, f.endAddress)
}

type frameSectionCIE struct {
	version byte

	// augmentation not stored

	codeAlignment    uint64 // unsigned leb128
	dataAlignment    int64  // signed leb128
	returnAddressReg uint64 // unsigned leb128

	instructions []byte
}

func (c *frameSectionCIE) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("version: %d; ", c.version))
	s.WriteString(fmt.Sprintf("code alignment: %d; ", c.codeAlignment))
	s.WriteString(fmt.Sprintf("data alignment: %d; ", c.dataAlignment))
	s.WriteString(fmt.Sprintf("ret addr reg: %0d; ", c.returnAddressReg))
	s.WriteString(fmt.Sprintf("instructions : % 02x", c.instructions))
	return s.String()
}

// the coprocessor interface required by the frame section
type frameCoproc interface {
	CoProcRegister(n int) (uint32, bool)
}

// information about the structure of call frame information can be found in
// the "DWARF-4 Specification" in section 6.4
type frameSection struct {
	coproc frameCoproc
	cie    map[uint32]*frameSectionCIE
	fde    []*frameSectionFDE

	// reading data should be done with the byteOrder functions
	byteOrder binary.ByteOrder

	// the derivation for the framebase is written to the io.Writer
	derivation io.Writer
}

// controls whether frame section should be relocated towards the actual executable origin
type frameSectionRelocate struct {
	relocate bool
	origin   uint32
}

func newFrameSectionFromFile(ef *elf.File, coproc frameCoproc,
	rel frameSectionRelocate) (*frameSection, error) {

	sec := ef.Section(".debug_frame")
	if sec == nil {
		return nil, nil
	}
	data, err := sec.Data()
	if err != nil {
		return nil, err
	}
	return newFrameSection(data, ef.ByteOrder, coproc, rel)
}

func newFrameSection(data []uint8, byteOrder binary.ByteOrder,
	coproc frameCoproc, rel frameSectionRelocate) (*frameSection, error) {

	frm := &frameSection{
		coproc:    coproc,
		cie:       make(map[uint32]*frameSectionCIE),
		byteOrder: byteOrder,
	}

	// index into the data
	var idx int

	// while there is data to be read
	for idx < len(data) {
		// length of next block (either a CIA or FDE)
		l := int(byteOrder.Uint32(data[idx:]))
		idx += 4

		// take a slice of the data block for further processing (it's just
		// easier to think about working with a smaller slice)
		b := data[idx : idx+l]
		idx += l

		// step through buffer according to whether the id indicates whether
		// the block is a CIE or an FDE
		id := byteOrder.Uint32(b)
		n := 4

		if id == 0xffffffff {
			// Common Information Entry (CIE)
			cie := &frameSectionCIE{}

			// version number
			cie.version = b[n]
			n++

			// Appendix F in "DWARF-4 Standard" lists the version numbers that
			// may appear the CIE block
			//
			// ironically, we're only going to support version 1 for now, which
			// corresponds with version 2 of the DWARF standard. this is
			// because GCC seems to emit .debug_frame section for DWARF-2 even
			// when the .debug_info section follows DWARF-4 rules
			//
			// supporting version 4 CIE blocks shouldn't be too difficult
			// if/when the need arises
			if cie.version != 1 {
				return nil, fmt.Errorf("cannot handle a CIE block version %d", cie.version)
			}

			// augmentation string. only support no augemntation for now
			if b[n] != 0x00 {
				return nil, fmt.Errorf("cannot handle a CIE block with an augmentation byte of %02x", b[n+1])
			}
			n++

			// the following fields are LEB128 encoded
			var m int
			cie.codeAlignment, m = leb128.DecodeULEB128(b[n:])
			n += m
			cie.dataAlignment, m = leb128.DecodeSLEB128(b[n:])
			n += m
			cie.returnAddressReg, m = leb128.DecodeULEB128(b[n:])
			n += m

			// instructions form the remainder of the CIE block
			cie.instructions = append(cie.instructions, b[n:l]...)

			// the real id of the CIE is the current offset into the
			// debug_frame section. we can calculate this with a bit of
			// subtraction
			id = uint32(idx - l - 4)

			// CIE is complete so we can add it to the CIE collection for
			// future reference
			frm.cie[id] = cie

		} else {
			// Frame Description Entry (FDE)
			fde := &frameSectionFDE{}

			// FDEs all refer to a CIE. we should have already found this
			cie, ok := frm.cie[id]
			if !ok {
				return nil, fmt.Errorf("FDE refers to a CIE that doesn't seem to exist")
			}
			fde.cie = cie

			// start address (named "initial location" in the DWARF-4
			// specification) is the lower instruction address for which this
			// FDE applies
			fde.startAddress = byteOrder.Uint32(b[n:])
			n += 4

			// adjust start address to bring it into range of the executable
			if rel.relocate {
				fde.startAddress += uint32(int(rel.origin) - int(fde.startAddress))
			}

			// end address (named "address range" in the DWARF-4 specification)
			// is the highest instruction address for which this FDE applies
			fde.endAddress = byteOrder.Uint32(b[n:])
			fde.endAddress += fde.startAddress
			n += 4

			// instructions form the remainder of the CIE block
			fde.instructions = append(fde.instructions, b[n:l]...)

			// FDE is complete so we can add it to the FDE collection for
			// future reference
			frm.fde = append(frm.fde, fde)
		}
	}

	return frm, nil
}

// sentinal error returned by framebase()
var noFDE = errors.New("no FDE")

// coproc implements the loclistFramebase interface
func (fr *frameSection) framebase() (uint64, error) {
	// TODO: replace magic number with a PC mnemonic. the mnemonic can then
	// refer to appropriate register for the coprocessor. the value of 15 is
	// fine for the ARM coprocessor
	addr, ok := fr.coproc.CoProcRegister(15)
	if !ok {
		return 0, fmt.Errorf("cannot retrieve value from PC of coprocessor")
	}

	return fr.framebaseForAddr(addr)
}

func (fr *frameSection) framebaseForAddr(addr uint32) (uint64, error) {
	var tab frameTable
	tab.remember()

	var fde *frameSectionFDE
	for _, f := range fr.fde {
		if addr >= f.startAddress && addr <= f.endAddress {
			tab.location = f.startAddress
			fde = f
		}
	}
	if fde == nil {
		return 0, fmt.Errorf("%w for address %08x", noFDE, addr)
	}
	if fde.cie == nil {
		return 0, fmt.Errorf("no parent CIE for FDE at address %08x", addr)
	}

	if fr.derivation != nil {
		fr.derivation.Write([]byte(fmt.Sprintf("looking for address %08x\n", addr)))
	}

	ptr := 0
	for ptr < len(fde.cie.instructions) {
		r, err := decodeFrameInstruction(fr.coproc, fr.byteOrder, fde.cie, fde.cie.instructions[ptr:], &tab)
		if err != nil {
			if errors.Is(err, frameInstructionNotImplemented) {
				return 0, fmt.Errorf("%s %w", r.opcode, err)
			}
			return 0, err
		}
		ptr += r.length

		if fr.derivation != nil {
			fr.derivation.Write([]byte(fmt.Sprintf("CIE %v\n", r)))
		}
	}

	if fr.derivation != nil {
		fr.derivation.Write([]byte(fmt.Sprintf("trying FDE Block: %v\n", fde)))
	}

	ptr = 0
	for ptr < len(fde.instructions) {
		r, err := decodeFrameInstruction(fr.coproc, fr.byteOrder, fde.cie, fde.instructions[ptr:], &tab)
		if err != nil {
			if errors.Is(err, frameInstructionNotImplemented) {
				return 0, fmt.Errorf("%s %w", r.opcode, err)
			}
			return 0, err
		}
		ptr += r.length

		if fr.derivation != nil {
			fr.derivation.Write([]byte(fmt.Sprintf("FDE %v [%08x]\n", r, tab.location)))
		}

		// we've found the row of the call frame table we need
		if tab.location >= addr {
			break
		}
	}

	framebase, ok := fr.coproc.CoProcRegister(tab.rows[0].cfaRegister)
	if !ok {
		return 0, fmt.Errorf("error retreiving framebase from register %d", tab.rows[0].cfaRegister)
	}
	framebase = uint32(int64(framebase) + tab.rows[0].registers[tab.rows[0].cfaRegister].value)

	return uint64(framebase), nil
}
