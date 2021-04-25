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

package crt

import (
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

type Preferences struct {
	dsk *prefs.Disk

	Enabled prefs.Bool

	Phosphor  prefs.Bool
	Mask      prefs.Bool
	Scanlines prefs.Bool
	Noise     prefs.Bool
	Fringing  prefs.Bool
	Flicker   prefs.Bool

	PhosphorLatency prefs.Float
	BloomAmount     prefs.Float
	MaskBright      prefs.Float
	ScanlinesBright prefs.Float
	NoiseLevel      prefs.Float
	FringingLevel   prefs.Float
	FlickerLevel    prefs.Float

	Vignette prefs.Bool
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

const (
	enabled         = true
	phosphor        = true
	mask            = true
	scanlines       = true
	noise           = true
	fringing        = true
	flicker         = true
	phosphorLatency = 0.5
	bloomAmount     = 1.0
	maskBright      = 0.70
	scanlinesBright = 0.70
	noiseLevel      = 0.19
	fringingLevel   = 0.15
	flickerLevel    = 0.004
	vignette        = true
)

// NewPreferences is the preferred method of initialisation for the Preferences type.
func NewPreferences() (*Preferences, error) {
	p := &Preferences{}
	p.SetDefaults()

	// save server using the prefs package
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("crt.enabled", &p.Enabled)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphor", &p.Phosphor)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.mask", &p.Mask)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlines", &p.Scanlines)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.noise", &p.Noise)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.fringing", &p.Fringing)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.flicker", &p.Flicker)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorLatency", &p.PhosphorLatency)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.bloomAmount", &p.BloomAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskBright", &p.MaskBright)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesBright", &p.ScanlinesBright)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.noiseLevel", &p.NoiseLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.fringingLevel", &p.FringingLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.flickerlevel", &p.FlickerLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.vignette", &p.Vignette)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults revers all CRT settings to default values.
func (p *Preferences) SetDefaults() {
	p.Enabled.Set(enabled)
	p.Phosphor.Set(phosphor)
	p.Mask.Set(mask)
	p.Scanlines.Set(scanlines)
	p.Noise.Set(noise)
	p.Fringing.Set(fringing)
	p.Flicker.Set(flicker)
	p.PhosphorLatency.Set(phosphorLatency)
	p.BloomAmount.Set(bloomAmount)
	p.MaskBright.Set(maskBright)
	p.ScanlinesBright.Set(scanlinesBright)
	p.NoiseLevel.Set(noiseLevel)
	p.FringingLevel.Set(fringingLevel)
	p.FlickerLevel.Set(flickerLevel)
	p.Vignette.Set(vignette)
}

// Load disassembly preferences and apply to the current disassembly.
func (p *Preferences) Load() error {
	return p.dsk.Load(false)
}

// Save current disassembly preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
