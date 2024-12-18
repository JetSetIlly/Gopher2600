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

package display

import (
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type CRT struct {
	dsk *prefs.Disk

	Enabled prefs.Bool

	Curve          prefs.Bool
	RoundedCorners prefs.Bool
	Bevel          prefs.Bool
	Shine          prefs.Bool
	Mask           prefs.Bool
	Scanlines      prefs.Bool
	Interference   prefs.Bool
	Flicker        prefs.Bool
	Fringing       prefs.Bool
	Ghosting       prefs.Bool
	Phosphor       prefs.Bool

	CurveAmount          prefs.Float
	RoundedCornersAmount prefs.Float
	BevelSize            prefs.Float
	MaskIntensity        prefs.Float
	ScanlinesIntensity   prefs.Float
	InterferenceLevel    prefs.Float
	FlickerLevel         prefs.Float
	FringingAmount       prefs.Float
	GhostingAmount       prefs.Float
	PhosphorLatency      prefs.Float
	PhosphorBloom        prefs.Float
	Sharpness            prefs.Float
	BlackLevel           prefs.Float

	PixelPerfectFade prefs.Float
}

func (p *CRT) String() string {
	return p.dsk.String()
}

func newCRT() (*CRT, error) {
	p := &CRT{}
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
	err = p.dsk.Add("crt.flicker", &p.Flicker)
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
	err = p.dsk.Add("crt.bevelSize", &p.BevelSize)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskIntensity", &p.MaskIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesIntensity", &p.ScanlinesIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.interferenceLevel", &p.InterferenceLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.flickerLevel", &p.FlickerLevel)
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

	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults revers all CRT settings to default values.
func (p *CRT) SetDefaults() {
	p.Enabled.Set(true)
	p.Curve.Set(true)
	p.RoundedCorners.Set(true)
	p.Bevel.Set(false)
	p.Shine.Set(true)
	p.Mask.Set(false)
	p.Scanlines.Set(true)
	p.Interference.Set(true)
	p.Flicker.Set(false)
	p.Fringing.Set(true)
	p.Ghosting.Set(true)
	p.Phosphor.Set(true)
	p.CurveAmount.Set(0.5)
	p.RoundedCornersAmount.Set(0.059)
	p.BevelSize.Set(0.01)
	p.MaskIntensity.Set(0.07)
	p.ScanlinesIntensity.Set(0.08)
	p.InterferenceLevel.Set(0.15)
	p.FlickerLevel.Set(0.025)
	p.FringingAmount.Set(0.15)
	p.GhostingAmount.Set(2.9)
	p.PhosphorLatency.Set(0.5)
	p.PhosphorBloom.Set(1.0)
	p.Sharpness.Set(0.55)
	p.BlackLevel.Set(0.045)

	p.PixelPerfectFade.Set(0.4)
}

// Load CRT values from disk.
func (p *CRT) Load() error {
	return p.dsk.Load()
}

// Save current CRT values to disk.
func (p *CRT) Save() error {
	return p.dsk.Save()
}
