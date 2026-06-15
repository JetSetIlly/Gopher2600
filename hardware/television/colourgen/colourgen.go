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

package colourgen

import (
	"image/color"
	"math"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type entry struct {
	col       color.RGBA
	generated bool
}

// ColourGen creates and caches colour values for the different types of
// television systems
type ColourGen struct {
	// cached colour entries. used for legacy and non-legacy colour models
	ntsc      []entry
	pal       []entry
	secam     []entry
	zeroBlack entry

	dsk *prefs.Disk

	AdjustNTSC  Adjust
	AdjustPAL   Adjust
	AdjustSECAM Adjust

	// gamma is the same for both legacy and non-legacy models
	Gamma prefs.Float
}

// NewColourGen is the preferred method of intialisation for the ColourGen type.
func NewColourGen() (*ColourGen, error) {
	c := &ColourGen{
		ntsc:  make([]entry, 128),
		pal:   make([]entry, 128),
		secam: make([]entry, 2048),
	}

	c.SetDefaults(true, "")

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	c.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = c.dsk.Add("television.color.ntsc.brightness", &c.AdjustNTSC.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.ntsc.contrast", &c.AdjustNTSC.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.ntsc.saturation", &c.AdjustNTSC.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.ntsc.hue", &c.AdjustNTSC.Hue)
	if err != nil {
		return nil, err
	}

	err = c.dsk.Add("television.color.pal.brightness", &c.AdjustPAL.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.pal.contrast", &c.AdjustPAL.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.pal.saturation", &c.AdjustPAL.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.pal.hue", &c.AdjustPAL.Hue)
	if err != nil {
		return nil, err
	}

	err = c.dsk.Add("television.color.secam.brightness", &c.AdjustSECAM.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.secam.contrast", &c.AdjustSECAM.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.secam.saturation", &c.AdjustSECAM.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.secam.hue", &c.AdjustSECAM.Hue)
	if err != nil {
		return nil, err
	}

	// the cache of generated colours are always cleared when most adjustment
	// settings are changed
	//
	// the only exceptions are the phase values that only need to clear NTSC or
	// PAL depending on which phase is being adjusted
	f := func(_ prefs.Value) error {
		clear(c.ntsc)
		clear(c.pal)
		clear(c.secam)
		c.zeroBlack.generated = false
		return nil
	}

	c.AdjustNTSC.Brightness.SetHookPost(f)
	c.AdjustNTSC.Contrast.SetHookPost(f)
	c.AdjustNTSC.Saturation.SetHookPost(f)
	c.AdjustNTSC.Hue.SetHookPost(f)
	c.AdjustPAL.Brightness.SetHookPost(f)
	c.AdjustPAL.Contrast.SetHookPost(f)
	c.AdjustPAL.Saturation.SetHookPost(f)
	c.AdjustPAL.Hue.SetHookPost(f)
	c.AdjustSECAM.Brightness.SetHookPost(f)
	c.AdjustSECAM.Contrast.SetHookPost(f)
	c.AdjustSECAM.Saturation.SetHookPost(f)
	c.AdjustSECAM.Hue.SetHookPost(f)

	err = c.dsk.Add("television.color.ntsc.phase", &c.AdjustNTSC.Phase)
	if err != nil {
		return nil, err
	}
	c.AdjustNTSC.Phase.SetHookPost(func(_ prefs.Value) error {
		clear(c.ntsc)
		c.zeroBlack.generated = false
		return nil
	})

	err = c.dsk.Add("television.color.pal.phase", &c.AdjustPAL.Phase)
	if err != nil {
		return nil, err
	}
	c.AdjustPAL.Phase.SetHookPost(func(_ prefs.Value) error {
		clear(c.pal)
		c.zeroBlack.generated = false
		return nil
	})

	// gamma is the same for every variation of colour generation
	err = c.dsk.Add("television.color.gamma", &c.Gamma)
	if err != nil {
		return nil, err
	}
	c.Gamma.SetHookPost(f)

	err = c.dsk.Load()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *ColourGen) SetDefaults(all bool, spec string) {
	if spec == "NTSC" || all {
		c.AdjustNTSC.Brightness.Set(1.196)
		c.AdjustNTSC.Contrast.Set(1.000)
		c.AdjustNTSC.Saturation.Set(0.963)
		c.AdjustNTSC.Hue.Set(0.0)
		c.AdjustNTSC.Phase.Set(0.0)
	}

	if spec == "PAL" || all {
		c.AdjustPAL.Brightness.Set(1.196)
		c.AdjustPAL.Contrast.Set(1.000)
		c.AdjustPAL.Saturation.Set(0.963)
		c.AdjustPAL.Hue.Set(0.0)
		c.AdjustPAL.Phase.Set(0.0)
	}

	if spec == "SECAM" || all {
		c.AdjustSECAM.Brightness.Set(1.037)
		c.AdjustSECAM.Contrast.Set(0.873)
		c.AdjustSECAM.Saturation.Set(0.996)
		c.AdjustSECAM.Hue.Set(0.0)
		c.AdjustSECAM.Phase.Set(0.0)
	}

	// I used to think that the different TV specifications had a specific
	// gamma. NTSC has an inherent gamma of 2.2 and PAL has 2.8. I no longer
	// belive this and now use a single gamma value for all specififcations.
	//
	// this is currently 2.2 and if the user wants to change it, they need to
	// change the preference file
	c.Gamma.Set(Gamma)
}

// Load colour values from disk
func (c *ColourGen) Load() error {
	return c.dsk.Load()
}

// Save current colour values to disk
func (c *ColourGen) Save() error {
	return c.dsk.Save()
}

func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// the black level for a ZeroBlank signal
const zeroBlack = 0.0

// there are some RGB mods that output 7.5 IRE for "black" and 0.0 for ZeroBlank. we
// don't emulate that difference yet. if we did it would complicate handling of
// the legacy palettes a little bit
//
// Note that according to the NTSC standard videoBlack should be 0.075 (7.5 IRE)
// but the 2600 does not seem to output that value
//
// IRE levels taken from https://en.wikipedia.org/w/index.php?title=NTSC&oldid=1274410783
//
// # PAL Video Black is 0.0 IRE I believe which simplifies things for us
//
// a great example of a ROM that uses ZeroBlack as a shortcut for VideoBlack
// (ie. it enabled VBLANK instead of dealing with colour registers) is the CDFJ
// game Boom, by Chris Walton. in this game the bouncing AtariAge logo at the
// beginning of the is bounded by a VBLANK "box"
const videoBlack = zeroBlack

func (c *ColourGen) generateZeroBlack(adjust Adjust) color.RGBA {
	if c.zeroBlack.generated {
		return c.zeroBlack.col
	}

	Y, _, _ := adjust.yiq(videoBlack, 0, 0)

	y := uint8(clamp(Y) * 255)
	c.zeroBlack.col = color.RGBA{R: y, G: y, B: y, A: 255}
	c.zeroBlack.generated = true

	gamma := c.Gamma.Get().(float64)
	c.zeroBlack.col = gammaCorrectRGB(c.zeroBlack.col, gamma)

	return c.zeroBlack.col
}

// GenerateNTSC creates the RGB values for the ColorSignal using the current
// colour generation preferences and for the NTSC television system
func (c *ColourGen) GenerateNTSC(col signal.ColorSignal, _ signal.ColorSignal, _ bool) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack(c.AdjustNTSC)
	}

	idx := uint8(col) >> 1

	if c.ntsc[idx].generated == true {
		return c.ntsc[idx].col
	}

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// special case for colour-luminance of zero
	if hue == 0x00 && lum == 0x00 {
		c.ntsc[idx].col = c.generateZeroBlack(c.AdjustNTSC)
		c.ntsc[idx].generated = true
		return c.ntsc[idx].col
	}

	// Y, phi and saturation is all that's needed to create the RGB value
	Y := legacyNTSC_yiq[hue][lum].y
	phi := legacyNTSC_yiq[hue][lum].phi
	saturation := legacyNTSC_yiq[hue][lum].saturation

	// stretch phi by a fraction of the phase setting
	phi -= (float64(hue - 1)) * c.AdjustNTSC.Phase.Get().(float64) * math.Pi / 180

	// (IQ used to by multplied by the luminance (Y) value but I no longer
	// believe this is correct)
	I := saturation * math.Sin(phi)
	Q := saturation * math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YIQ
	Y, I, Q = c.AdjustNTSC.yiq(Y, I, Q)

	// YIQ to RGB conversion
	//
	// YIQ conversion values taken from the "NTSC 1953 colorimetry" section
	// of: https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306
	R := clamp(Y + (0.956 * I) + (0.619 * Q))
	G := clamp(Y - (0.272 * I) - (0.647 * Q))
	B := clamp(Y - (1.106 * I) + (1.703 * Q))

	// from the "FCC NTSC Standard (SMPTE C)" of the same wikipedia article
	// 	R := clamp(Y + (0.9469 * I) + (0.6236 * Q))
	// 	G := clamp(Y - (0.2748 * I) - (0.6357 * Q))
	// 	B := clamp(Y - (1.1 * I) + (1.7 * Q))

	// the coefficients used by Stella (7.0)
	// 	R := clamp(Y + (0.9563 * I) + (0.6210 * Q))
	// 	G := clamp(Y - (0.2721 * I) - (0.6474 * Q))
	// 	B := clamp(Y - (1.1070 * I) + (1.7046 * Q))

	// gamma correction
	gamma := c.Gamma.Get().(float64)
	R, G, B = gammaCorrect(R, G, B, gamma)

	// create and cache
	c.ntsc[idx].generated = true
	c.ntsc[idx].col = color.RGBA{
		R: uint8(R * 255.0),
		G: uint8(G * 255.0),
		B: uint8(B * 255.0),
		A: 255,
	}

	return c.ntsc[idx].col
}

// GeneratePAL creates the RGB values for the ColorSignal using the current
// colour generation preferences and for the PAL television system
func (c *ColourGen) GeneratePAL(col signal.ColorSignal, _ signal.ColorSignal, _ bool) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack(c.AdjustPAL)
	}

	idx := uint8(col) >> 1

	if c.pal[idx].generated == true {
		return c.pal[idx].col
	}

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// special case for colour-luminance of zero
	if hue == 0x00 && lum == 0x00 {
		c.pal[idx].col = c.generateZeroBlack(c.AdjustPAL)
		c.pal[idx].generated = true
		return c.pal[idx].col
	}

	// Y, phi and saturation is all that's needed to create the RGB value
	Y := legacyPAL_yuv[hue][lum].y
	phi := legacyPAL_yuv[hue][lum].phi
	saturation := legacyPAL_yuv[hue][lum].saturation

	// stretch phi by a fraction of the phase setting
	if hue&0x01 == 0x01 {
		phi -= float64(hue) * c.AdjustPAL.Phase.Get().(float64) * math.Pi / 180
	} else {
		phi += (float64(hue) - 2) * c.AdjustPAL.Phase.Get().(float64) * math.Pi / 180
	}

	// (UV used to by multplied by the luminance (Y) value but I no longer
	// believe this is correct)
	U := saturation * -math.Sin(phi)
	V := saturation * -math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YUV
	Y, U, V = c.AdjustNTSC.yuv(Y, U, V)

	// YUV to RGB conversion
	//
	// YUV conversion values taken from the "SDTV with BT.470" section of:
	// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
	R := clamp(Y + (1.140 * V))
	G := clamp(Y - (0.395 * U) - (0.581 * V))
	B := clamp(Y + (2.033 * U))

	// gamma correction
	gamma := c.Gamma.Get().(float64)
	R, G, B = gammaCorrect(R, G, B, gamma)

	// create and cache
	c.pal[idx].generated = true
	c.pal[idx].col = color.RGBA{
		R: uint8(R * 255.0),
		G: uint8(G * 255.0),
		B: uint8(B * 255.0),
		A: 255,
	}

	return c.pal[idx].col
}

