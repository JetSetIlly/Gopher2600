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

package preferences

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

// RevisionPreferences defines the details of the TIA revisins.
//
// Unlike the other preferences types in this pacakge, "live" values should be
// preferred in most case because accessing the Dsk* values through the prefs
// system is too slow.
//
// The Dsk* values should be used from any GUI preferences editor however - the
// live values will be updated automatically when the Dsk* value are changed.
type RevisionPreferences struct {
	dsk *prefs.Disk

	// The names of the preference fields match the Bug enumerations. These
	// values are updated automatically when the corresponding Dsk* field is
	// updated.
	LateVDELGRP0     bool
	LateVDELGRP1     bool
	LateRESPx        bool
	EarlyScancounter bool
	LatePFx          bool
	LateCOLUPF       bool
	LostMOTCK        bool
	RESPxHBLANK      bool

	// Disk copies of preferences
	DskLateVDELGRP0     prefs.Bool
	DskLateVDELGRP1     prefs.Bool
	DskLateRESPx        prefs.Bool
	DskEarlyScancounter prefs.Bool
	DskLatePFx          prefs.Bool
	DskLateCOLUPF       prefs.Bool
	DskLostMOTCK        prefs.Bool
	DskRESPxHBLANK      prefs.Bool
}

func newRevisionPreferences() (*RevisionPreferences, error) {
	p := &RevisionPreferences{}

	// register callbacks to update the "live" values from the disk value
	p.DskLateVDELGRP0.SetHookPost(func(v prefs.Value) error {
		p.LateVDELGRP0 = v.(bool)
		return nil
	})
	p.DskLateVDELGRP1.SetHookPost(func(v prefs.Value) error {
		p.LateVDELGRP1 = v.(bool)
		return nil
	})
	p.DskLateRESPx.SetHookPost(func(v prefs.Value) error {
		p.LateRESPx = v.(bool)
		return nil
	})
	p.DskEarlyScancounter.SetHookPost(func(v prefs.Value) error {
		p.EarlyScancounter = v.(bool)
		return nil
	})
	p.DskLatePFx.SetHookPost(func(v prefs.Value) error {
		p.LatePFx = v.(bool)
		return nil
	})
	p.DskLateCOLUPF.SetHookPost(func(v prefs.Value) error {
		p.LateCOLUPF = v.(bool)
		return nil
	})
	p.DskLostMOTCK.SetHookPost(func(v prefs.Value) error {
		p.LostMOTCK = v.(bool)
		return nil
	})
	p.DskRESPxHBLANK.SetHookPost(func(v prefs.Value) error {
		p.RESPxHBLANK = v.(bool)
		return nil
	})

	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.grp0.latevdel", &p.DskLateVDELGRP0)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.grp1.latevdel", &p.DskLateVDELGRP1)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.hmove.laterespx", &p.DskLateRESPx)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.hmove.earlyscancounter", &p.DskEarlyScancounter)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.playfield.latepfx", &p.DskLatePFx)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.playfield.latecolupf", &p.DskLateCOLUPF)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.lostmotck", &p.DskLostMOTCK)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.respx.hmovethreshold", &p.DskRESPxHBLANK)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *RevisionPreferences) SetDefaults() {
	p.DskLateVDELGRP0.Set(false)
	p.DskLateVDELGRP1.Set(false)
	p.DskLateRESPx.Set(false)
	p.DskEarlyScancounter.Set(false)
	p.DskLatePFx.Set(false)
	p.DskLateCOLUPF.Set(false)
	p.DskLostMOTCK.Set(false)
	p.DskRESPxHBLANK.Set(false)
}

// Load revision preferences from disk.
func (p *RevisionPreferences) Load() error {
	return p.dsk.Load(false)
}

// Save current revision preferences to disk.
func (p *RevisionPreferences) Save() error {
	return p.dsk.Save()
}
