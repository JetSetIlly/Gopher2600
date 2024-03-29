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

	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

const panelSetupEntryType = "panel"

const (
	panelSetupFieldCartHash int = iota
	panelSetupFieldCartName
	panelSetupFieldP0
	panelSetupFieldP1
	panelSetupFieldCol
	panelSetupFieldNotes
	numPanelSetupFields
)

// PanelSetup is used to adjust the VCS's front panel.
type PanelSetup struct {
	cartHash string
	cartName string

	p0  bool
	p1  bool
	col bool

	notes string
}

func deserialisePanelSetupEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &PanelSetup{}

	// basic sanity check
	if len(fields) > numPanelSetupFields {
		return nil, fmt.Errorf("panel: too many fields in panel entry")
	}
	if len(fields) < numPanelSetupFields {
		return nil, fmt.Errorf("panel: too few fields in panel entry")
	}

	var err error

	set.cartHash = fields[panelSetupFieldCartHash]
	set.cartName = fields[panelSetupFieldCartName]

	if set.p0, err = strconv.ParseBool(fields[panelSetupFieldP0]); err != nil {
		return nil, fmt.Errorf("panel: invalid player 0 setting")
	}

	if set.p1, err = strconv.ParseBool(fields[panelSetupFieldP1]); err != nil {
		return nil, fmt.Errorf("panel: invalid player 1 setting")
	}

	if set.col, err = strconv.ParseBool(fields[panelSetupFieldCol]); err != nil {
		return nil, fmt.Errorf("panel: invalid color setting")
	}

	set.notes = fields[panelSetupFieldNotes]

	return set, nil
}

// EntryType implements the database.Entry interface.
func (set PanelSetup) EntryType() string {
	return panelSetupEntryType
}

// Serialise implements the database.Entry interface.
func (set *PanelSetup) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			set.cartName,
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
func (set PanelSetup) apply(vcs *hardware.VCS) (string, error) {
	if set.p0 {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer0Pro, D: true}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	} else {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer0Pro, D: false}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	}

	if set.p1 {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer1Pro, D: true}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	} else {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetPlayer1Pro, D: false}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	}

	if set.col {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetColor, D: true}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	} else {
		inp := ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSetColor, D: false}
		if _, err := vcs.Input.HandleInputEvent(inp); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("panel preset: %s: %s", set.cartName, set.notes), nil
}
