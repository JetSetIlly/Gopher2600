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
	"debug/elf"
	"fmt"
)

func relocateELFSection(ef *elf.File, section string) ([]uint8, error) {
	sec := ef.Section(section)
	if sec == nil {
		return nil, fmt.Errorf("no section named %s", section)
	}

	data, err := sec.Data()
	if err != nil {
		return nil, err
	}

	rel := ef.Section(fmt.Sprintf(".rel%s", section))
	if rel == nil {
		// there is no relocation section so we can just return the data of the
		// section that has been requested
		return data, nil
	}

	relData, err := rel.Data()
	if err != nil {
		return nil, err
	}

	// symbols used during relocation
	symbols, err := ef.Symbols()
	if err != nil {
		return nil, err
	}

	// every relocation entry
	for i := 0; i < len(relData); i += 8 {
		// the relocation entry fields
		offset := ef.ByteOrder.Uint32(relData[i:])
		info := ef.ByteOrder.Uint32(relData[i+4:])

		// symbol is encoded in the info value
		symbolIdx := info >> 8
		sym := symbols[symbolIdx-1]

		// reltype is encoded in the info value
		relType := info & 0xff

		switch elf.R_ARM(relType) {
		case elf.R_ARM_TARGET1:
			fallthrough

		case elf.R_ARM_ABS32:
			addend := ef.ByteOrder.Uint32(data[offset:])
			v := uint32(sym.Value) + addend

			// commit write
			ef.ByteOrder.PutUint32(data[offset:], v)

		default:
			return nil, fmt.Errorf("unhandled ARM relocation type (%v)", relType)
		}
	}

	return data, nil
}
