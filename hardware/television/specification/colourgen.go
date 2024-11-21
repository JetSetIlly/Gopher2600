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

package specification

import (
	"image/color"
	"math"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// VideoBlack is the color produced by a television in the absence of a color signal
var VideoBlack = color.RGBA{0, 0, 0, 255}

// gamma values taken from the "Standard Gammas" section of:
// https://en.wikipedia.org/w/index.php?title=Gamma_correction&oldid=1253179068
const (
	ntscGamma = 2.2
	palGamma  = 2.8
)

var NTSCPhase float64

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

func init() {
	NTSCPhase = NTSCFieldService
}

func generateNTSC(col signal.ColorSignal) color.RGBA {
	if col == signal.VideoBlack {
		return VideoBlack
	}

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// the min/max values for the Y component of greyscale hues
	const (
		minY = 0.35
		maxY = 1.00
	)

	// Y value in the range minY to MaxY based on the lum value
	Y := minY + (float64(lum)/8)*(maxY-minY)

	// if hue is zero then that indicates there is no colour component and
	// only the luminance is used
	if hue == 0x00 {
		if lum == 0x00 {
			// black is defined as 0% luminance, the same as for when VBLANK is
			// enabled
			//
			// some RGB mods for the 2600 produce a non-zero black value. for
			// example, the CyberTech AV mod produces a black with a value of 0.075
			return color.RGBA{A: 255}
		}
		g := uint8(Y * 255)
		return color.RGBA{R: g, G: g, B: g, A: 255}
	}

	// the colour component indicates a point on the 'colour wheel'
	phiHue := (float64(hue) - 1) * -NTSCPhase

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
	I := Y * saturation * math.Sin(phi)
	Q := Y * saturation * math.Cos(phi)

	// YIQ to RGB conversion
	//
	// YIQ conversion values taken from the "NTSC 1953 colorimetry" section
	// of: https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306
	R := clamp(Y + (0.956 * I) + (0.619 * Q))
	G := clamp(Y - (0.272 * I) - (0.647 * Q))
	B := clamp(Y - (1.106 * I) + (1.703 * Q))

	// from the "FCC NTSC Standard (SMPTE C)" of the same wikipedia article
	// 		R := clamp(Y + (0.9469 * I) + (0.6236 * Q))
	// 		G := clamp(Y - (0.2748 * I) - (0.6357 * Q))
	// 		B := clamp(Y - (1.1 * I) + (1.7 * Q))

	// the coefficients used by Stella (7.0)
	// 	R := clamp(Y + (0.9563 * I) + (0.6210 * Q))
	// 	G := clamp(Y - (0.2721 * I) - (0.6474 * Q))
	// 	B := clamp(Y - (1.1070 * I) + (1.7046 * Q))

	return color.RGBA{
		R: uint8(R * 255.0),
		G: uint8(G * 255.0),
		B: uint8(B * 255.0),
		A: 255,
	}
}

var PALPhase float64

func init() {
	PALPhase = 16.35
}

func generatePAL(col signal.ColorSignal) color.RGBA {
	if col == signal.VideoBlack {
		return VideoBlack
	}

	// color-luminance components of color signal
	lum := (col & 0x0e) >> 1
	hue := (col & 0xf0) >> 4

	// the min/max values for the Y component of greyscale hues
	const (
		minY = 0.35
		maxY = 1.00
	)

	// Y value in the range minY to MaxY based on the lum value
	Y := minY + (float64(lum)/8)*(maxY-minY)

	// PAL creates a grayscale for hues 0, 1, 14 and 15
	if hue <= 0x01 || hue >= 0x0e {
		if lum == 0x00 {
			// black is defined as 0% luminance, the same as for when VBLANK is
			// enabled
			//
			// some RGB mods for the 2600 produce a non-zero black value. for
			// example, the CyberTech AV mod produces a black with a value of 0.075
			return color.RGBA{A: 255}
		}
		g := uint8(Y * 255)
		return color.RGBA{R: g, G: g, B: g, A: 255}
	}

	var phiHue float64

	// even-numbered hue numbers go in the opposite direction for some reason
	if hue&0x01 == 0x01 {
		// green to lilac
		phiHue = float64(hue) * -PALPhase
	} else {
		// gold to purple
		phiHue = (float64(hue) - 2) * PALPhase
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
	U := Y * saturation * -math.Sin(phi)
	V := Y * saturation * -math.Cos(phi)

	// YUV to RGB conversion
	//
	// YUV conversion values taken from the "SDTV with BT.470" section of:
	// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
	R := clamp(Y + (1.140 * V))
	G := clamp(Y - (0.395 * U) - (0.581 * V))
	B := clamp(Y + (2.033 * U))

	return color.RGBA{
		R: uint8(R * 255.0),
		G: uint8(G * 255.0),
		B: uint8(B * 255.0),
		A: 255,
	}
}

func generateSECAM(col signal.ColorSignal) color.RGBA {
	if col == signal.VideoBlack {
		return VideoBlack
	}

	// only the luminance data of the colour signal is used
	lum := (col & 0x0e) >> 1

	// the luminance is actually fixed (Y = 1.0) and is used to create a U and V value
	// rather than calculate the U and V we just looked up the RGB value directly
	var secam = []uint32{0x000000, 0x2121ff, 0xf03c79, 0xff50ff, 0x7fff00, 0x7fffff, 0xffff3f, 0xffffff}
	v := secam[lum]

	return color.RGBA{
		R: uint8((v & 0xff0000) >> 16),
		G: uint8((v & 0xff00) >> 8),
		B: uint8(v & 0xff),
		A: 255,
	}
}
