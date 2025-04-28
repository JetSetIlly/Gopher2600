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
	"math"
)

type component struct {
	y          float64
	phi        float64 // in radians
	saturation float64
}

var legacyNTSC_yiq [16][8]component
var legacyPAL_yuv [16][8]component
var legacySECAM_yuv [8]component

func init() {
	for i, rgb := range legacyNTSCfromStella {
		hue, lum := i/8, i%8

		R := float64((rgb>>16)&0xff) / 255
		G := float64((rgb>>8)&0xff) / 255
		B := float64(rgb&0xff) / 255

		// RGB to YIQ conversion
		//
		// YIQ conversion values taken from the "NTSC 1953 colorimetry" section
		// of: https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306
		Y := 0.299*R + 0.587*G + 0.114*B
		I := 0.5959*R - 0.2746*G - 0.3213*B
		Q := 0.2115*R - 0.5227*G + 0.3112*B

		phi := math.Atan2(I, Q)
		sat := math.Sqrt(math.Pow(I, 2) + math.Pow(Q, 2))

		legacyNTSC_yiq[hue][lum] = component{
			y:          Y,
			phi:        phi,
			saturation: sat,
		}
	}

	for i, rgb := range legacyPALfromStella {
		hue, lum := i/8, i%8

		R := float64((rgb>>16)&0xff) / 255
		G := float64((rgb>>8)&0xff) / 255
		B := float64(rgb&0xff) / 255

		// RGB to YUV conversion
		//
		// YUV conversion values taken from the "SDTV with BT.470" section of:
		// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
		Y := 0.299*R + 0.587*G + 0.114*B
		U := -0.14713*R - 0.28886*G + 0.436*B
		V := 0.615*R - 0.51499*G - 0.10001*B

		phi := math.Atan2(-U, -V)
		sat := math.Sqrt(math.Pow(U, 2) + math.Pow(V, 2))

		legacyPAL_yuv[hue][lum] = component{
			y:          Y,
			phi:        phi,
			saturation: sat,
		}
	}

	for lum, rgb := range legacySECAMfromStella {
		R := float64((rgb>>16)&0xff) / 255
		G := float64((rgb>>8)&0xff) / 255
		B := float64(rgb&0xff) / 255

		// RGB to YUV conversion
		//
		// YUV conversion values taken from the "SDTV with BT.470" section of:
		// https://en.wikipedia.org/w/index.php?title=Y%E2%80%B2UV&oldid=1249546174
		Y := 0.299*R + 0.587*G + 0.114*B
		U := -0.14713*R - 0.28886*G + 0.436*B
		V := 0.615*R - 0.51499*G - 0.10001*B

		phi := math.Atan2(-U, -V)
		sat := math.Sqrt(math.Pow(U, 2) + math.Pow(V, 2))

		legacySECAM_yuv[lum] = component{
			y:          Y,
			phi:        phi,
			saturation: sat,
		}
	}
}
