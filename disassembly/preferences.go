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

package disassembly

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

type Preferences struct {
	dsm *Disassembly
	dsk *prefs.Disk

	// whether to apply the high mirror bits to the displayed address
	FxxxMirror prefs.Bool
	Symbols    prefs.Bool

	// the lowest value to use when formatting address values. changed by the
	// preferences system
	mirrorOrigin uint16
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

// newPreferences is the preferred method of initialisation for the Preferences type.
func newPreferences(dsm *Disassembly) (*Preferences, error) {
	p := &Preferences{
		dsm:          dsm,
		mirrorOrigin: memorymap.OriginCart,
	}

	// save server using the prefs package
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("disassembly.fxxxMirror", &p.FxxxMirror)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("disassembly.symbols", &p.Symbols)
	if err != nil {
		return nil, err
	}

	p.FxxxMirror.RegisterCallback(func(v prefs.Value) error {
		if v.(bool) {
			p.mirrorOrigin = memorymap.OriginCartFxxxMirror
		} else {
			p.mirrorOrigin = memorymap.OriginCart
		}
		dsm.setCartMirror()
		return nil
	})

	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Load disassembly preferences and apply to the current disassembly.
func (p *Preferences) Load() error {
	err := p.dsk.Load(false)
	if err != nil {
		return err
	}
	return nil
}

// Save current disassembly preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}

// setCartMirror sets the mirror bits to the user's preference. called by the
// FxxxMirror callback.
func (dsm *Disassembly) setCartMirror() {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	for b := range dsm.entries {
		for _, e := range dsm.entries[b] {
			// mask off bits that indicate the cartridge/segment origin and reset
			// them with the chosen origin
			a := e.Result.Address&memorymap.CartridgeBits | dsm.Prefs.mirrorOrigin
			e.Address = fmt.Sprintf("$%04x", a)

			// branch instructions need special handling because for readability we
			// translate the offset to an absolute address, which has changed.
			if e.Result.Defn.IsBranch() {
				// mask off bits that indicate the cartridge/segment origin and reset
				// them with the chosen origin
				a := e.Result.Address&memorymap.CartridgeBits | dsm.Prefs.mirrorOrigin
				e.Operand.nonSymbolic = fmt.Sprintf("$%04x", absoluteBranchDestination(a, e.Result.InstructionData))
			}
		}
	}
}
