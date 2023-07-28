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
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

// Indicators that the ARM should put the MAM into the mode inidcated by the
// emulated driver for the cartridge mapper.
const MAMDriver = -1

type ARMPreferences struct {
	dsk *prefs.Disk

	// the specific model of ARM to use. this will affect things like memory
	// addressing for cartridge formats that use the ARM.
	//
	// NOTE: this may be superceded in the future to allow for more flexibility
	Model prefs.String

	// speed of processor
	Clock prefs.Float // Mhz

	// whether the ARM coprocessor (as found in Harmony cartridges) executes
	// instantly
	Immediate prefs.Bool

	// a value of MAMDriver says to use the driver supplied MAM value. any
	// other value "forces" the MAM setting on Thumb program execution.
	MAM prefs.Int

	// abort execution on memory fault (eg. accessing memory that doesn't exist)
	AbortOnMemoryFault prefs.Bool

	// include disassembly and register details when logging memory faults
	ExtendedMemoryFaultLogging prefs.Bool
}

func (p *ARMPreferences) String() string {
	return p.dsk.String()
}

func newARMprefrences() (*ARMPreferences, error) {
	p := &ARMPreferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.model", &p.Model)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.clock", &p.Clock)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.immediate", &p.Immediate)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.mam", &p.MAM)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.abortOnMemoryFault", &p.AbortOnMemoryFault)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.extendedMemoryFaultLogging", &p.ExtendedMemoryFaultLogging)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *ARMPreferences) SetDefaults() {
	// initialise random number generator
	p.Model.Set("AUTO")
	p.Clock.Set(70.0)
	p.Immediate.Set(false)
	p.MAM.Set(-1)
	p.AbortOnMemoryFault.Set(false)
	p.ExtendedMemoryFaultLogging.Set(false)
}

// Load current arm preference from disk.
func (p *ARMPreferences) Load() error {
	return p.dsk.Load(false)
}

// Save current arm preferences to disk.
func (p *ARMPreferences) Save() error {
	return p.dsk.Save()
}
