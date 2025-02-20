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
	_ "embed"
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"os"

	"github.com/jetsetilly/gopher2600/resources"
)

// values taken from Stella 7.0 file common/PaletteHandler.cxx
var legacyNTSCfromStella = []uint32{
	0x000000, 0x4a4a4a, 0x6f6f6f, 0x8e8e8e, 0xaaaaaa, 0xc0c0c0, 0xd6d6d6, 0xececec,
	0x484800, 0x69690f, 0x86861d, 0xa2a22a, 0xbbbb35, 0xd2d240, 0xe8e84a, 0xfcfc54,
	0x7c2c00, 0x904811, 0xa26221, 0xb47a30, 0xc3903d, 0xd2a44a, 0xdfb755, 0xecc860,
	0x901c00, 0xa33915, 0xb55328, 0xc66c3a, 0xd5824a, 0xe39759, 0xf0aa67, 0xfcbc74,
	0x940000, 0xa71a1a, 0xb83232, 0xc84848, 0xd65c5c, 0xe46f6f, 0xf08080, 0xfc9090,
	0x840064, 0x97197a, 0xa8308f, 0xb846a2, 0xc659b3, 0xd46cc3, 0xe07cd2, 0xec8ce0,
	0x500084, 0x68199a, 0x7d30ad, 0x9246c0, 0xa459d0, 0xb56ce0, 0xc57cee, 0xd48cfc,
	0x140090, 0x331aa3, 0x4e32b5, 0x6848c6, 0x7f5cd5, 0x956fe3, 0xa980f0, 0xbc90fc,
	0x000094, 0x181aa7, 0x2d32b8, 0x4248c8, 0x545cd6, 0x656fe4, 0x7580f0, 0x8490fc,
	0x001c88, 0x183b9d, 0x2d57b0, 0x4272c2, 0x548ad2, 0x65a0e1, 0x75b5ef, 0x84c8fc,
	0x003064, 0x185080, 0x2d6d98, 0x4288b0, 0x54a0c5, 0x65b7d9, 0x75cceb, 0x84e0fc,
	0x004030, 0x18624e, 0x2d8169, 0x429e82, 0x54b899, 0x65d1ae, 0x75e7c2, 0x84fcd4,
	0x004400, 0x1a661a, 0x328432, 0x48a048, 0x5cba5c, 0x6fd26f, 0x80e880, 0x90fc90,
	0x143c00, 0x355f18, 0x527e2d, 0x6e9c42, 0x87b754, 0x9ed065, 0xb4e775, 0xc8fc84,
	0x303800, 0x505916, 0x6d762b, 0x88923e, 0xa0ab4f, 0xb7c25f, 0xccd86e, 0xe0ec7c,
	0x482c00, 0x694d14, 0x866a26, 0xa28638, 0xbb9f47, 0xd2b656, 0xe8cc63, 0xfce070,
}

// values taken from Stella 7.0 file common/PaletteHandler.cxx
//
// black levels changed to 0x000000 from 0x0b0b0b
var legacyPALfromStella = []uint32{
	0x000000, 0x333333, 0x595959, 0x7b7b7b, 0x999999, 0xb6b6b6, 0xcfcfcf, 0xe6e6e6,
	0x000000, 0x333333, 0x595959, 0x7b7b7b, 0x999999, 0xb6b6b6, 0xcfcfcf, 0xe6e6e6,
	0x3b2400, 0x664700, 0x8b7000, 0xac9200, 0xc5ae36, 0xdec85e, 0xf7e27f, 0xfff19e,
	0x004500, 0x006f00, 0x3b9200, 0x65b009, 0x85ca3d, 0xa3e364, 0xbffc84, 0xd5ffa5,
	0x590000, 0x802700, 0xa15700, 0xbc7937, 0xd6985f, 0xeeb381, 0xffce9e, 0xffdcbd,
	0x004900, 0x007200, 0x169216, 0x45af45, 0x6bc96b, 0x8be38b, 0xa9fba9, 0xc5ffc5,
	0x640012, 0x890821, 0xa73d4d, 0xc26472, 0xdc8491, 0xf4a3ae, 0xffbeca, 0xffdae0,
	0x003d29, 0x006a48, 0x048e63, 0x3caa84, 0x62c5a2, 0x83dfbe, 0xa1f8d9, 0xbeffe9,
	0x550046, 0x88006e, 0xa5318d, 0xc159aa, 0xda7cc5, 0xf39adf, 0xffb9f3, 0xffd4f6,
	0x003651, 0x005a7d, 0x117e9c, 0x429cb8, 0x68b7d2, 0x88d2eb, 0xa6ebff, 0xc3ffff,
	0x4c007c, 0x75009d, 0x932eb8, 0xaf57d2, 0xca7aeb, 0xe499ff, 0xecb7ff, 0xf3d4ff,
	0x002d83, 0x003ea4, 0x2d65bf, 0x5685da, 0x79a2f2, 0x99bfff, 0xb7dbff, 0xd3f5ff,
	0x220096, 0x5200b6, 0x7538cf, 0x945fe8, 0xb181ff, 0xc5a0ff, 0xd6bdff, 0xe8daff,
	0x00009a, 0x241db6, 0x504ad0, 0x746fe9, 0x928eff, 0xb1adff, 0xcecaff, 0xe9e5ff,
	0x000000, 0x333333, 0x595959, 0x7b7b7b, 0x999999, 0xb6b6b6, 0xcfcfcf, 0xe6e6e6,
	0x000000, 0x333333, 0x595959, 0x7b7b7b, 0x999999, 0xb6b6b6, 0xcfcfcf, 0xe6e6e6,
}

