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

package mapper

import (
	"fmt"
	"strconv"
	"strings"
)

// BankContent contains data and ID of a cartridge bank. Used by CopyBanks()
// and helps the disassembly process.
type BankContent struct {
	// bank number
	Number int

	// copy of the bank data
	Data []uint8

	// the segment origins that this data is allowed to be mapped to. most
	// cartridges will have one entry. values in the array will refer to
	// addresses in the cartridge address space. by convention the mappers will
	// refer to the primary mirror.
	//
	//	memorymap.OriginCart <= origins[n] <= memorymap.MemtopCart
	//
	// to index the Data field, transform the origin and any address derived
	// from it, with memorymap.CartridgeBits
	//
	//	idx := Origins[0] & memorymap.CartridgeBits
	//	v := Data[idx]
	//
	// address values are supplied by the mapper implementation and must be
	// cartridge addresses and should in the primary cartridge mirror range
	// (ie. 0x1000 to 0x1fff)j
	Origins []uint16
}

// BankInfo is used to identify a cartridge bank. In some instance a bank can
// be identified by it's bank number only. In other contexts more detail is
// required and so BankInfo is used isntead.
type BankInfo struct {
	// bank number
	Number int

	// name of bank. used for special purpose banks (eg. supercharger BIOS).
	// should be empty if bank has no name.
	Name string

	// is cartridge memory segmented and if so which segment is this bank
	// mapped to
	IsSegmented bool
	Segment     int

	// is cartridge bank writable
	IsRAM bool

	// if the address used to generate the Details is not a cartridge address.
	// this happens deliberately for example, during the Supercharger load
	// procedure, where execution happens (briefly) inside the main VCS RAM
	NonCart bool

	// the cartridge is currently feeding NOP bytes onto the data bus and
	// therefore the data from this bank should not be considered predictable.
	//
	// this flag has been added to support the ARM coprocessor found in
	// conjunction with CDF* and DPC+ mappers. future coprocessors may work
	// differently.
	ExecutingCoprocessor bool

	// if ExecutingCoprocessor is valid then we also record the address the
	// processor will resume from.
	CoprocessorResumeAddr uint16
}

// very basic String representation of BankInfo.
func (b BankInfo) String() string {
	if b.ExecutingCoprocessor {
		return "*"
	}
	if b.NonCart {
		return "-"
	}
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}

// IsAutoBankSelection returns true if bank specifier indicates that the
// selection should be automatic
func IsAutoBankSelection(bank string) bool {
	return strings.TrimSpace(strings.ToUpper(bank)) == "AUTO"
}

// BankSelection specifies a bank with enough information such that it can be
// used by a cartridge implementation for bank-switching. It is up to the
// implementation to decide whether the selection is valid
type BankSelection struct {
	Number int
	IsRAM  bool
}

func (b BankSelection) String() string {
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}

// SingleBankSelection converts a bank specifier from a string to a new instance
// of the Bank type
func SingleBankSelection(bank string) (BankSelection, error) {
	bank = strings.TrimSpace(strings.ToUpper(bank))

	var isRAM bool

	b, err := strconv.Atoi(bank)
	if err != nil {
		if bank, ok := strings.CutSuffix(bank, "R"); ok {
			b, err = strconv.Atoi(bank)
			isRAM = true
		}
		if err != nil {
			return BankSelection{}, fmt.Errorf("startup bank not a valid value: %s", bank)
		}
	}

	if b < 0 {
		return BankSelection{}, fmt.Errorf("startup bank not a valid value: %s", bank)
	}

	return BankSelection{
		Number: b,
		IsRAM:  isRAM,
	}, nil
}

// SegmentedBankSelection converts a bank specifier for a segmented scheme
func SegmentedBankSelection(bank string) ([]BankSelection, error) {
	var segments []BankSelection

	for _, s := range strings.Split(bank, ":") {
		b, err := SingleBankSelection(s)
		if err != nil {
			return []BankSelection{}, err
		}
		segments = append(segments, b)
	}

	return segments, nil
}
