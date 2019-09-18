package setup

import (
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
)

const setupDBFile = ".gopher2600/setupDB"

func initDBSession(db *database.Session) error {
	return nil
}

// AttachCartridge to the VCS and apply setup information from the setupDB
func AttachCartridge(vcs *hardware.VCS, filename string) error {
	err := vcs.AttachCartridge(filename)
	if err != nil {
		return err
	}

	db, err := database.StartSession(setupDBFile, database.ActivityReading, initDBSession)
	if err != nil {
		if errors.Is(err, errors.DatabaseFileUnavailable) {
			// silently ignore absence of setup database
			return nil
		}
		return errors.New(errors.SetupError, err)
	}
	defer db.EndSession(false)

	return nil
}
