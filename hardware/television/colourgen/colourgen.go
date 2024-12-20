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
	ntsc  []entry
	pal   []entry
	secam []entry

	dsk *prefs.Disk

	NTSCPhase prefs.Float
	PALPhase  prefs.Float

	Brightness prefs.Float
	Contrast   prefs.Float
	Saturation prefs.Float
	Hue        prefs.Float
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

	err = c.dsk.Add("television.color.ntscphase", &c.NTSCPhase)
	if err != nil {
		return nil, err
	}
	c.NTSCPhase.SetHookPost(func(_ prefs.Value) error {
		clear(c.ntsc)
		return nil
	})

	err = c.dsk.Add("television.color.palphase", &c.PALPhase)
	if err != nil {
		return nil, err
	}
	c.PALPhase.SetHookPost(func(_ prefs.Value) error {
		clear(c.pal)
		return nil
	})

	err = c.dsk.Add("television.color.brightness", &c.Brightness)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.contrast", &c.Contrast)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.saturation", &c.Saturation)
	if err != nil {
		return nil, err
	}
	err = c.dsk.Add("television.color.hue", &c.Hue)
	if err != nil {
		return nil, err
	}

	f := func(_ prefs.Value) error {
		clear(c.ntsc)
		clear(c.pal)
		clear(c.secam)
		return nil
	}

	c.Brightness.SetHookPost(f)
	c.Contrast.SetHookPost(f)
	c.Saturation.SetHookPost(f)
	c.Hue.SetHookPost(f)

	err = c.dsk.Load()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// The correct calibration of the NTSC console is somewhat controversial.
// However, there are three basic values that we can identify. In all cases
// besides the ideal case, the values should only be seen as best guesses.
const (
	// The ideal phase is what we have if we divide the colour wheel's 360°
	// equally by 15. This is a result favoured by Chris Wilkins's but it is not
	// clear if this is historically accurate
	NTSCIdealDistribution      = 24.0
	NTSCIdealDistributionLabel = "Full Range / Ideal Distribution"

	// The VideoSoft phase is what we get if we follow the instructions of the
	// VideoSoft colour bar generator. This is very possibly how many people
	// experienced NTSC colour historically
	//
	// https://www.atarimania.com/game-atari-2600-vcs-color-bar-generator-cart_11600.html
	NTSCVideoSoft     = 25.7
	NTSCVidoSoftLabel = "Video Soft Test Pattern Cartridge"

	// The Field Service phase is what we get if we follow the "VCS Domestic
	// Field Service Manual", page 3-9. The main text says that the two
	// reference colours in the diagnostic screen should be "within one shade of
	// one another".
	//
	// https://www.atarimania.com/documents/Atari_2600_2600_A_VCS_Domestic_Field_Service_Manual.pdf
	//
	// Note that the accompanying diagram says that the colours should be "the
	// same" rather than "within a shade". If we take the diagram to be
	// authoratative then we get the same result as with the VideoSoft
	// diagnostic cartridge
	//
	// However, there is an internal Atari document that describes the colours
	// that are intended for each hue. This document describes hue 15 to be
	// "light orange"
	//
	// https://ia800900.us.archive.org/30/items/Atari_2600_TIA_Technical_Manual/Atari_2600_TIA_Technical_Manual_text.pdf
	//
	// These colour descriptions also agree with the more public "Stella
	// Programmer's Guide" written by Steven Wright in 1979
	NTSCFieldService     = 26.7
	NTSCFieldSericeLabel = "Field Service Manual"
)

const PALDefault = 16.35

const (
	ntscGamma = 2.2
	palGamma  = 2.8
)

func (c *ColourGen) SetDefaults() {
	c.NTSCPhase.Set(NTSCFieldService)
	c.PALPhase.Set(PALDefault)
	c.Brightness.Set(0.0)
	c.Contrast.Set(1.0)
	c.Saturation.Set(1.0)
	c.Hue.Set(0.0)
}

// Load colour values from disk
func (c *ColourGen) Load() error {
	return c.dsk.Load()
}

// Save current colour values to disk
func (c *ColourGen) Save() error {
	return c.dsk.Save()
}

// VideoBlack is the color produced by a television in the absence of a color signal
var VideoBlack = color.RGBA{0, 0, 0, 255}

