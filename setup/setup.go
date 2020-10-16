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

package setup

import (
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/paths"
)

// the location of the setupDB file.
const setupDBFile = "setupDB"

// setupEntry is the generic entry type in the setupDB.
type setupEntry interface {
	database.Entry

	// match the hash stored in the database with the user supplied hash
	matchCartHash(hash string) bool

	// apply changes indicated in the entry to the VCS
	apply(vcs *hardware.VCS) error
}

// when starting a database session we must add the details of the entry types
// that will be found in the database.
func initDBSession(db *database.Session) error {
	// add entry types
	if err := db.RegisterEntryType(panelSetupID, deserialisePanelSetupEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(patchID, deserialisePatchEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(televisionID, deserialiseTelevisionEntry); err != nil {
		return err
	}

	return nil
}

// AttachCartridge to the VCS and apply setup information from the setupDB.
// This function should be preferred to the hardware.VCS.AttachCartridge()
// function in almost all cases.
func AttachCartridge(vcs *hardware.VCS, cartload cartridgeloader.Loader) error {
	err := vcs.AttachCartridge(cartload)
	if err != nil {
		return err
	}

	dbPth, err := paths.ResourcePath("", setupDBFile)
	if err != nil {
		return curated.Errorf("setup: %v", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityReading, initDBSession)
	if err != nil {
		if curated.Is(err, database.NotAvailable) {
			// silently ignore absence of setup database
			return nil
		}
		return curated.Errorf("setup: %v", err)
	}
	defer db.EndSession(false)

	onSelect := func(ent database.Entry) error {
		// database entry should also satisfy setupEntry interface
		set, ok := ent.(setupEntry)
		if !ok {
			return curated.Errorf("setup: attach cartridge: database entry does not satisfy setupEntry interface")
		}

		if set.matchCartHash(vcs.Mem.Cart.Hash) {
			err := set.apply(vcs)
			if err != nil {
				return err
			}
		}

		return nil
	}

	_, err = db.SelectAll(onSelect)
	if err != nil {
		return curated.Errorf("setup: %v", err)
	}

	return nil
}
