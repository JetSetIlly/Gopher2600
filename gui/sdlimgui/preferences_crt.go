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

package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type preferencesCRT struct {
	img *SdlImgui

	dsk *prefs.Disk

	pixelPerfect         prefs.Bool
	pixelPerfectFade     prefs.Float
	useBevel             prefs.Bool
	curve                prefs.Bool
	curveAmount          prefs.Float
	roundedCorners       prefs.Bool
	roundedCornersAmount prefs.Float
	scanlines            prefs.Bool
	scanlinesIntensity   prefs.Float
	mask                 prefs.Bool
	maskIntensity        prefs.Float
	rfInterference       prefs.Bool
	rfNoiseLevel         prefs.Float
	rfGhostingLevel      prefs.Float
	phosphor             prefs.Bool
	phosphorLatency      prefs.Float
	phosphorBloom        prefs.Float
	chromaticAberration  prefs.Float
	sharpness            prefs.Float
	blackLevel           prefs.Float
	shine                prefs.Bool

	ambientTint         prefs.Bool
	ambientTintStrength prefs.Float
}

func (p *preferencesCRT) String() string {
	return p.dsk.String()
}

func newPreferenceCRT(img *SdlImgui) (*preferencesCRT, error) {
	p := &preferencesCRT{
		img: img,
	}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("crt.pixelPerfect", &p.pixelPerfect)
	if err != nil {
		return nil, err
	}
	p.pixelPerfect.SetHookPost(func(v prefs.Value) error {
		if p.img.playScr != nil {
			// resize playscreen on pixelPerfect change because the CRT and
			// pixel perfect displays may have different scaling requirements
			p.img.playScr.resize()
		}
		return nil
	})
	err = p.dsk.Add("crt.pixelPerfectFade", &p.pixelPerfectFade)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.useBevel", &p.useBevel)
	if err != nil {
		return nil, err
	}
	p.useBevel.SetHookPost(func(v prefs.Value) error {
		if p.img.playScr != nil {
			// resize playscreen on useBevel change because the bevel and non
			// bevel displays may have different scaling requirements
			p.img.playScr.resize()
		}
		return nil
	})
	err = p.dsk.Add("crt.curve", &p.curve)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.curveAmount", &p.curveAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCorners", &p.roundedCorners)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.roundedCornersAmount", &p.roundedCornersAmount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlines", &p.scanlines)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.scanlinesIntensity", &p.scanlinesIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.mask", &p.mask)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.maskIntensity", &p.maskIntensity)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.rfInterference", &p.rfInterference)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.rfNoiseLevel", &p.rfNoiseLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.rfGhostingLevel", &p.rfGhostingLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphor", &p.phosphor)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorLatency", &p.phosphorLatency)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.phosphorBloom", &p.phosphorBloom)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.chromaticAberration", &p.chromaticAberration)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.sharpness", &p.sharpness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.blackLevel", &p.blackLevel)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.shine", &p.shine)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.ambient.tint", &p.ambientTint)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("crt.ambient.tintStrength", &p.ambientTintStrength)
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
func (p *preferencesCRT) SetDefaults() {
	p.pixelPerfect.Set(false)
	p.pixelPerfectFade.Set(0.4)
	p.useBevel.Set(true)
	p.curve.Set(true)
	p.curveAmount.Set(-0.030)
	p.roundedCorners.Set(true)
	p.roundedCornersAmount.Set(0.060)
	p.scanlines.Set(false)
	p.scanlinesIntensity.Set(0.039)
	p.mask.Set(false)
	p.maskIntensity.Set(0.037)
	p.rfInterference.Set(true)
	p.rfNoiseLevel.Set(0.127)
	p.rfGhostingLevel.Set(0.092)
	p.phosphor.Set(true)
	p.phosphorLatency.Set(0.5)
	p.phosphorBloom.Set(1.0)
	p.chromaticAberration.Set(0.044)
	p.sharpness.Set(0.408)
	p.blackLevel.Set(0.030)
	p.shine.Set(true)
	p.ambientTint.Set(false)
	p.ambientTintStrength.Set(0.3)
}

// Load CRT values from disk.
func (p *preferencesCRT) Load() error {
	return p.dsk.Load()
}

// Save current CRT values to disk.
func (p *preferencesCRT) Save() error {
	return p.dsk.Save()
}