func (c *ColourGen) GenerateSECAM(col signal.ColorSignal, store signal.ColorSignal, odd bool) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack(c.AdjustSECAM)
	}

	// the hue nibble of the two signal.ColourSignal values is ignored by SECAM
	lum := (col & 0x0e) >> 1
	storeLum := (store & 0x0e) >> 1

	// index is based on lum value of the two colour signals
	idx := (int(lum) | (int(storeLum) << 7)) << 1
	if odd {
		idx |= 1
	}

	// use indexed colour if available
	if c.secam[idx].generated == true {
		return c.secam[idx].col
	}

	// special case for luminance of zero (both luminance values are added together)
	if lum+storeLum == 0x00 {
		c.secam[idx].col = c.generateZeroBlack(c.AdjustSECAM)
		c.secam[idx].generated = true
		return c.secam[idx].col
	}

	// Y, phi and saturation can be looked up based on lum
	Y := legacySECAM_yuv[lum].y
	phi := legacySECAM_yuv[lum].phi
	saturation := legacySECAM_yuv[lum].saturation

	// phi and saturation only for stored signal
	storePhi := legacySECAM_yuv[storeLum].phi
	storeSaturation := legacySECAM_yuv[storeLum].saturation

	// (U and V used to by multplied by the luminance (Y) value but I no longer believe this is correct)
	var U, V float64
	if odd {
		U = saturation * -math.Sin(phi)
		V = storeSaturation * -math.Cos(storePhi)
	} else {
		U = storeSaturation * -math.Sin(storePhi)
		V = saturation * -math.Cos(phi)
	}

	// apply brightness/constrast/saturation/hue settings to YUV
	Y, U, V = c.AdjustSECAM.yuv(Y, U, V)

	// YUV to RGB conversion
	//
	// YUV conversion values taken from the "SDTV with BT.470" section of:
	// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
	R := clamp(Y + (1.140 * V))
	G := clamp(Y - (0.395 * U) - (0.581 * V))
	B := clamp(Y + (2.033 * U))

	// gamma correction
	gamma := c.Gamma.Get().(float64)
	R, G, B = gammaCorrect(R, G, B, gamma)

	// create and cache
	c.secam[idx].generated = true
	c.secam[idx].col = color.RGBA{
		R: uint8(R * 255.0),
		G: uint8(G * 255.0),
		B: uint8(B * 255.0),
		A: 255,
	}

	return c.secam[idx].col
}
