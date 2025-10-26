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
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

// *** NOTE ***
// remove DisablePeriphFingerprint field from VCS type once setup package has
// been updated to handle peripherals/controllers correctly

// the location of the setupDB file.
const setupDBFile = "setupDB"

// setupEntry is the generic entry type in the setupDB.
type setupEntry interface {
	database.Entry

	// match the hash stored in the database with the user supplied hash
	matchCartHash(hash string) bool

	// apply changes indicated in the entry to the VCS. returned string value
	// will be used for log message
	apply(vcs *hardware.VCS) (string, error)
}

// when starting a database session we must add the details of the entry types
// that will be found in the database.
func initDBSession(db *database.Session) error {
	// add entry types
	if err := db.RegisterEntryType(panelSetupEntryType, deserialisePanelSetupEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(patchEntryType, deserialisePatchEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(televisionEntryType, deserialiseTelevisionEntry); err != nil {
		return err
	}

	return nil
}

// AttachCartridge to the VCS and apply setup information from the setupDB.
// This function should be preferred to the hardware.VCS.AttachCartridge()
// function in almost all cases.
func AttachCartridge(vcs *hardware.VCS, cartload cartridgeloader.Loader, hook func()) error {
	err := vcs.AttachCartridge(cartload, hook)
	if err != nil {
		// not adding the "setup" prefix for this. the setup package has not
		// added any value yet and it would just be noise
		return err
	}

	dbPth, err := resources.JoinPath(setupDBFile)
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityReading, initDBSession)
	if err != nil {
		if errors.Is(err, database.NotAvailable) {
			// silently ignore absence of setup database
			return nil
		}
		return fmt.Errorf("setup: %w", err)
	}
	defer db.EndSession(false)

	onSelect := func(ent database.Entry) error {
		// database entry should also satisfy setupEntry interface
		set, ok := ent.(setupEntry)
		if !ok {
			return fmt.Errorf("setup: attach cartridge: database entry does not satisfy setupEntry interface")
		}

		if set.matchCartHash(vcs.Mem.Cart.Hash) {
			msg, err := set.apply(vcs)
			if err != nil {
				return err
			}
			logger.Log(logger.Allow, "setup", msg)
		}

		return nil
	}

	_, err = db.SelectAll(onSelect)
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	return nil
}
