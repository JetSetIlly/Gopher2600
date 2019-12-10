package setup

import (
	"gopher2600/cartridgeloader"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/paths"
)

// the location of the setupDB file
const setupDBFile = "setupDB"

// setupEntry is the generic entry type in the setupDB
type setupEntry interface {
	database.Entry

	// match the hash stored in the database with the user supplied hash
	matchCartHash(hash string) bool

	// apply changes indicated in the entry to the VCS
	apply(vcs *hardware.VCS) error
}

// when starting a database session we must add the details of the entry types
// that will be found in the database
func initDBSession(db *database.Session) error {
	if err := db.RegisterEntryType(panelSetupID, deserialisePanelSetupEntry); err != nil {
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

	db, err := database.StartSession(paths.ResourcePath(setupDBFile), database.ActivityReading, initDBSession)
	if err != nil {
		if errors.Is(err, errors.DatabaseFileUnavailable) {
			// silently ignore absence of setup database
			return nil
		}
		return errors.New(errors.SetupError, err)
	}
	defer db.EndSession(false)

	onSelect := func(ent database.Entry) (bool, error) {
		// datbase entry should also satisfy Regressor interface
		set, ok := ent.(setupEntry)
		if !ok {
			return false, errors.New(errors.PanicError, "setup.AttachCartridge()", "database entry does not satisfy setupEntry interface")
		}

		if set.matchCartHash(vcs.Mem.Cart.Hash) {
			err := set.apply(vcs)
			if err != nil {
				return false, err
			}

			// even though we've matched the cart hash we should return true to
			// indicate that we should continue the select and look for other
			// matching entries
		}

		return true, nil
	}

	_, err = db.SelectAll(onSelect)
	if err != nil {
		return errors.New(errors.SetupError, err)
	}

	return nil
}
