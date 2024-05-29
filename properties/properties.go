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

package properties

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

const propertiesFile = "stella.pro"

type Entry struct {
	Hash         string
	Name         string
	Manufacturer string
	Note         string
	Rarity       string
	Model        string
}

func (e Entry) IsValid() bool {
	return len(e.Hash) > 0
}

type Properties struct {
	available bool
	entries   map[string]Entry
}

func Load() (Properties, error) {
	pro := Properties{
		entries: make(map[string]Entry),
	}

	path, err := resources.JoinPath(propertiesFile)
	if err != nil {
		return Properties{}, fmt.Errorf("properties: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return Properties{}, fmt.Errorf("properties: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	var entry Entry
	var line int
	var rejected int

	for scanner.Scan() {
		line++

		flds := strings.SplitN(scanner.Text(), " ", 2)
		if len(flds) < 2 {
			continue // for loop
		}

		flds[0] = strings.Trim(flds[0], "\"")
		flds[1] = strings.Trim(flds[1], "\"")
		switch strings.ToUpper(flds[0]) {
		case "CART.MD5":
			// create new entry
			if len(entry.Hash) == 32 {
				pro.entries[entry.Hash] = entry
			}

			// prepare new entry if has is valid
			if len(flds[1]) != 32 {
				logger.Logf(logger.Allow, "properties", "invalid hash entry at line %d", line)
				rejected++
			} else {
				entry = Entry{
					Hash: flds[1],
				}
			}

		case "CART.NAME":
			entry.Name = flds[1]

		case "CART.MANUFACTURER":
			entry.Manufacturer = flds[1]

		case "CART.NOTE":
			entry.Note = flds[1]

		case "CART.RARITY":
			entry.Rarity = flds[1]

		case "CART.MODEL":
			entry.Model = flds[1]
		}
	}

	if err = scanner.Err(); err != nil {
		return Properties{}, fmt.Errorf("pro: %w", err)
	}

	logger.Logf(logger.Allow, "properties", "%d entries loaded", len(pro.entries))
	if rejected > 0 {
		logger.Logf(logger.Allow, "properties", "%d entries rejected", rejected)
	}

	return pro, nil
}

// Find the property entry for the ROM with the supplied md5 hash
func (pro Properties) Lookup(md5Hash string) Entry {
	if e, ok := pro.entries[md5Hash]; ok {
		return e
	}

	return Entry{}
}
