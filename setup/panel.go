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
	"fmt"
	"strconv"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
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

// PanelSetup is used to adjust the VCS's front panel.
type PanelSetup struct {
	cartHash string

	p0  bool
	p1  bool
	col bool

	notes string
}

func deserialisePanelSetupEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &PanelSetup{}

	// basic sanity check
	if len(fields) > numPanelSetupFields {
		return nil, curated.Errorf("panel: too many fields in panel entry")
	}
	if len(fields) < numPanelSetupFields {
		return nil, curated.Errorf("panel: too few fields in panel entry")
	}

	var err error

	set.cartHash = fields[panelSetupFieldCartHash]

	if set.p0, err = strconv.ParseBool(fields[panelSetupFieldP0]); err != nil {
		return nil, curated.Errorf("panel: invalid player 0 setting")
	}

	if set.p1, err = strconv.ParseBool(fields[panelSetupFieldP1]); err != nil {
		return nil, curated.Errorf("panel: invalid player 1 setting")
	}

	if set.col, err = strconv.ParseBool(fields[panelSetupFieldCol]); err != nil {
		return nil, curated.Errorf("panel: invalid color setting")
	}

	set.notes = fields[panelSetupFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface.
func (set PanelSetup) ID() string {
	return panelSetupID
}

// String implements the database.Entry interface.
func (set PanelSetup) String() string {
	return fmt.Sprintf("%s, p0=%v, p1=%v, col=%v\n", set.cartHash, set.p0, set.p1, set.col)
}

// Serialise implements the database.Entry interface.
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

// CleanUp implements the database.Entry interface.
func (set PanelSetup) CleanUp() error {
	// no cleanup necessary
	return nil
}

// matchCartHash implements setupEntry interface.
func (set PanelSetup) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface.
func (set PanelSetup) apply(vcs *hardware.VCS) error {
	if set.p0 {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetPlayer0Pro, true); err != nil {
			return err
		}
	} else {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetPlayer0Pro, false); err != nil {
			return err
		}
	}

	if set.p1 {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetPlayer1Pro, true); err != nil {
			return err
		}
	} else {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetPlayer1Pro, false); err != nil {
			return err
		}
	}

	if set.col {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetColor, true); err != nil {
			return err
		}
	} else {
		if err := vcs.RIOT.Ports.HandleEvent(ports.PanelID, ports.PanelSetColor, false); err != nil {
			return err
		}
	}

	return nil
}