func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// GenerateNTSC creates the RGB values for the ColorSignal using the current
// colour generation preferences and for the NTSC television system
func (c *ColourGen) GenerateNTSC(col signal.ColorSignal) color.RGBA {
	// the video black signal is special and is never cached
	if col == signal.VideoBlack {
		return VideoBlack
	}

	idx := uint8(col) >> 1

	if c.ntsc[idx].generated == true {
		return c.ntsc[idx].col
	}

	// YIQ is the colour space used by the NTSC television system
	var Y, I, Q float64

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// the min/max values for the Y component
	const (
		minY = 0.35
		maxY = 1.00
	)

	// Y value in the range minY to MaxY based on the lum value
	Y = minY + (float64(lum)/8)*(maxY-minY)

	// if hue is zero then that indicates there is no colour component and
	// only the luminance is used
	if hue == 0x00 {
		if lum == 0x00 {
			// black is defined as 0% luminance, the same as for when VBLANK is enabled
			//
			// some RGB mods for the 2600 produce a non-zero black value. for
			// example, the CyberTech AV mod produces a black with a value of 0.075
			c.ntsc[idx].col = color.RGBA{A: 255}
		} else {
			Y, I, Q = c.adjustYIQ(Y, I, Q)
			y := uint8(clamp(Y) * 255)
			c.ntsc[idx].col = color.RGBA{R: y, G: y, B: y, A: 255}
		}
		c.ntsc[idx].generated = true
		return c.ntsc[idx].col
	}

	// the colour component indicates a point on the 'colour wheel'
	phiHue := (float64(hue) - 1) * -c.NTSCPhase.Get().(float64)

	// angle of the colour burst reference is 180 by defintion
	const phiBurst = 180

	// however, from the "Stella Programmer's Guide" (page 28):
	//
	// "Binary code 0 selects no color. Code 1 selects gold (same phase as
	// color burst)"
	//
	// this means that hue 1 must be "gold" rather than "green"
	//
	// what "gold" means is subjective but none-the-less, we must adjust the hue
	// so that hue 1 is more gold than green
	//
	// the current value was arrived at from study of the "Stella Programmer's
	// Guide" (page 18):
	//
	// "A hardware counter on this chip produces all horizontal timing (such as
	// sync, blank, burst) independent of the microprocessor, This counter is
	// driven from an external 3.58 Mhz oscillator and has a total count of 228.
	// Blank is decoded as 68 counts and sync and color burst as 16 counts."
	//
	// so 16 multipled by 3.58 is 57.28. the negative of this value seems
	// correct because we know that hue 1 must be in the yellow region
	//
	// I think this is what the test in the programmer's guide is saying but I'm
	// not sure. none-the-less, the results seem accurate and there is at least
	// some rationale for the value
	const phiAdj = -57.28
	phiHue += phiAdj

	// the final angle is the angle of the calculated hue plus the adjusted color burst
	phi := phiHue + phiBurst

	// phi has been calculated in degrees but the math functions require radians
	phi *= math.Pi / 180

	// saturation of chroma in final colour. value currently uncertain
	const saturation = 0.3

	// the chroma values are scaled by the luminance value
	I = Y * saturation * math.Sin(phi)
	Q = Y * saturation * math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YIQ
	Y, I, Q = c.adjustYIQ(Y, I, Q)

	// YIQ to RGB conversion
	//
	// YIQ conversion values taken from the "NTSC 1953 colorimetry" section
	// of: https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306
	R := clamp(Y + (0.956 * I) + (0.619 * Q))
	G := clamp(Y - (0.272 * I) - (0.647 * Q))
	B := clamp(Y - (1.106 * I) + (1.703 * Q))

	// gamma correction
	R = math.Pow(R, ntscGamma)
	G = math.Pow(G, ntscGamma)
	B = math.Pow(B, ntscGamma)

	// from the "FCC NTSC Standard (SMPTE C)" of the same wikipedia article
	// 		R := clamp(Y + (0.9469 * I) + (0.6236 * Q))
	// 		G := clamp(Y - (0.2748 * I) - (0.6357 * Q))
	// 		B := clamp(Y - (1.1 * I) + (1.7 * Q))

	// the coefficients used by Stella (7.0)
	// 	R := clamp(Y + (0.9563 * I) + (0.6210 * Q))
	// 	G := clamp(Y - (0.2721 * I) - (0.6474 * Q))
	// 	B := clamp(Y - (1.1070 * I) + (1.7046 * Q))

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
	// the video black signal is special and is never cached
	if col == signal.VideoBlack {
		return VideoBlack
	}

	idx := uint8(col) >> 1

	if c.pal[idx].generated == true {
		return c.pal[idx].col
	}

	// YUV is the colour space used by the PAL television system
	var Y, U, V float64

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// the min/max values for the Y component
	const (
		minY = 0.35
		maxY = 1.00
	)

	// Y value in the range minY to MaxY based on the lum value
	Y = minY + (float64(lum)/8)*(maxY-minY)

	// PAL creates a grayscale for hues 0, 1, 14 and 15
	if hue <= 0x01 || hue >= 0x0e {
		if lum == 0x00 {
			// black is defined as 0% luminance, the same as for when VBLANK is enabled
			//
			// some RGB mods for the 2600 produce a non-zero black value. for
			// example, the CyberTech AV mod produces a black with a value of 0.075
			c.ntsc[idx].col = color.RGBA{A: 255}
		} else {
			Y, U, V = c.adjustYUV(Y, U, V)
			y := uint8(clamp(Y) * 255)
			c.ntsc[idx].col = color.RGBA{R: y, G: y, B: y, A: 255}
		}
		c.ntsc[idx].generated = true
		return c.ntsc[idx].col
	}

	var phiHue float64

	// even-numbered hue numbers go in the opposite direction for some reason
	if hue&0x01 == 0x01 {
		// green to lilac
		phiHue = float64(hue) * -c.PALPhase.Get().(float64)
	} else {
		// gold to purple
		phiHue = (float64(hue) - 2) * c.PALPhase.Get().(float64)
	}

	// angle of the colour burst reference is 180 by defintion
	const phiBurst = 180

	// see comments in generateNTSC for why we apply the adjusment and burst value to the
	// calculated phi
	const phiAdj = -57.28
	phiHue += phiAdj
	phi := phiHue + phiBurst

	// phi has been calculated in degrees but the math functions require radians
	phi *= math.Pi / 180

	// saturation of chroma in final colour. value currently uncertain
	const saturation = 0.3

	// create UV from hue
	U = Y * saturation * -math.Sin(phi)
	V = Y * saturation * -math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YUV
	Y, U, V = c.adjustYUV(Y, U, V)

	// YUV to RGB conversion
	//
	// YUV conversion values taken from the "SDTV with BT.470" section of:
	// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
	R := clamp(Y + (1.140 * V))
	G := clamp(Y - (0.395 * U) - (0.581 * V))
	B := clamp(Y + (2.033 * U))

	// gamma correction
	R = math.Pow(R, palGamma)
	G = math.Pow(G, palGamma)
	B = math.Pow(B, palGamma)

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
	// the video black signal is special and is never cached
	if col == signal.VideoBlack {
		return VideoBlack
	}

	idx := uint8(col) >> 1

	if c.secam[idx].generated == true {
		return c.secam[idx].col
	}

	// SECAM uses the YUV colour space but in a different way to PAL
	var Y, U, V float64

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1

	// the hue nibble of the signal.ColourSignal value is ignored by SECAM
	// consoles

	// special treatment of a lum value of zero
	if lum == 0 {
		c.secam[idx].col = color.RGBA{A: 255}
		c.secam[idx].generated = true
		return c.secam[idx].col
	}

	// the min/max values for the Y component is different for SECAM when
	// compared to NTSCL and PAL
	const (
		minY = 0.60
		maxY = 1.00
	)

	// Y value in the range minY to MaxY based on the lum value
	Y = minY + (float64(lum)/8)*(maxY-minY)

	var phi float64
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
	case 7:
		Y, U, V = c.adjustYUV(Y, U, V)
		y := uint8(clamp(Y) * 255)
		c.secam[idx].col = color.RGBA{R: y, G: y, B: y, A: 255}
		return c.secam[idx].col
	}

	// saturation of chroma in final colour. value currently uncertain
	const saturation = 0.3

	// create UV from hue
	U = Y * saturation * -math.Sin(phi)
	V = Y * saturation * -math.Cos(phi)

	// apply brightness/constrast/saturation/hue settings to YUV
	Y, U, V = c.adjustYUV(Y, U, V)

	// YUV to RGB conversion
	//
	// YUV conversion values taken from the "SDTV with BT.470" section of:
	// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
	R := clamp(Y + (1.140 * V))
	G := clamp(Y - (0.395 * U) - (0.581 * V))
	B := clamp(Y + (2.033 * U))

	// gamma correction
	R = math.Pow(R, palGamma)
	G = math.Pow(G, palGamma)
	B = math.Pow(B, palGamma)

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
