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

	PixelPerfect         prefs.Bool
	PixelPerfectFade     prefs.Float
	Curve                prefs.Bool
	CurveAmount          prefs.Float
	RoundedCorners       prefs.Bool
	RoundedCornersAmount prefs.Float
	Scanlines            prefs.Bool
	ScanlinesIntensity   prefs.Float
	Mask                 prefs.Bool
	MaskIntensity        prefs.Float
	Interference         prefs.Bool
	InterferenceLevel    prefs.Float
	Phosphor             prefs.Bool
	PhosphorLatency      prefs.Float
	PhosphorBloom        prefs.Float
	ChromaticAberration  prefs.Float
	Sharpness            prefs.Float
	BlackLevel           prefs.Float
	Shine                prefs.Bool
}

func (p *CRT) String() string {
	return p.dsk.String()
}

func NewCRT() (*CRT, error) {
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

	err = p.dsk.Add("crt.pixelPerfect", &p.PixelPerfect)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.pixelPerfectFade", &p.PixelPerfectFade)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.curve", &p.Curve)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.curveAmount", &p.CurveAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCorners", &p.RoundedCorners)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCornersAmount", &p.RoundedCornersAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlines", &p.Scanlines)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesIntensity", &p.ScanlinesIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.mask", &p.Mask)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskIntensity", &p.MaskIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.interference", &p.Interference)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.interferenceLevel", &p.InterferenceLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphor", &p.Phosphor)
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
	err = p.dsk.Add("crt.chromaticAberration", &p.ChromaticAberration)
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
	err = p.dsk.Add("crt.shine", &p.Shine)
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
	p.PixelPerfect.Set(false)
	p.PixelPerfectFade.Set(0.4)
	p.Curve.Set(true)
	p.CurveAmount.Set(0.5)
	p.RoundedCorners.Set(true)
	p.RoundedCornersAmount.Set(0.059)
	p.Scanlines.Set(true)
	p.ScanlinesIntensity.Set(0.08)
	p.Mask.Set(false)
	p.MaskIntensity.Set(0.07)
	p.Interference.Set(true)
	p.InterferenceLevel.Set(0.15)
	p.Phosphor.Set(true)
	p.PhosphorLatency.Set(0.5)
	p.PhosphorBloom.Set(1.0)
	p.ChromaticAberration.Set(0.15)
	p.Sharpness.Set(0.55)
	p.BlackLevel.Set(0.045)
	p.Shine.Set(true)
}

// Load CRT values from disk.
func (p *CRT) Load() error {
	return p.dsk.Load()
}

// Save current CRT values to disk.
func (p *CRT) Save() error {
	return p.dsk.Save()
}
