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

package revision

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// Preferences for TIA revisins. Use the "live" values inside the emulation
// code. Accessing the Dsk* values through the prefs system is too slow.
//
// The Dsk* values should be used from any GUI preferences editor however - the
// live values will be updated automatically when the Dsk* value are changed.
type Preferences struct {
	dsk *prefs.Disk

	// The names of the preference fields match the Bug enumerations. These
	// values are updated automatically when the corresponding Dsk* field is
	// updated.
	LateVDELGRP0    bool
	LateVDELGRP1    bool
	LateRippleStart bool
	LateRippleEnd   bool
	LatePFx         bool
	LateCOLUPF      bool
	LostMOTCK       bool
	RESPxHBLANK     bool

	// Disk copies of preferences
	DskLateVDELGRP0    prefs.Bool
	DskLateVDELGRP1    prefs.Bool
	DskLateRippleStart prefs.Bool
	DskLateRippleEnd   prefs.Bool
	DskLatePFx         prefs.Bool
	DskLateCOLUPF      prefs.Bool
	DskLostMOTCK       prefs.Bool
	DskRESPxHBLANK     prefs.Bool
}

// NewPreferences is the preferred method of initialisation for the Preferences type.
func newPreferences() (*Preferences, error) {
	p := &Preferences{}

	// save server using the prefs package
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
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

	err = p.dsk.Add("tia.revision.hmove.ripplestart", &p.DskLateRippleStart)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.hmove.rippleend", &p.DskLateRippleEnd)
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

	// register callbacks to update the "live" values from the disk value
	p.DskLateVDELGRP0.RegisterCallback(func(v prefs.Value) error {
		p.LateVDELGRP0 = v.(bool)
		return nil
	})
	p.DskLateVDELGRP1.RegisterCallback(func(v prefs.Value) error {
		p.LateVDELGRP1 = v.(bool)
		return nil
	})
	p.DskLateRippleStart.RegisterCallback(func(v prefs.Value) error {
		p.LateRippleStart = v.(bool)
		return nil
	})
	p.DskLateRippleEnd.RegisterCallback(func(v prefs.Value) error {
		p.LateRippleEnd = v.(bool)
		return nil
	})
	p.DskLatePFx.RegisterCallback(func(v prefs.Value) error {
		p.LatePFx = v.(bool)
		return nil
	})
	p.DskLateCOLUPF.RegisterCallback(func(v prefs.Value) error {
		p.LateCOLUPF = v.(bool)
		return nil
	})
	p.DskLostMOTCK.RegisterCallback(func(v prefs.Value) error {
		p.LostMOTCK = v.(bool)
		return nil
	})
	p.DskRESPxHBLANK.RegisterCallback(func(v prefs.Value) error {
		p.RESPxHBLANK = v.(bool)
		return nil
	})

	err = p.dsk.Load(true)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	return p, nil
}

// Load disassembly preferences and apply to the current disassembly.
func (p *Preferences) Load() error {
	return p.dsk.Load(false)
}

// Save current disassembly preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
