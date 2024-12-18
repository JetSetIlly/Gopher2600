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

type Colour struct {
	dsk *prefs.Disk

	Brightness prefs.Float
	Contrast   prefs.Float
	Saturation prefs.Float
	Hue        prefs.Float
	Adj        prefs.Float
}

func (p *Colour) String() string {
	return p.dsk.String()
}

const (
	brightness = 1.0
	contrast   = 1.0
	saturation = 1.0
	hue        = 0.0
)

func newColour() (*Colour, error) {
	p := &Colour{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("display.color.brightness", &p.Brightness)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("display.color.contrast", &p.Contrast)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("display.color.saturation", &p.Saturation)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("display.color.hue", &p.Hue)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all colour values to default
func (p *Colour) SetDefaults() {
	p.Brightness.Set(brightness)
	p.Contrast.Set(contrast)
	p.Saturation.Set(saturation)
	p.Hue.Set(hue)
}

// Load colour values from disk
func (p *Colour) Load() error {
	return p.dsk.Load()
}

// Save current colour values to disk
func (p *Colour) Save() error {
	return p.dsk.Save()
}
