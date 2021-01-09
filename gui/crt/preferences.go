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
	Blur      prefs.Bool

	PhosphorSpeed       prefs.Float
	MaskBrightness      prefs.Float
	ScanlinesBrightness prefs.Float
	NoiseLevel          prefs.Float
	BlurLevel           prefs.Float

	Vignette prefs.Bool
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

const (
	enabled             = true
	phosphor            = true
	mask                = true
	scanlines           = true
	noise               = true
	blur                = true
	phosphorSpeed       = 1.0
	maskBrightness      = 0.70
	scanlinesBrightness = 0.70
	noiseLevel          = 0.10
	blurLevel           = 0.15

	vignette = true
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
	err = p.dsk.Add("crt.blur", &p.Blur)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorSpeed", &p.PhosphorSpeed)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskBrightness", &p.MaskBrightness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesBrightness", &p.ScanlinesBrightness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.noiseLevel", &p.NoiseLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.blurLevel", &p.BlurLevel)
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
	p.Blur.Set(blur)
	p.PhosphorSpeed.Set(phosphorSpeed)
	p.MaskBrightness.Set(maskBrightness)
	p.ScanlinesBrightness.Set(scanlinesBrightness)
	p.NoiseLevel.Set(noiseLevel)
	p.BlurLevel.Set(blurLevel)
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