// values taken from Stella 7.0 file common/PaletteHandler.cxx
var legacySECAMfromStella = []uint32{
	0x000000, 0x2121ff, 0xf03c79, 0xff50ff, 0x7fff00, 0x7fffff, 0xffff3f, 0xffffff,
}

// the values used when the legacy colour model is enabled
type LegacyModel struct {
	Adjust Adjust
	ntsc   []color.RGBA
	pal    []color.RGBA
	secam  []color.RGBA
}

const legacyFile = "legacy.pal"

func initialiseLegacyModel(legacy *LegacyModel) error {
	clear(legacy.ntsc)
	clear(legacy.pal)
	clear(legacy.secam)

	// try loading legacy palette file first
	pth, err := resources.JoinPath(legacyFile)
	if err != nil {
		useStellaPalette(legacy)
		return fmt.Errorf("palette file: %w", err)
	}
	data, err := os.ReadFile(pth)

	if err != nil {
		useStellaPalette(legacy)

		// if ReadFile() is not about the file not existing then return the
		// error, otherwise return nil
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("palette file: %w", err)
		}
		return nil
	}

	// check validity of file
	if len(data) != 792 {
		useStellaPalette(legacy)
		return fmt.Errorf("palette file: incorrect length")
	}

	usePaletteFile(legacy, data)
	return fmt.Errorf("palette file: loaded legacy.pal")
}

func useStellaPalette(legacy *LegacyModel) {
	for _, v := range legacyNTSCfromStella {
		legacy.ntsc = append(legacy.ntsc, color.RGBA{
			R: uint8(v >> 16),
			G: uint8(v >> 8),
			B: uint8(v),
			A: 255,
		})
	}

	for _, v := range legacyPALfromStella {
		legacy.pal = append(legacy.pal, color.RGBA{
			R: uint8(v >> 16),
			G: uint8(v >> 8),
			B: uint8(v),
			A: 255,
		})
	}

	for range 16 {
		for _, v := range legacySECAMfromStella {
			legacy.secam = append(legacy.secam, color.RGBA{
				R: uint8(v >> 16),
				G: uint8(v >> 8),
				B: uint8(v),
				A: 255,
			})
		}
	}
}

func usePaletteFile(legacy *LegacyModel, data []byte) {
	for i := range 128 {
		idx := i * 3
		rgb := color.RGBA{R: data[idx], G: data[idx+1], B: data[idx+2], A: 255}
		legacy.ntsc = append(legacy.ntsc, rgb)

		idx += (128 * 3)
		rgb = color.RGBA{R: data[idx], G: data[idx+1], B: data[idx+2], A: 255}
		legacy.pal = append(legacy.pal, rgb)
	}

	for range 16 {
		for i := range 8 {
			idx := 768 + (i * 3)
			rgb := color.RGBA{R: data[idx], G: data[idx+1], B: data[idx+2], A: 255}
			legacy.secam = append(legacy.secam, rgb)
		}
	}
}
