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

package banking

import (
	"fmt"
	"strconv"
	"strings"
)

// IsAutoSelection returns true if bank specifier indicates that the selection should be
// automatic and decided by the mapper
func IsAutoSelection(bank string) bool {
	return strings.TrimSpace(strings.ToUpper(bank)) == "AUTO"
}

// Selection specifies a bank with enough information such that it can be used by a cartridge
// implementation for bank-switching. It is up to the implementation to decide whether the selection
// is valid
type Selection struct {
	Number int
	IsRAM  bool
}

func (b Selection) String() string {
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}

// SingleSelection converts a bank specifier from a string to a new instance of the Bank type
func SingleSelection(bank string) (Selection, error) {
	bank = strings.TrimSpace(strings.ToUpper(bank))

	var isRAM bool

	b, err := strconv.Atoi(bank)
	if err != nil {
		if bank, ok := strings.CutSuffix(bank, "R"); ok {
			b, err = strconv.Atoi(bank)
			isRAM = true
		}
		if err != nil {
			return Selection{}, fmt.Errorf("startup bank not a valid value: %s", bank)
		}
	}

	if b < 0 {
		return Selection{}, fmt.Errorf("startup bank not a valid value: %s", bank)
	}

	return Selection{
		Number: b,
		IsRAM:  isRAM,
	}, nil
}

// SegmentedSelection converts a bank specifier for a segmented scheme
func SegmentedSelection(bank string) ([]Selection, error) {
	var segments []Selection

	for s := range strings.SplitSeq(bank, ":") {
		b, err := SingleSelection(s)
		if err != nil {
			return []Selection{}, err
		}
		segments = append(segments, b)
	}

	return segments, nil
}
