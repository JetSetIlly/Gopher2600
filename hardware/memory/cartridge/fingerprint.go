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

package cartridge

import (
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/logger"
)

func fingerprint3e(b []byte) bool {
	// 3E cart bankswitching is triggered by storing the bank number in address
	// 3E using 'STA $3E', commonly followed by an  immediate mode LDA
	//
	// fingerprint method taken from:
	//
	// https://gitlab.com/firmaplus/atari-2600-pluscart/-/blob/master/source/STM32firmware/PlusCart/Src/cartridge_detection.c#L140

	for i := 0; i < len(b)-3; i++ {
		if b[i] == 0x85 && b[i+1] == 0x3e && b[i+2] == 0xa9 && b[i+3] == 0x00 {
			return true
		}
	}

	return false
}

func fingerprint3ePlus(b []byte) bool {
	// previous versions of this function worked similarly to the tigervision
	// method but this is more accurate
	//
	// fingerprint method taken from:
	//
	// https://gitlab.com/firmaplus/atari-2600-pluscart/-/blob/master/source/STM32firmware/PlusCart/Src/cartridge_detection.c#L148
	for i := 0; i < len(b)-3; i++ {
		if b[i] == 'T' && b[i+1] == 'J' && b[i+2] == '3' && b[i+3] == 'E' {
			return true
		}
	}

	return false
}

func fingerprintMnetwork(b []byte) bool {
	// Bump 'n' Jump is the fussiest mnetwork cartridge I've found. Matching
	// hotspots:
	//
	//	$fdd5 LDA      BANK5
	//	$fde3 LDA      BANK6
	//	$fe0e LDA      BANK5
	//	$fe16 LDA      BANK4
	//
	// This also catches modern games not created by mnetwork, eg Pitkat

	threshold := 4
	for i := 0; i < len(b)-3; i++ {
		if b[i] == 0xad && b[i+2] == 0xff && (b[i+1] == 0xe4 || b[i+1] == 0xe5 || b[i+1] == 0xe6) {
			threshold--
		}
		if threshold == 0 {
			return true
		}
	}

	return false
}

func fingerprintParkerBros(b []byte) bool {
	// fingerprint patterns taken from Stella CartDetector.cxx
	for i := 0; i <= len(b)-3; i++ {
		if (b[i] == 0x8d && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0x8d && b[i+1] == 0xe0 && b[i+2] == 0x5f) ||
			(b[i] == 0x8d && b[i+1] == 0xe9 && b[i+2] == 0xff) ||
			(b[i] == 0x0c && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0xad && b[i+1] == 0xe0 && b[i+2] == 0x1f) ||
			(b[i] == 0xad && b[i+1] == 0xe9 && b[i+2] == 0xff) ||
			(b[i] == 0xad && b[i+1] == 0xed && b[i+2] == 0xff) ||
			(b[i] == 0xad && b[i+1] == 0xf3 && b[i+2] == 0xbf) {
			return true
		}
	}

	return false
}

func fingerprintDF(b []byte) bool {
	if len(b) < 0xffb {
		return false
	}
	return b[0xff8] == 'D' && b[0xff9] == 'F' && b[0xffa] == 'S' && b[0xffb] == 'C'
}

func fingerprintHarmony(b []byte) bool {
	if len(b) < 0x23 {
		return false
	}
	return b[0x20] == 0x1e && b[0x21] == 0xab && b[0x22] == 0xad && b[0x23] == 0x10
}

func fingerprintSuperchargerFastLoad(cartload cartridgeloader.Loader) bool {
	l := len(cartload.Data)
	return l == 8448 || l == 25344 || l == 33792
}

func fingerprintTigervision(b []byte) bool {
	// tigervision cartridges change banks by writing to memory address 0x3f. we
	// can hypothesise that these types of cartridges will have that instruction
	// sequence "85 3f" many times in a ROM whereas other cartridge types will not

	threshold := 5
	for i := 0; i < len(b)-1; i++ {
		if b[i] == 0x85 && b[i+1] == 0x3f {
			threshold--
		}
		if threshold == 0 {
			return true
		}
	}
	return false
}

func fingerprint8k(data []byte) func([]byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintParkerBros(data) {
		return newParkerBros
	}

	return newAtari8k
}

func fingerprint16k(data []byte) func([]byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	return newAtari16k
}

func fingerprint32k(data []byte) func([]byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	return newAtari32k
}

func fingerprint128k(data []byte) func([]byte) (mapper.CartMapper, error) {
	if fingerprintDF(data) {
		return newDF
	}

	logger.Log("fingerprint", "not confident that this is DF file")
	return newDF
}

func (cart *Cartridge) fingerprint(cartload cartridgeloader.Loader) error {
	var err error

	if fingerprintHarmony(cartload.Data) {
		// !!TODO: this might be a CFDJ cartridge. check for that.
		cart.mapper, err = harmony.NewDPCplus(cartload.Data)
		return err
	}

	if fingerprintSuperchargerFastLoad(cartload) {
		cart.mapper, err = supercharger.NewSupercharger(cartload)
		return err
	}

	if fingerprint3e(cartload.Data) {
		cart.mapper, err = new3e(cartload.Data)
		return err
	}

	if fingerprint3ePlus(cartload.Data) {
		cart.mapper, err = new3ePlus(cartload.Data)
		return err
	}

	switch len(cartload.Data) {
	case 2048:
		cart.mapper, err = newAtari2k(cartload.Data)
		if err != nil {
			return err
		}

	case 4096:
		cart.mapper, err = newAtari4k(cartload.Data)
		if err != nil {
			return err
		}

	case 8192:
		cart.mapper, err = fingerprint8k(cartload.Data)(cartload.Data)
		if err != nil {
			return err
		}

	case 10240:
		fallthrough

	case 10495:
		cart.mapper, err = newDPC(cartload.Data)
		if err != nil {
			return err
		}

	case 12288:
		cart.mapper, err = newCBS(cartload.Data)
		if err != nil {
			return err
		}

	case 16384:
		cart.mapper, err = fingerprint16k(cartload.Data)(cartload.Data)
		if err != nil {
			return err
		}

	case 32768:
		cart.mapper, err = fingerprint32k(cartload.Data)(cartload.Data)
		if err != nil {
			return err
		}

	case 65536:
		return curated.Errorf("65536 bytes not yet supported")

	case 131072:
		cart.mapper, err = fingerprint128k(cartload.Data)(cartload.Data)
		if err != nil {
			return err
		}

	default:
		return curated.Errorf("unrecognised size (%d bytes)", len(cartload.Data))
	}

	// if cartridge mapper implements the optionalSuperChip interface then try
	// to add the additional RAM
	if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
		superchip.AddSuperchip()
	}

	return nil
}

// fingerprinting a PlusROM cartridge is slightly different to the main
// fingerprint() function above. the fingerprintPlusROM() function below is the
// first step. it checks for the byte sequence 8d f1 x1, which is the
// equivalent to STA $xff1, a necessary instruction in a PlusROM cartridge
//
// if this sequence is found then the function returns true, whereupon
// plusrom.NewPlusROM() can be called. the seoncd part of the fingerprinting
// process occurs in that function. if that fails then we can say that the true
// result from this function was a false positive.
func (cart *Cartridge) fingerprintPlusROM(cartload cartridgeloader.Loader) bool {
	for i := 0; i < len(cartload.Data)-2; i++ {
		if cartload.Data[i] == 0x8d && cartload.Data[i+1] == 0xf1 && (cartload.Data[i+2]&0x10) == 0x10 {
			return true
		}
	}
	return false
}
