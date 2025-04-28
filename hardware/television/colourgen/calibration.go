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
	//
	// This phase value is supported by the information on page 38 of the "JAN_programming_guide"
	//
	// Mathematically this is the same as dividing 360° by 14. This makes sense
	// because hue-1 and hue-15 have the same colour description and are thus in
	// the same location on the colour wheel.
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

// Unlike NTSC the default phase for PAL seems to be less contentious. A single
// value acts as the default preset
const PALDefault = NTSCFieldService / 2.0

// The gamma value assumed by all colour conversion
const Gamma = 2.2
