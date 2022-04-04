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
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

// LiveRevisionPrefrences encapsulates the current (live) revision values.
//
// For performance critical situations these values should be preferred to the
// prefs.Bool values in RevisionPreferences.
type LiveRevisionPreferences struct {
	// The names of the preference fields match the Bug enumerations. These
	// values are updated automatically when the corresponding Dsk* field is
	// updated.
	LateVDELGRP0     atomic.Value // bool
	LateVDELGRP1     atomic.Value // bool
	LateRESPx        atomic.Value // bool
	EarlyScancounter atomic.Value // bool
	LatePFx          atomic.Value // bool
	LateCOLUPF       atomic.Value // bool
	LostMOTCK        atomic.Value // bool
	RESPxHBLANK      atomic.Value // bool
}

// RevisionPreferences defines the details of the TIA revisins.
type RevisionPreferences struct {
	dsk *prefs.Disk

	// Prefer live values in performance critical code
	Live LiveRevisionPreferences

	// Disk copies of preferences
	LateVDELGRP0     prefs.Bool
	LateVDELGRP1     prefs.Bool
	LateRESPx        prefs.Bool
	EarlyScancounter prefs.Bool
	LatePFx          prefs.Bool
	LateCOLUPF       prefs.Bool
	LostMOTCK        prefs.Bool
	RESPxHBLANK      prefs.Bool
}

func newRevisionPreferences() (*RevisionPreferences, error) {
	p := &RevisionPreferences{}

	// register callbacks to update the "live" values from the disk value
	p.LateVDELGRP0.SetHookPost(func(v prefs.Value) error {
		p.Live.LateVDELGRP0.Store(v.(bool))
		return nil
	})
	p.LateVDELGRP1.SetHookPost(func(v prefs.Value) error {
		p.Live.LateVDELGRP1.Store(v.(bool))
		return nil
	})
	p.LateRESPx.SetHookPost(func(v prefs.Value) error {
		p.Live.LateRESPx.Store(v.(bool))
		return nil
	})
	p.EarlyScancounter.SetHookPost(func(v prefs.Value) error {
		p.Live.EarlyScancounter.Store(v.(bool))
		return nil
	})
	p.LatePFx.SetHookPost(func(v prefs.Value) error {
		p.Live.LatePFx.Store(v.(bool))
		return nil
	})
	p.LateCOLUPF.SetHookPost(func(v prefs.Value) error {
		p.Live.LateCOLUPF.Store(v.(bool))
		return nil
	})
	p.LostMOTCK.SetHookPost(func(v prefs.Value) error {
		p.Live.LostMOTCK.Store(v.(bool))
		return nil
	})
	p.RESPxHBLANK.SetHookPost(func(v prefs.Value) error {
		p.Live.RESPxHBLANK.Store(v.(bool))
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

	err = p.dsk.Add("tia.revision.grp0.latevdel", &p.LateVDELGRP0)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.grp1.latevdel", &p.LateVDELGRP1)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.hmove.laterespx", &p.LateRESPx)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.hmove.earlyscancounter", &p.EarlyScancounter)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.playfield.latepfx", &p.LatePFx)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.playfield.latecolupf", &p.LateCOLUPF)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.lostmotck", &p.LostMOTCK)
	if err != nil {
		return nil, curated.Errorf("revision: %v", err)
	}

	err = p.dsk.Add("tia.revision.respx.hmovethreshold", &p.RESPxHBLANK)
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
	p.LateVDELGRP0.Set(false)
	p.LateVDELGRP1.Set(false)
	p.LateRESPx.Set(false)
	p.EarlyScancounter.Set(false)
	p.LatePFx.Set(false)
	p.LateCOLUPF.Set(false)
	p.LostMOTCK.Set(false)
	p.RESPxHBLANK.Set(false)
}

// Load revision preferences from disk.
func (p *RevisionPreferences) Load() error {
	return p.dsk.Load(false)
}

// Save current revision preferences to disk.
func (p *RevisionPreferences) Save() error {
	return p.dsk.Save()
}
