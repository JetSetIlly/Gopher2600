package setup

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
	"strconv"
)

const panelSetupID = "panel"

const (
	panelSetupFieldCartHash int = iota
	panelSetupFieldP0
	panelSetupFieldP1
	panelSetupFieldCol
	panelSetupFieldNotes
	numPanelSetupFields
)

// PanelSetup is used to adjust the VCS's front panel
type PanelSetup struct {
	key      database.Key
	cartHash string

	p0  bool
	p1  bool
	col bool

	notes string
}

func deserialisePanelSetupEntry(key database.Key, fields []string) (database.Entry, error) {
	set := &PanelSetup{key: key}

	// basic sanity check
	if len(fields) > numPanelSetupFields {
		return nil, errors.New(errors.SetupPanelError, "too many fields in panel entry")
	}
	if len(fields) < numPanelSetupFields {
		return nil, errors.New(errors.SetupPanelError, "too few fields in panel entry")
	}

	var err error

	set.cartHash = fields[panelSetupFieldCartHash]

	if set.p0, err = strconv.ParseBool(fields[panelSetupFieldP0]); err != nil {
		return nil, errors.New(errors.DatabaseError, err)
	}

	if set.p1, err = strconv.ParseBool(fields[panelSetupFieldP1]); err != nil {
		return nil, errors.New(errors.DatabaseError, err)
	}

	if set.col, err = strconv.ParseBool(fields[panelSetupFieldCol]); err != nil {
		return nil, errors.New(errors.DatabaseError, err)
	}

	set.notes = fields[panelSetupFieldNotes]

	return set, nil
}

// GetID implements the database.Entry interface
func (set PanelSetup) GetID() string {
	return panelSetupID
}

// SetKey implements the database.Entry interface
func (set *PanelSetup) SetKey(key database.Key) {
	set.key = key
}

// GetKey implements the database.Entry interface
func (set PanelSetup) GetKey() database.Key {
	return set.key
}

// Serialise implements the database.Entry interface
func (set *PanelSetup) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			strconv.FormatBool(set.p0),
			strconv.FormatBool(set.p1),
			strconv.FormatBool(set.col),
			set.notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (set PanelSetup) CleanUp() {
	// no cleanup necessary
}

func (set PanelSetup) String() string {
	return fmt.Sprintf("%s, p0=%v, p1=%v, col=%v\n", set.cartHash, set.p0, set.p1, set.col)
}

// matchCartHash implements setupEntry interface
func (set PanelSetup) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface
func (set PanelSetup) apply(vcs *hardware.VCS) error {
	if set.p0 {
		if err := vcs.Panel.Handle(peripherals.PanelSetPlayer0Pro); err != nil {
			return err
		}
	} else {
		if err := vcs.Panel.Handle(peripherals.PanelSetPlayer0Am); err != nil {
			return err
		}
	}

	if set.p1 {
		if err := vcs.Panel.Handle(peripherals.PanelSetPlayer1Pro); err != nil {
			return err
		}
	} else {
		if err := vcs.Panel.Handle(peripherals.PanelSetPlayer1Am); err != nil {
			return err
		}
	}

	if set.col {
		if err := vcs.Panel.Handle(peripherals.PanelSetColor); err != nil {
			return err
		}
	} else {
		if err := vcs.Panel.Handle(peripherals.PanelSetBlackAndWhite); err != nil {
			return err
		}
	}

	return nil
}
