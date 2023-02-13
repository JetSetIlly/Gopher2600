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
	"bytes"
	"fmt"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/ace"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/dpcplus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/elf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
)

// if anwhere parameter is true then the ELF magic number can appear anywhere
// in the data (the b parameter). otherwise it must appear at the beginning of
// the data
func fingerprintElf(b []byte, anywhere bool) bool {
	if anywhere {
		if bytes.Contains(b, []byte{0x7f, 'E', 'L', 'F'}) {
			return true
		}
	} else if bytes.HasPrefix(b, []byte{0x7f, 'E', 'L', 'F'}) {
		return true
	}

	return false
}

func fingerprintAce(b []byte) (bool, string, bool) {
	if len(b) < 144 {
		return false, "", false
	}

	// some ACE files embed an ELF file inside the ACE data. these files are
	// identified by the presence of "elf-relocatable" in the data premable
	wrappedELF := bytes.Contains(b[:144], []byte("elf-relocatable"))

	// make double sure this is actually an elf file. otherwise it's just an
	// ACE file with elf-relocatable in the data preamble
	wrappedELF = wrappedELF && fingerprintElf(b, true)

	if bytes.Contains(b[:144], []byte("ACE-2600")) {
		return true, "ACE-2600", wrappedELF
	}

	if bytes.Contains(b[:144], []byte("ACE-PC00")) {
		return true, "ACE-PC00", wrappedELF
	}

	return false, "", false
}

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
	// This also catches modern games not created by mnetwork but which use the
	// format, eg Pitkat
	//
	// (24/01/21)
	//
	// Splendidnut's Congobongo demo is even fussier than Bump 'n' Jump. While
	// I expect the ROM to get more complex we should support the demo ROM if
	// only because it exists.
	//
	// notably, it uses a non-primary-mirror cartridge address and there are
	// only two bankswitch instructions in the data (although I didn't search
	// it exhaustively - just LDA instructions).
	//
	// threshold has been reduced to two.
	//
	// (19/06/21)
	//
	// Thomas Jentzch's Elite demo requires a threshold of one
	//
	// https://atariage.com/forums/topic/155657-elite-3d-graphics/?do=findComment&comment=2444328
	//
	// with such a low threshold, mnetwork should probably be the very last
	// type to check for
	threshold := 1

	for i := 0; i < len(b)-3; i++ {
		if b[i] == 0xad && (b[i+1] >= 0xe0 && b[i+1] <= 0xe7) {
			// bank switching can address any cartidge mirror so mask off
			// insignificant bytes
			//
			// (09/03/21) mask wasn't correct (0x0f selects non-cartridge
			// mirrors too) correct mask is 0x1f.
			//
			// the incorrect mask caused a false positive for Solaris when the
			// threshold is 2.
			//
			// (20/06/21) this caused a falso positive for "Hack Em Hangly Pacman"
			// when the threshold is 1
			//
			// change to only look for mirrors 0x1f and 0xff
			if b[i+2] == 0x1f || b[i+2] == 0xff {
				threshold--
				if threshold == 0 {
					return true
				}
			}
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

func fingerprintWickstead(b []byte) bool {
	// wickstead design fingerprint taken from Stella
	return bytes.Contains(b, []byte{0xa5, 0x39, 0x4c})
}

func fingerprintDPCplus(b []byte) bool {
	if len(b) < 0x23 {
		return false
	}
	return b[0x20] == 0x1e && b[0x21] == 0xab && b[0x22] == 0xad && b[0x23] == 0x10
}

func fingerprintCDFJplus(b []byte) (bool, string) {
	// all CDF formats must be 32k or 64k
	if len(b) != 32768 && len(b) != 65536 {
		return false, ""
	}

	if bytes.Contains(b[:2048], []byte("PLUSCDFJ")) {
		return true, "CDFJ+"
	}

	return false, ""
}

func fingerprintCDF(b []byte) (bool, string) {
	// all CDF formats must be 32k or 64k
	if len(b) != 32768 {
		return false, ""
	}

	count := 0
	version := ""

	for i := 0; i < len(b)-3; i++ {
		if b[i] == 'C' && b[i+1] == 'D' && b[i+2] == 'F' {
			var newVersion string
			count++

			// create version string. slightly different for CDFJ
			if b[i+3] == 'J' {
				newVersion = "CDFJ"
			} else {
				newVersion = fmt.Sprintf("CDF%1d", b[i+3])
			}

			// make sure the version number hasn't changed
			if version != "" && version != newVersion {
				return false, ""
			}
			version = newVersion
		}
	}

	return count >= 3, version
}

func fingerprintSuperchargerFastLoad(cartload cartridgeloader.Loader) bool {
	return len(*cartload.Data) > 0 && len(*cartload.Data)%8448 == 0
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

func fingerprint8k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintParkerBros(data) {
		return newParkerBros
	}

	// mnetwork has the lowest threshold so place it at the end
	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	if fingerprintWickstead(data) {
		return newWicksteadDesign
	}

	return newAtari8k
}

func fingerprint16k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	return newAtari16k
}

func fingerprint32k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	return newAtari32k
}

