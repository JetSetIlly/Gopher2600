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

	"github.com/jetsetilly/gopher2600/hardware/clocks"
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

	LegacyEnabled prefs.Bool
	LegacyAdjust  Adjust
	Adjust        Adjust

	// gamma is the same for both legacy and non-legacy models
	Gamma prefs.Float
}

// NewColourGen is the preferred method of intialisation for the ColourGen type.
func NewColourGen() (*ColourGen, error) {
	c := &ColourGen{
		ntsc:  make([]entry, 128),
		pal:   make([]entry, 128),
		secam: make([]entry, 128),
	}

	c.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	c.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = c.dsk.Add("television.color.legacy.enabled", &c.LegacyEnabled)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.legacy.brightness", &c.LegacyAdjust.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.legacy.contrast", &c.LegacyAdjust.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.legacy.saturation", &c.LegacyAdjust.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.legacy.hue", &c.LegacyAdjust.Hue)
	if err != nil {
		return nil, err
	}

	err = c.dsk.Add("television.color.brightness", &c.Adjust.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.contrast", &c.Adjust.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.saturation", &c.Adjust.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.hue", &c.Adjust.Hue)
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

	c.LegacyEnabled.SetHookPost(f)
	c.LegacyAdjust.Brightness.SetHookPost(f)
	c.LegacyAdjust.Contrast.SetHookPost(f)
	c.LegacyAdjust.Saturation.SetHookPost(f)
	c.LegacyAdjust.Hue.SetHookPost(f)

	c.Adjust.Brightness.SetHookPost(f)
	c.Adjust.Contrast.SetHookPost(f)
	c.Adjust.Saturation.SetHookPost(f)
	c.Adjust.Hue.SetHookPost(f)

	err = c.dsk.Add("television.color.legacy.ntscphase", &c.LegacyAdjust.NTSCPhase)
	if err != nil {
		return nil, err
	}
	c.LegacyAdjust.NTSCPhase.SetHookPost(func(_ prefs.Value) error {
		clear(c.ntsc)
		c.zeroBlack.generated = false
		return nil
	})

	err = c.dsk.Add("television.color.legacy.palphase", &c.LegacyAdjust.PALPhase)
	if err != nil {
		return nil, err
	}
	c.LegacyAdjust.PALPhase.SetHookPost(func(_ prefs.Value) error {
		clear(c.pal)
		c.zeroBlack.generated = false
		return nil
	})

	err = c.dsk.Add("television.color.ntscphase", &c.Adjust.NTSCPhase)
	if err != nil {
		return nil, err
	}
	c.Adjust.NTSCPhase.SetHookPost(func(_ prefs.Value) error {
		clear(c.ntsc)
		c.zeroBlack.generated = false
		return nil
	})

	err = c.dsk.Add("television.color.palphase", &c.Adjust.PALPhase)
	if err != nil {
		return nil, err
	}
	c.Adjust.PALPhase.SetHookPost(func(_ prefs.Value) error {
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

func (c *ColourGen) SetDefaults() {
	c.LegacyEnabled.Set(true)
	c.LegacyAdjust.Brightness.Set(1.196)
	c.LegacyAdjust.Contrast.Set(1.000)
	c.LegacyAdjust.Saturation.Set(0.963)
	c.LegacyAdjust.Hue.Set(0.0)
	c.LegacyAdjust.NTSCPhase.Set(0.0)
	c.LegacyAdjust.PALPhase.Set(0.0)

	c.Adjust.Brightness.Set(0.949)
	c.Adjust.Contrast.Set(1.205)
	c.Adjust.Saturation.Set(0.874)
	c.Adjust.Hue.Set(0.0)
	c.Adjust.NTSCPhase.Set(NTSCFieldService)
	c.Adjust.PALPhase.Set(PALDefault)

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

func (c *ColourGen) generateZeroBlack() color.RGBA {
	if c.zeroBlack.generated {
		return c.zeroBlack.col
	}

	legacy := c.LegacyEnabled.Get().(bool)

	var Y float64

	if legacy {
		Y, _, _ = c.LegacyAdjust.yiq(videoBlack, 0, 0)
	} else {
		Y, _, _ = c.Adjust.yiq(videoBlack, 0, 0)
	}

	y := uint8(clamp(Y) * 255)
	c.zeroBlack.col = color.RGBA{R: y, G: y, B: y, A: 255}
	c.zeroBlack.generated = true

	if legacy {
		gamma := c.Gamma.Get().(float64)
		c.zeroBlack.col = gammaCorrectRGB(c.zeroBlack.col, gamma)
	}

	return c.zeroBlack.col
}

// the min/max values for the Y component
// used by the new colour model to generate the luminosity range for hues 1 to
// 15 for NTSC, PAL and SECAM
const (
	minY = 0.40
	maxY = 1.00
)

// GenerateNTSC creates the RGB values for the ColorSignal using the current
// colour generation preferences and for the NTSC television system
func (c *ColourGen) GenerateNTSC(col signal.ColorSignal) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack()
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
		c.ntsc[idx].col = c.generateZeroBlack()
		c.ntsc[idx].generated = true
		return c.ntsc[idx].col
	}

	// Y, phi and saturation is all that's needed to create the RGB value
	var Y, phi, saturation float64

	if c.LegacyEnabled.Get().(bool) {
		Y = legacyNTSC_yiq[hue][lum].y
		phi = legacyNTSC_yiq[hue][lum].phi
		saturation = legacyNTSC_yiq[hue][lum].saturation

		// stretch phi by a fraction of the phase setting
		phi -= (float64(hue - 1)) * c.LegacyAdjust.NTSCPhase.Get().(float64) * math.Pi / 180
	} else {
		// NTSC creates a grayscale for hue 0
		if hue == 0x00 {
			saturation = 0.0
		} else {
			saturation = 0.3
		}

		// Y value in the range minY to MaxY based on the lum value
		Y = minY + (float64(lum)/8)*(maxY-minY)

		// the colour component indicates a point on the 'colour wheel'
		phi = (float64(hue - 1)) * -c.Adjust.NTSCPhase.Get().(float64)

		// angle of the colour burst reference is 180 by defintion
		const phiBurst = 180
		phi += phiBurst

		// however, from the "Stella Programmer's Guide" (page 28):
		//
		// "Binary code 0 selects no color. Code 1 selects gold (same phase as
		// color burst)"
		//
		// what "gold" means is subjective. indeed the JAN programming guide says it
		// is light orange. but whatever the precise colour, we can say that hue 1
		// should be more in the orange section of the colour wheel than the green
		// section, which it would be if we left it at 180°
		//
		// from page 28 of the "Stella Programmer's Guide":
		//
		// "A hardware counter on this chip produces all horizontal timing (such as
		// sync, blank, burst) independent of the microprocessor, This counter is
		// driven from an external 3.58 Mhz oscillator and has a total count of 228.
		// Blank is decoded as 68 counts and sync and color burst as 16 counts."
		//
		// (NOTE: I really don't know if the following is correct. It produces
		// pleasing results but I can't really justify the logic behind it. either
		// way it doesn't really matter - the end result is for hue 1 to be "gold"
		// or "light orange" so even if the colour being created here isn't the
		// 'natural' colour burst value the user will be adjusting the TV's hue so
		// that it is gold/orange. all we're doing is making the zero hue adjustment
		// the 'correct' value)
		//
		// using the values on page 28 of the guide: 16 multipled by 3.58 is 57.28.
		// if we subtract this from 180 we get an angle that is in the gold/orange
		// section of the colour wheel
		const phiAdjBurst = -(clocks.NTSC_TIA * 16)
		phi += phiAdjBurst

		// phi has been calculated in degrees but the math functions require radians
		phi *= math.Pi / 180
	}

	// (IQ used to by multplied by the luminance (Y) value but I no longer
	// believe this is correct)
	I := saturation * math.Sin(phi)
	Q := saturation * math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YIQ
	if c.LegacyEnabled.Get().(bool) {
		Y, I, Q = c.LegacyAdjust.yiq(Y, I, Q)
	} else {
		Y, I, Q = c.Adjust.yiq(Y, I, Q)
	}

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
func (c *ColourGen) GeneratePAL(col signal.ColorSignal) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack()
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
		c.pal[idx].col = c.generateZeroBlack()
		c.pal[idx].generated = true
		return c.pal[idx].col
	}

	// Y, phi and saturation is all that's needed to create the RGB value
	var Y, phi, saturation float64

	if c.LegacyEnabled.Get().(bool) {
		Y = legacyPAL_yuv[hue][lum].y
		phi = legacyPAL_yuv[hue][lum].phi
		saturation = legacyPAL_yuv[hue][lum].saturation

		// stretch phi by a fraction of the phase setting
		if hue&0x01 == 0x01 {
			phi -= float64(hue) * c.LegacyAdjust.PALPhase.Get().(float64) * math.Pi / 180
		} else {
			phi += (float64(hue) - 2) * c.LegacyAdjust.PALPhase.Get().(float64) * math.Pi / 180
		}
	} else {
		// PAL creates a grayscale for hues 0, 1, 14 and 15
		if hue <= 0x01 || hue >= 0x0e {
			saturation = 0.0
		} else {
			saturation = 0.3
		}

		// Y value in the range minY to MaxY based on the lum value
		Y = minY + (float64(lum)/8)*(maxY-minY)

		// even-numbered hue numbers go in the opposite direction for some reason
		if hue&0x01 == 0x01 {
			// green to lilac
			phi = float64(hue) * -c.Adjust.PALPhase.Get().(float64)
		} else {
			// gold to purple
			phi = (float64(hue) - 2) * c.Adjust.PALPhase.Get().(float64)
		}

		// angle of the colour burst reference is 180 by defintion
		const phiBurst = 180
		phi += phiBurst

		// see comments in generateNTSC for/how why we apply the adjustment and
		// burst value to the calculated phi. we use the PAL clock rather than the
		// NTSC clock of course. and rather than hue 1 being gold/orange it is hue 2
		// that must be that colour
		const phiAdjBurst = -(clocks.PAL_TIA * 16)
		phi += phiAdjBurst

		// phi has been calculated in degrees but the math functions require radians
		phi *= math.Pi / 180
	}

	// (UV used to by multplied by the luminance (Y) value but I no longer
	// believe this is correct)
	U := saturation * -math.Sin(phi)
	V := saturation * -math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YUV
	if c.LegacyEnabled.Get().(bool) {
		Y, U, V = c.LegacyAdjust.yuv(Y, U, V)
	} else {
		Y, V, V = c.Adjust.yuv(Y, U, V)
	}

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

func (c *ColourGen) GenerateSECAM(col signal.ColorSignal) color.RGBA {
	if col == signal.ZeroBlack {
		return c.generateZeroBlack()
	}

	idx := uint8(col) >> 1

	if c.secam[idx].generated == true {
		return c.secam[idx].col
	}

	// the hue nibble of the signal.ColourSignal value is ignored by SECAM
	// consoles
	lum := (col & 0x0e) >> 1

	// special case for luminance of zero
	if lum == 0x00 {
		c.pal[idx].col = c.generateZeroBlack()
		c.pal[idx].generated = true
		return c.pal[idx].col
	}

	// Y, phi and saturation is all that's needed to create the RGB value
	var Y, phi, saturation float64

	if c.LegacyEnabled.Get().(bool) {
		Y = legacySECAM_yuv[lum].y
		phi = legacySECAM_yuv[lum].phi
		saturation = legacySECAM_yuv[lum].saturation
	} else {
		// SECAM lum 7 is completely desaturated
		if lum == 7 {
			saturation = 0.0
		} else {
			saturation = 0.3
		}

		// Y value in the range minY to MaxY based on the lum value
		Y = minY + (float64(lum)/8)*(maxY-minY)

		switch lum {
		case 1:
			phi = 225
		case 2:
			phi = 135
		case 3:
			phi = 180
		case 4:
			phi = 45
		case 5:
			phi = 270
		case 6:
			phi = 90
		}
	}

	// (UV used to by multplied by the luminance (Y) value but I no longer
	// believe this is correct)
	U := saturation * -math.Sin(phi)
	V := saturation * -math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YUV
	if c.LegacyEnabled.Get().(bool) {
		Y, U, V = c.LegacyAdjust.yuv(Y, U, V)
	} else {
		Y, V, V = c.Adjust.yuv(Y, U, V)
	}

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
