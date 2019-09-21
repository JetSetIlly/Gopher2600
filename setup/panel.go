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
	cartHash string

	p0  bool
	p1  bool
	col bool

	notes string
}

func deserialisePanelSetupEntry(fields []string) (database.Entry, error) {
	set := &PanelSetup{}

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
		return nil, errors.New(errors.SetupPanelError, "invalid player 0 setting")
	}

	if set.p1, err = strconv.ParseBool(fields[panelSetupFieldP1]); err != nil {
		return nil, errors.New(errors.SetupPanelError, "invalid player 1 setting")
	}

	if set.col, err = strconv.ParseBool(fields[panelSetupFieldCol]); err != nil {
		return nil, errors.New(errors.SetupPanelError, "invalid color setting")
	}

	set.notes = fields[panelSetupFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface
func (set PanelSetup) ID() string {
	return panelSetupID
}

// String implements the database.Entry interface
func (set PanelSetup) String() string {
	return fmt.Sprintf("%s, p0=%v, p1=%v, col=%v\n", set.cartHash, set.p0, set.p1, set.col)
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
