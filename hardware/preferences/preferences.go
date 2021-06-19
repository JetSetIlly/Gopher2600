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
	"math/rand"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi/armclocks"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// Preferences defines and collates all the preference values used by the debugger.
type Preferences struct {
	dsk *prefs.Disk

	// random values generated in the hardware package should use the following
	// number source
	RandSrc *rand.Rand

	// the number used to seed RandSrc
	RandSeed int64

	// initialise hardware to unknown state after reset
	RandomState prefs.Bool

	// unused pins when reading TIA/RIOT registers take the value of the last
	// value on the bus. if RandomPins is true then the values of the unusued
	// pins are randomised. this is the equivalent of the Stella option "drive
	// unused pins randomly on a read/peek"
	RandomPins prefs.Bool

	// ARM preferences are contained in their own type so they can be passed
	// easily to the ARM subsystem
	ARM ARMPreferences
}

// Indicats that the ARM should put the MAM into the mode inidcated by the
// emulated driver for the cartridge mapper.
const MAMDriver = -1

type ARMPreferences struct {
	// the specific model of ARM to use. this will affect things like memory
	// addressing for cartridge formats that use the ARM.
	//
	// NOTE: this may be superceded in the future to allow for more flexibility
	Model prefs.String

	// whether the ARM coprocessor (as found in Harmony cartridges) execute
	// instantly or if the cycle accurate steppint is attempted
	Immediate prefs.Bool

	// a value of MAMDriver says to use the driver supplied MAM value. any other value
	// "forces" the MAM setting on Thumb program execution.
	MAM prefs.Int

	Clock           prefs.Float
	FlashAccessTime prefs.Float
	SRAMAccessTime  prefs.Float
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

// NewPreferences is the preferred method of initialisation for the Preferences type.
func NewPreferences() (*Preferences, error) {
	p := &Preferences{}
	p.SetDefaults()

	// setup preferences and load from disk
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randState", &p.RandomState)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randPins", &p.RandomPins)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.model", &p.ARM.Model)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.immediate", &p.ARM.Immediate)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.mam", &p.ARM.MAM)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.clock", &p.ARM.Clock)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.flashAccessTime1", &p.ARM.FlashAccessTime)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.arm7.sramAccessTime", &p.ARM.SRAMAccessTime)
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
func (p *Preferences) SetDefaults() {
	// initialise random number generator
	p.Reseed(0)
	p.RandomState.Set(false)
	p.RandomPins.Set(false)
	p.ARM.Model.Set("LPC2000")
	p.ARM.Immediate.Set(false)
	p.ARM.MAM.Set(-1)
	p.ARM.Clock.Set(armclocks.MasterClock)               // Mhz
	p.ARM.FlashAccessTime.Set(armclocks.FlashAccessTime) // ns
	p.ARM.SRAMAccessTime.Set(armclocks.SRAMAccessTime)   // ns
}

// Reseed initialises the random number generator. Use a seed value of 0 to
// initialise with the current time.
func (p *Preferences) Reseed(seed int64) {
	if seed == 0 {
		p.RandSeed = int64(time.Now().Nanosecond())
	} else {
		p.RandSeed = seed
	}
	p.RandSrc = rand.New(rand.NewSource(p.RandSeed))
}

// Reset all hardware preferences to the default values.
func (p *Preferences) Reset() error {
	return p.dsk.Reset()
}

// Load current hardware preference from disk.
func (p *Preferences) Load() error {
	return p.dsk.Load(false)
}

// Save current hardware preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