func fingerprint64k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	return newEF
}

func fingerprint128k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	if fingerprintDF(data) {
		return newDF
	}

	return newSuperbank
}

func fingerprint256k(data []byte) func(*instance.Instance, []byte) (mapper.CartMapper, error) {
	return newSuperbank
}

func (cart *Cartridge) fingerprint(cartload cartridgeloader.Loader) error {
	var err error

	if ok := fingerprintElf(*cartload.Data, false); ok {
		cart.mapper, err = elf.NewElf(cart.instance, cart.Filename, false)
		return err
	}

	if ok, version, wrappedElf := fingerprintAce(*cartload.Data); ok {
		if wrappedElf {
			cart.mapper, err = elf.NewElf(cart.instance, cart.Filename, true)
			return err
		}
		cart.mapper, err = ace.NewAce(cart.instance, version, *cartload.Data)
		return err
	}

	if ok, version := fingerprintCDFJplus(*cartload.Data); ok {
		cart.mapper, err = cdf.NewCDF(cart.instance, version, *cartload.Data)
		return err
	}

	if ok, version := fingerprintCDF(*cartload.Data); ok {
		cart.mapper, err = cdf.NewCDF(cart.instance, version, *cartload.Data)
		return err
	}

	if fingerprintDPCplus(*cartload.Data) {
		cart.mapper, err = dpcplus.NewDPCplus(cart.instance, *cartload.Data)
		return err
	}

	if fingerprintSuperchargerFastLoad(cartload) {
		cart.mapper, err = supercharger.NewSupercharger(cart.instance, cartload)
		return err
	}

	if fingerprint3ePlus(*cartload.Data) {
		cart.mapper, err = new3ePlus(cart.instance, *cartload.Data)
		return err
	}

	if fingerprint3e(*cartload.Data) {
		cart.mapper, err = new3e(cart.instance, *cartload.Data)
		return err
	}

	sz := len(*cartload.Data)
	switch sz {
	case 4096:
		cart.mapper, err = newAtari4k(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 8195:
		// a widely distributed bad ROM dump of the Pink Panther prototype is
		// 8195 bytes long. we'll treat it like an 8k ROM and see if it's
		// recognised as a Wickstead Design ROM. if it's not then it's just a
		// file that's 8195 bytes long and will be rejected
		fallthrough

	case 8192:
		cart.mapper, err = fingerprint8k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 10240:
		fallthrough

	case 10495:
		cart.mapper, err = newDPC(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 12288:
		cart.mapper, err = newCBS(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 16384:
		cart.mapper, err = fingerprint16k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 32768:
		cart.mapper, err = fingerprint32k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 65536:
		cart.mapper, err = fingerprint64k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 131072:
		cart.mapper, err = fingerprint128k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	case 262144:
		cart.mapper, err = fingerprint256k(*cartload.Data)(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	default:
		if sz >= 4096 {
			return fmt.Errorf("unrecognised size (%d bytes)", len(*cartload.Data))
		}

		cart.mapper, err = newAtari2k(cart.instance, *cartload.Data)
		if err != nil {
			return err
		}

	}

	// if cartridge mapper implements the optionalSuperChip interface then try
	// to add the additional RAM
	if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
		superchip.AddSuperchip(false)
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
	for i := 0; i < len(*cartload.Data)-2; i++ {
		if (*cartload.Data)[i] == 0x8d && (*cartload.Data)[i+1] == 0xf1 && ((*cartload.Data)[i+2]&0x10) == 0x10 {
			return true
		}
	}
	return false
}
