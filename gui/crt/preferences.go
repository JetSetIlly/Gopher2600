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

	InputGamma  prefs.Float
	OutputGamma prefs.Float

	Mask      prefs.Bool
	Scanlines prefs.Bool
	Noise     prefs.Bool

	MaskBrightness      prefs.Float
	ScanlinesBrightness prefs.Float
	NoiseLevel          prefs.Float
	MaskScanlineScaling prefs.Int

	Vignette prefs.Bool
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

const (
	inputGamma          = 2.4
	outputGamma         = 2.2
	mask                = true
	scanlines           = true
	noise               = true
	maskBrightness      = 0.70
	scanlinesBrightness = 0.70
	noiseLevel          = 0.10
	maskScanlineScaling = 1

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

	err = p.dsk.Add("crt.inputGamma", &p.InputGamma)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.outputGamma", &p.OutputGamma)
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
	err = p.dsk.Add("crt.maskBrightness", &p.MaskBrightness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesBrightness", &p.ScanlinesBrightness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskScanlineScaling", &p.MaskScanlineScaling)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.noiseLevel", &p.NoiseLevel)
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
	p.InputGamma.Set(inputGamma)
	p.OutputGamma.Set(outputGamma)
	p.Mask.Set(mask)
	p.Scanlines.Set(scanlines)
	p.Noise.Set(noise)
	p.MaskBrightness.Set(maskBrightness)
	p.ScanlinesBrightness.Set(scanlinesBrightness)
	p.MaskScanlineScaling.Set(maskScanlineScaling)
	p.NoiseLevel.Set(noiseLevel)
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
