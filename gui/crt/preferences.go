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
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type Preferences struct {
	dsk *prefs.Disk

	Enabled prefs.Bool

	Curve          prefs.Bool
	RoundedCorners prefs.Bool
	Bevel          prefs.Bool
	Shine          prefs.Bool
	Mask           prefs.Bool
	Scanlines      prefs.Bool
	Interference   prefs.Bool
	Noise          prefs.Bool
	Fringing       prefs.Bool
	Ghosting       prefs.Bool
	Phosphor       prefs.Bool

	CurveAmount          prefs.Float
	RoundedCornersAmount prefs.Float
	MaskIntensity        prefs.Float
	MaskFine             prefs.Float
	ScanlinesIntensity   prefs.Float
	ScanlinesFine        prefs.Float
	InterferenceLevel    prefs.Float
	NoiseLevel           prefs.Float
	FringingAmount       prefs.Float
	GhostingAmount       prefs.Float
	PhosphorLatency      prefs.Float
	PhosphorBloom        prefs.Float
	Sharpness            prefs.Float
	BlackLevel           prefs.Float

	PixelPerfectFade prefs.Float

	SyncPowerOn     prefs.Bool
	SyncSpeed       prefs.Int
	SyncSensitivity prefs.Int

	IntegerScaling prefs.Bool
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

const (
	enabled              = true
	curve                = true
	roundedCorners       = false
	bevel                = false
	shine                = false
	mask                 = true
	scanlines            = true
	interference         = true
	noise                = true
	fringing             = true
	ghosting             = true
	phosphor             = true
	curveAmount          = 0.5
	roundedCornersAmount = 0.04
	maskIntensity        = 0.07
	maskFine             = 2.9
	scanlinesIntensity   = 0.08
	scanlinesFine        = 1.80
	interferenceLevel    = 0.15
	noiseLevel           = 0.19
	fringingAmount       = 0.15
	ghostingAmount       = 2.9
	phosphorLatency      = 0.5
	phosphorBloom        = 1.0
	sharpness            = 0.55
	blackLevel           = 0.06
	pixelPerfectFade     = 0.4
	syncSpeed            = 2
	syncPowerOn          = true
	syncSensitivity      = 2
	integerScaling       = false
)

// NewPreferences is the preferred method of initialisation for the Preferences type.
func NewPreferences() (*Preferences, error) {
	p := &Preferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
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
	err = p.dsk.Add("crt.curve", &p.Curve)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCorners", &p.RoundedCorners)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.bevel", &p.Bevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.shine", &p.Shine)
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
	err = p.dsk.Add("crt.interference", &p.Interference)
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
	err = p.dsk.Add("crt.ghosting", &p.Ghosting)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphor", &p.Phosphor)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.curveAmount", &p.CurveAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCornersAmount", &p.RoundedCornersAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskIntensity", &p.MaskIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskFine", &p.MaskFine)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesIntensity", &p.ScanlinesIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesFine", &p.ScanlinesFine)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.interferenceLevel", &p.InterferenceLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.noiseLevel", &p.NoiseLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.fringingAmount", &p.FringingAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.ghostingAmount", &p.GhostingAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorLatency", &p.PhosphorLatency)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorBloom", &p.PhosphorBloom)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.sharpness", &p.Sharpness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.blackLevel", &p.BlackLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.pixelPerfectFade", &p.PixelPerfectFade)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.syncPowerOn", &p.SyncPowerOn)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.syncSpeed", &p.SyncSpeed)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.syncSensitivity", &p.SyncSensitivity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.integerScaling", &p.IntegerScaling)
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
	p.Curve.Set(curve)
	p.RoundedCorners.Set(roundedCorners)
	p.Bevel.Set(bevel)
	p.Shine.Set(shine)
	p.Mask.Set(mask)
	p.Scanlines.Set(scanlines)
	p.Interference.Set(interference)
	p.Noise.Set(noise)
	p.Fringing.Set(fringing)
	p.Ghosting.Set(ghosting)
	p.Phosphor.Set(phosphor)
	p.CurveAmount.Set(curveAmount)
	p.RoundedCornersAmount.Set(roundedCornersAmount)
	p.MaskIntensity.Set(maskIntensity)
	p.MaskFine.Set(maskFine)
	p.ScanlinesIntensity.Set(scanlinesIntensity)
	p.ScanlinesFine.Set(scanlinesFine)
	p.InterferenceLevel.Set(interferenceLevel)
	p.NoiseLevel.Set(noiseLevel)
	p.FringingAmount.Set(fringingAmount)
	p.GhostingAmount.Set(ghostingAmount)
	p.PhosphorLatency.Set(phosphorLatency)
	p.PhosphorBloom.Set(phosphorBloom)
	p.Sharpness.Set(sharpness)
	p.BlackLevel.Set(blackLevel)
	p.PixelPerfectFade.Set(pixelPerfectFade)
	p.SyncPowerOn.Set(syncPowerOn)
	p.SyncSpeed.Set(syncSpeed)
	p.SyncSensitivity.Set(syncSensitivity)
	p.IntegerScaling.Set(integerScaling)
}

// Load disassembly preferences and apply to the current disassembly.
func (p *Preferences) Load() error {
	return p.dsk.Load(false)
}

// Save current disassembly preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
